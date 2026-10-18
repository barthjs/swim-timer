package i18n_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/i18n"
)

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		value  any
		want   string
	}{
		{
			name:   "groups German digits",
			locale: "de",
			value:  1234567,
			want:   "1.234.567",
		},
		{
			name:   "uses the German decimal comma",
			locale: "de",
			value:  1234.5,
			want:   "1.234,5",
		},
		{
			name:   "groups English digits",
			locale: "en",
			value:  1234567,
			want:   "1,234,567",
		},
		{
			name:   "uses the English decimal point",
			locale: "en",
			value:  1234.5,
			want:   "1,234.5",
		},
		{
			name:   "follows a regional locale",
			locale: "de-AT",
			value:  1234.5,
			want:   "1\u00a0234,5",
		},
		{
			name:   "falls back to the default locale for an invalid one",
			locale: "not a locale",
			value:  1234.5,
			want:   "1,234.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := i18n.FormatNumber(tt.locale, tt.value)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDate(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 5, 1, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{
			name:   "formats a German date",
			locale: "de",
			want:   "01.05.2026",
		},
		{
			name:   "formats the date of a regional locale like its language",
			locale: "de-CH",
			want:   "01.05.2026",
		},
		{
			name:   "formats an English date",
			locale: "en",
			want:   "May 1, 2026",
		},
		{
			name:   "formats the date as ISO for a locale without own pattern",
			locale: "fr",
			want:   "2026-05-01",
		},
		{
			name:   "falls back to the default locale for an invalid one",
			locale: "not a locale",
			want:   "May 1, 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := i18n.FormatDate(tt.locale, at)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 5, 1, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{
			name:   "formats a German date and time",
			locale: "de",
			want:   "01.05.2026, 14:30",
		},
		{
			name:   "formats the date and time of a regional locale like its language",
			locale: "de-CH",
			want:   "01.05.2026, 14:30",
		},
		{
			name:   "formats an English date and time",
			locale: "en",
			want:   "May 1, 2026, 2:30 PM",
		},
		{
			name:   "formats the date and time as ISO for a locale without own pattern",
			locale: "fr",
			want:   "2026-05-01 14:30",
		},
		{
			name:   "falls back to the default locale for an invalid one",
			locale: "not a locale",
			want:   "May 1, 2026, 2:30 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := i18n.FormatDateTime(tt.locale, at)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("keeps the location of the time", func(t *testing.T) {
		t.Parallel()

		berlin, err := time.LoadLocation("Europe/Berlin")
		require.NoError(t, err)
		at := time.Date(2026, 5, 1, 23, 30, 0, 0, time.UTC).In(berlin)

		got := i18n.FormatDateTime("de", at)

		assert.Equal(t, "02.05.2026, 01:30", got)
	})
}

func TestLocaleFuncs(t *testing.T) {
	t.Parallel()

	t.Run("formats the messages of a catalog for their locale", func(t *testing.T) {
		t.Parallel()

		bundle, err := i18n.NewFS(catalogs(map[string]string{
			"en": `{"stats": "{{ number .Total }} items since {{ date .Since }} ({{ datetime .Since }})"}`,
			"de": `{"stats": "{{ number .Total }} Elemente seit {{ date .Since }} ({{ datetime .Since }})"}`,
		}))
		require.NoError(t, err)
		data := map[string]any{"Total": 12345.5, "Since": time.Date(2026, 5, 1, 9, 5, 0, 0, time.UTC)}

		german := bundle.T("de", "stats", data)
		english := bundle.T("en", "stats", data)

		assert.Equal(t, "12.345,5 Elemente seit 01.05.2026 (01.05.2026, 09:05)", german)
		assert.Equal(t, "12,345.5 items since May 1, 2026 (May 1, 2026, 9:05 AM)", english)
	})
}
