package i18n_test

import (
	"slices"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/i18n"
)

// missingRecorder collects the keys a bundle reports as missing.
type missingRecorder struct {
	mu   sync.Mutex
	keys []string
}

func TestWithMissingKeyHandler(t *testing.T) {
	t.Parallel()

	files := catalogs(map[string]string{
		"en": `{"greeting": "Hello", "only_en": "English only", "items": {"one": "{{.Count}} item", "other": "{{.Count}} items"}}`,
		"de": `{"greeting": "Hallo", "items": {"other": "{{.Count}} Elemente"}}`,
	})

	tests := []struct {
		name     string
		call     func(*i18n.Bundle) string
		want     string
		reported []string
	}{
		{
			name: "reports nothing for a translated key",
			call: func(b *i18n.Bundle) string { return b.T("de", "greeting", nil) },
			want: "Hallo",
		},
		{
			name:     "reports a key the default locale fills in",
			call:     func(b *i18n.Bundle) string { return b.T("de", "only_en", nil) },
			want:     "English only",
			reported: []string{"de:only_en"},
		},
		{
			name:     "reports a key without any translation",
			call:     func(b *i18n.Bundle) string { return b.T("de", "missing", nil) },
			want:     "missing",
			reported: []string{"de:missing"},
		},
		{
			name:     "reports a missing plural category",
			call:     func(b *i18n.Bundle) string { return b.TN("de", "items", 1, nil) },
			want:     "1 Elemente",
			reported: []string{"de:items.one"},
		},
		{
			name:     "reports a missing plural message",
			call:     func(b *i18n.Bundle) string { return b.TN("de", "absent", 1, nil) },
			want:     "absent",
			reported: []string{"de:absent.one"},
		},
		{
			name: "reports nothing for a translated plural",
			call: func(b *i18n.Bundle) string { return b.TN("en", "items", 1, nil) },
			want: "1 item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := &missingRecorder{}
			bundle, err := i18n.NewFS(files, i18n.WithMissingKeyHandler(recorder.record))
			require.NoError(t, err)

			got := tt.call(bundle)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.reported, recorder.reported())
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("loads the embedded catalogs", func(t *testing.T) {
		t.Parallel()

		bundle, err := i18n.New()

		require.NoError(t, err)
		assert.NotNil(t, bundle)
	})

	t.Run("applies the missing key handler", func(t *testing.T) {
		t.Parallel()

		recorder := &missingRecorder{}
		bundle, err := i18n.New(i18n.WithMissingKeyHandler(recorder.record))
		require.NoError(t, err)

		bundle.T("en", "no.such.key", nil)

		assert.Equal(t, []string{"en:no.such.key"}, recorder.reported())
	})
}

// record stores a missing key for the locale.
func (r *missingRecorder) record(locale, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys = append(r.keys, locale+":"+key)
}

// reported returns the recorded missing keys.
func (r *missingRecorder) reported() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.keys)
}

func TestNewFS(t *testing.T) {
	t.Parallel()

	t.Run("ignores other files", func(t *testing.T) {
		t.Parallel()

		fsys := catalogs(map[string]string{"en": `{"a": "b"}`})
		fsys["locales/README.md"] = &fstest.MapFile{
			Data: []byte("not a catalog"),
		}

		bundle, err := i18n.NewFS(fsys)

		require.NoError(t, err)
		assert.Equal(t, "b", bundle.T("en", "a", nil))
	})

	tests := []struct {
		name    string
		fsys    fstest.MapFS
		wantErr string
	}{
		{
			name:    "rejects a missing default locale",
			fsys:    catalogs(map[string]string{"de": `{"a": "b"}`}),
			wantErr: `default locale "en" catalog is missing`,
		},
		{
			name:    "rejects invalid json",
			fsys:    catalogs(map[string]string{"en": `{`}),
			wantErr: `parsing locale "en"`,
		},
		{
			name:    "rejects a leaf that is no string",
			fsys:    catalogs(map[string]string{"en": `{"a": {"b": 1}}`}),
			wantErr: `key "a.b" must be a string or object`,
		},
		{
			name:    "rejects an invalid placeholder",
			fsys:    catalogs(map[string]string{"en": `{"a": "{{.Name"}`}),
			wantErr: `locale "en" key "a"`,
		},
		{
			name:    "rejects a missing locales directory",
			fsys:    fstest.MapFS{},
			wantErr: "locales",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := i18n.NewFS(tt.fsys)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("rejects an invalid catalog with options", func(t *testing.T) {
		t.Parallel()

		files := catalogs(map[string]string{"en": `{"broken": "{{"}`})

		_, err := i18n.NewFS(files, i18n.WithMissingKeyHandler(func(string, string) {}))

		require.Error(t, err)
	})
}

func TestBundle_T(t *testing.T) {
	t.Parallel()

	bundle := newBundle(t)
	tests := []struct {
		name   string
		locale string
		key    string
		data   any
		want   string
	}{
		{
			name:   "renders placeholders",
			locale: "de",
			key:    "greeting",
			data:   map[string]string{"Name": "Ada"},
			want:   "Hallo Ada",
		},
		{
			name:   "flattens nested keys",
			locale: "de",
			key:    "nested.deep.key",
			want:   "Tief",
		},
		{
			name:   "falls back to the default locale",
			locale: "de",
			key:    "only_en",
			want:   "English only",
		},
		{
			name:   "falls back for an unknown locale",
			locale: "fr",
			key:    "greeting",
			data:   map[string]string{"Name": "Ada"},
			want:   "Hello Ada",
		},
		{
			name:   "returns the key when no translation exists",
			locale: "de",
			key:    "missing.key",
			want:   "missing.key",
		},
		{
			name:   "returns the raw text when rendering fails",
			locale: "en",
			key:    "broken",
			data:   map[string]string{"Name": "Ada"},
			want:   "Value {{.Name.Missing}}",
		},
		{
			name:   "renders missing data as nothing",
			locale: "en",
			key:    "greeting",
			want:   "Hello ",
		},
		{
			name:   "renders a missing key of a map as nothing",
			locale: "en",
			key:    "greeting",
			data:   map[string]any{"Other": 1},
			want:   "Hello ",
		},
		{
			name:   "renders a missing nested field as nothing",
			locale: "en",
			key:    "nested_field",
			data:   map[string]any{},
			want:   "Value []",
		},
		{
			name:   "renders a nil pointer as nothing",
			locale: "en",
			key:    "greeting",
			data:   struct{ Name *string }{},
			want:   "Hello ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := bundle.T(tt.locale, tt.key, tt.data)

			assert.Equal(t, tt.want, got)
		})
	}
}

// newBundle returns a bundle with sample English and German catalogs.
func newBundle(t *testing.T) *i18n.Bundle {
	t.Helper()

	bundle, err := i18n.NewFS(catalogs(map[string]string{
		"en": `{"greeting": "Hello {{.Name}}", "nested": {"deep": {"key": "Deep"}}, "only_en": "English only", "broken": "Value {{.Name.Missing}}", "nested_field": "Value [{{.User.Name}}]"}`,
		"de": `{"greeting": "Hallo {{.Name}}", "nested": {"deep": {"key": "Tief"}}}`,
	}))
	require.NoError(t, err)
	return bundle
}

// catalogs builds a file system with one catalog per locale, as the embedded locales directory holds them.
func catalogs(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for locale, content := range files {
		fsys["locales/"+locale+".json"] = &fstest.MapFile{
			Data: []byte(content),
		}
	}
	return fsys
}
