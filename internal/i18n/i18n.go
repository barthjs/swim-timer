// Package i18n provides a small, JSON-backed translation catalog for the strings the backend produces.
//
// Catalogs are embedded at build time, one JSON file per locale under locales/. Nested objects are flattened
// into dotted keys ("section.key"), and values may contain text/template placeholders ("{{.Name}}") rendered per call.
// A placeholder without a value renders as nothing, see BlankMissing.
//
// A message that depends on a number has one key per CLDR plural category of the locale, nested under the message key,
// for example {"items": {"one": "{{.Count}} item", "other": "{{ number .Count }} items"}}.
// TN picks the category for the count, English and German use "one" and "other", other languages may also need "zero",
// "two", "few", or "many". Every message can format with the functions number, date, and datetime,
// which follow the conventions of the catalog's locale.
//
// WithMissingKeyHandler reports keys a locale does not translate, for example as a warning in development.
package i18n

import (
	"bytes"
	"embed"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

// localeFS holds the embedded locale catalogs.
//
//go:embed locales
var localeFS embed.FS

// DefaultLocale is the locale every other locale falls back to.
const DefaultLocale = "en"

// entry keeps both the parsed template and its raw source.
type entry struct {
	raw  string
	tmpl *template.Template
}

// Bundle holds the parsed message catalogs, keyed by locale and dotted message key.
type Bundle struct {
	messages map[string]map[string]entry
	missing  func(locale, key string)
}

// Option configures a Bundle.
type Option func(*options)

// options holds the settings applied by the Option values.
type options struct {
	missing func(locale, key string)
}

// WithMissingKeyHandler calls handle whenever a message is requested in a locale
// that has no translation for it, also when the default locale provides the text.
// Use it to log missing translations during development. It may be called concurrently.
func WithMissingKeyHandler(handle func(locale, key string)) Option {
	return func(o *options) {
		o.missing = handle
	}
}

// New creates a Bundle from the embedded locale catalogs.
func New(opts ...Option) (*Bundle, error) {
	return NewFS(localeFS, opts...)
}

// NewFS creates a Bundle from only the locale catalogs under the "locales" directory of fsys,
// one JSON file per locale named after it, such as "locales/en.json", without the embedded catalogs.
// Tests use it to start from catalogs of their own, for example a fstest.MapFS.
func NewFS(fsys fs.FS, opts ...Option) (*Bundle, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	messages, err := readCatalogs(fsys)
	if err != nil {
		return nil, err
	}

	bundle, err := compile(messages)
	if err != nil {
		return nil, err
	}
	bundle.missing = o.missing
	return bundle, nil
}

// readCatalogs reads every catalog under "locales" into flat messages keyed by locale and dotted key.
func readCatalogs(fsys fs.FS) (map[string]map[string]string, error) {
	var files []string
	err := fs.WalkDir(fsys, "locales", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	messages := make(map[string]map[string]string, len(files))
	for _, file := range files {
		locale := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}

		var nested map[string]any
		if err := json.Unmarshal(data, &nested); err != nil {
			return nil, fmt.Errorf("parsing locale %q: %w", locale, err)
		}

		flat := make(map[string]string)
		if err := flatten("", nested, flat); err != nil {
			return nil, fmt.Errorf("locale %q: %w", locale, err)
		}
		messages[locale] = flat
	}

	return messages, nil
}

// compile parses every message as a template with the formatting functions of its locale
// and requires a catalog for the default locale.
func compile(messages map[string]map[string]string) (*Bundle, error) {
	bundle := &Bundle{
		messages: make(map[string]map[string]entry, len(messages)),
	}
	for locale, flat := range messages {
		entries := make(map[string]entry, len(flat))
		funcs := LocaleFuncs(locale)
		for key, value := range flat {
			tmpl, err := template.New(key).Funcs(TemplateFuncs()).Funcs(funcs).Parse(value)
			if err != nil {
				return nil, fmt.Errorf("locale %q key %q: %w", locale, key, err)
			}
			BlankMissing(tmpl.Tree)
			entries[key] = entry{
				raw:  value,
				tmpl: tmpl,
			}
		}

		bundle.messages[locale] = entries
	}

	if _, ok := bundle.messages[DefaultLocale]; !ok {
		return nil, fmt.Errorf("default locale %q catalog is missing", DefaultLocale)
	}

	return bundle, nil
}

// T returns the localized message for a key, rendering placeholders with the optional data argument.
// It falls back to the default locale when the key is missing for the requested locale and returns
// the key itself when no translation exists at all, so missing strings stay visible rather than silent.
func (b *Bundle) T(locale, key string, data any) string {
	found, ok := b.lookup(locale, key)
	if !ok {
		return key
	}
	return found.render(data)
}

// TN returns the localized message for a key that depends on count, choosing the CLDR plural category of the locale,
// such as key "items" with count 1 rendering "items.one" in English. It falls back to the category "other"
// and then to the default locale like T, and returns the key when nothing matches. The count is added
// to map data under CountKey, see withCount.
func (b *Bundle) TN(locale, key string, count int, data any) string {
	data = withCount(data, count)

	candidates := []struct{ locale, key string }{
		{
			locale: locale,
			key:    key + "." + pluralForm(locale, count),
		},
		{
			locale: locale,
			key:    key + ".other",
		},
		{
			locale: DefaultLocale,
			key:    key + "." + pluralForm(DefaultLocale, count),
		},
		{
			locale: DefaultLocale,
			key:    key + ".other",
		},
	}
	for i, candidate := range candidates {
		found, ok := b.messages[candidate.locale][candidate.key]
		if !ok {
			continue
		}
		if i > 0 {
			b.reportMissing(locale, candidates[0].key)
		}
		return found.render(data)
	}

	b.reportMissing(locale, candidates[0].key)
	return key
}

// render executes the message with data and returns its raw text if that fails.
func (e entry) render(data any) string {
	var buf bytes.Buffer
	if err := e.tmpl.Execute(&buf, data); err != nil {
		return e.raw
	}
	return buf.String()
}

// lookup resolves a key for a locale, falling back to the default locale,
// and reports a key the requested locale does not translate.
func (b *Bundle) lookup(locale, key string) (entry, bool) {
	if found, ok := b.messages[locale][key]; ok {
		return found, true
	}
	b.reportMissing(locale, key)

	if locale != DefaultLocale {
		if found, ok := b.messages[DefaultLocale][key]; ok {
			return found, true
		}
	}

	return entry{}, false
}

// reportMissing passes a key without translation in the locale to the missing key handler, if any.
func (b *Bundle) reportMissing(locale, key string) {
	if b.missing != nil {
		b.missing(locale, key)
	}
}

// flatten collapses a nested message map into dotted keys, rejecting any non-string leaf.
func flatten(prefix string, nested map[string]any, out map[string]string) error {
	for k, v := range nested {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch value := v.(type) {
		case string:
			out[key] = value
		case map[string]any:
			if err := flatten(key, value, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("key %q must be a string or object, got %T", key, v)
		}
	}

	return nil
}
