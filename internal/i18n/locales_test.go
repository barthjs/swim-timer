package i18n_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/i18n"
)

// placeholder matches a template field such as "{{.Name}}" or "{{ .Name }}".
var placeholder = regexp.MustCompile(`{{\s*\.(\w+)\s*}}`)

// TestLocales checks the catalogs: every key resolves, every locale has the same keys as the default locale,
// and every translation uses the same placeholders, so no text falls back or loses data.
func TestLocales(t *testing.T) {
	t.Parallel()

	t.Run("resolves every key", func(t *testing.T) {
		t.Parallel()

		bundle, err := i18n.New()
		require.NoError(t, err)

		for locale, catalog := range readCatalogs(t) {
			for key := range catalog {
				assert.NotEqual(t, key, bundle.T(locale, key, nil), "locale %q key %q resolves", locale, key)
			}
		}
	})

	all := readCatalogs(t)
	defaults, ok := all[i18n.DefaultLocale]
	require.True(t, ok, "the default locale has a catalog")

	for locale, catalog := range all {
		if locale == i18n.DefaultLocale {
			continue
		}

		t.Run("keeps the keys and placeholders of the default locale in "+locale, func(t *testing.T) {
			t.Parallel()

			for key, value := range defaults {
				translated, ok := catalog[key]
				if !assert.True(t, ok, "key %q is missing", key) {
					continue
				}
				assert.Equal(t, placeholders(value), placeholders(translated), "placeholders of %q", key)
			}
			for key := range catalog {
				_, ok := defaults[key]
				assert.True(t, ok, "key %q is missing in the default locale", key)
			}
		})
	}
}

// placeholders returns the sorted, unique placeholder names in text.
func placeholders(text string) []string {
	var names []string
	for _, match := range placeholder.FindAllStringSubmatch(text, -1) {
		names = append(names, match[1])
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// readCatalogs reads the embedded catalogs from disk and flattens their keys.
func readCatalogs(t *testing.T) map[string]map[string]string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("locales", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	result := make(map[string]map[string]string, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)

		var nested map[string]any
		require.NoError(t, json.Unmarshal(data, &nested), file)

		flat := make(map[string]string)
		flattenForTest("", nested, flat)
		result[strings.TrimSuffix(filepath.Base(file), ".json")] = flat
	}
	return result
}

// flattenForTest flattens the nested catalog into out with dotted keys.
func flattenForTest(prefix string, nested map[string]any, out map[string]string) {
	for key, value := range nested {
		if prefix != "" {
			key = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]any:
			flattenForTest(key, v, out)
		case string:
			out[key] = v
		}
	}
}
