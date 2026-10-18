package i18n_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/i18n"
)

func TestBundle_TN(t *testing.T) {
	t.Parallel()

	bundle := pluralBundle(t)
	tests := []struct {
		name   string
		locale string
		key    string
		count  int
		data   any
		want   string
	}{
		{
			name:   "uses one in English",
			locale: "en",
			key:    "items",
			count:  1,
			want:   "1 item",
		},
		{
			name:   "uses other for zero in English",
			locale: "en",
			key:    "items",
			count:  0,
			want:   "0 items",
		},
		{
			name:   "formats the count",
			locale: "de",
			key:    "items",
			count:  1500,
			want:   "1.500 Elemente",
		},
		{
			name:   "uses one in German",
			locale: "de",
			key:    "items",
			count:  1,
			want:   "1 Element",
		},
		{
			name:   "uses the category of the amount for a negative count",
			locale: "de",
			key:    "items",
			count:  -1,
			want:   "-1 Element",
		},
		{
			name:   "uses few in Polish",
			locale: "pl",
			key:    "items",
			count:  3,
			want:   "3 elementy", //nolint:misspell // Polish plural form
		},
		{
			name:   "uses many in Polish",
			locale: "pl",
			key:    "items",
			count:  5,
			want:   "5 elementów",
		},
		{
			name:   "falls back to other",
			locale: "en",
			key:    "only_other",
			count:  1,
			want:   "1 items",
		},
		{
			name:   "falls back to the default locale",
			locale: "de",
			key:    "only_other",
			count:  2,
			want:   "2 items",
		},
		{
			name:   "falls back for an unknown locale",
			locale: "fr",
			key:    "items",
			count:  1,
			want:   "1 item",
		},
		{
			name:   "returns the key without any translation",
			locale: "de",
			key:    "missing",
			count:  1,
			want:   "missing",
		},
		{
			name:   "adds the count to map data",
			locale: "en",
			key:    "named",
			count:  2,
			data:   map[string]any{"Name": "Ada"},
			want:   "Ada took 2 items",
		},
		{
			name:   "keeps a count the data already holds",
			locale: "en",
			key:    "named",
			count:  1,
			data:   map[string]any{"Name": "Ada", "Count": "one"},
			want:   "Ada took one item",
		},
		{
			name:   "passes other data unchanged",
			locale: "en",
			key:    "named",
			count:  1,
			data: struct{ Name, Count string }{
				Name:  "Ada",
				Count: "a single",
			},
			want: "Ada took a single item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := bundle.TN(tt.locale, tt.key, tt.count, tt.data)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("keeps the map of the caller", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"Name": "Ada"}

		bundle.TN("en", "named", 3, data)

		assert.Equal(t, map[string]any{"Name": "Ada"}, data)
	})
}

// pluralBundle returns a bundle with sample plural catalogs.
func pluralBundle(t *testing.T) *i18n.Bundle {
	t.Helper()

	bundle, err := i18n.NewFS(catalogs(map[string]string{
		"en": `{"items": {"one": "{{.Count}} item", "other": "{{ number .Count }} items"}, "only_other": {"other": "{{.Count}} items"}, "named": {"one": "{{.Name}} took {{.Count}} item", "other": "{{.Name}} took {{.Count}} items"}}`,
		"de": `{"items": {"one": "{{.Count}} Element", "other": "{{ number .Count }} Elemente"}}`,
		"pl": `{"items": {"one": "{{.Count}} element", "few": "{{.Count}} elementy", "many": "{{.Count}} elementów", "other": "{{.Count}} elementu"}}`, //nolint:misspell // Polish plural forms
	}))
	require.NoError(t, err)
	return bundle
}
