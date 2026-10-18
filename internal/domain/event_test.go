package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestEvent_DisplayName(t *testing.T) {
	t.Parallel()

	t.Run("returns the distance, style, and category", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		distance := faker.LetterN(6)
		style := faker.LetterN(6)
		category := faker.LetterN(6)
		event := Event{
			Distance: &Distance{
				Name: distance,
			},
			Style: &Style{
				Name: style,
			},
			Category: &Category{
				Name: category,
			},
		}

		name := event.DisplayName()

		assert.Equal(t, distance+" "+style+" "+category, name)
	})

	t.Run("leaves no double space for a missing reference", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		distance := faker.LetterN(6)
		category := faker.LetterN(6)
		event := Event{
			Distance: &Distance{
				Name: distance,
			},
			Category: &Category{
				Name: category,
			},
		}

		name := event.DisplayName()

		assert.Equal(t, distance+" "+category, name)
	})

	t.Run("returns nothing without references", func(t *testing.T) {
		t.Parallel()

		name := (&Event{}).DisplayName()

		assert.Empty(t, name)
	})
}

func TestEvent_PhaseName(t *testing.T) {
	t.Parallel()

	t.Run("returns the trimmed phase name", func(t *testing.T) {
		t.Parallel()

		want := apitest.Faker(t).Noun()
		event := Event{
			Phase: &Phase{
				Name: " " + want + " ",
			},
		}

		name := event.PhaseName()

		assert.Equal(t, want, name)
	})

	t.Run("returns nothing without the phases", func(t *testing.T) {
		t.Parallel()

		name := (&Event{}).PhaseName()

		assert.Empty(t, name)
	})
}

func TestEvent_TrimmedTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		time string
		want string
	}{
		{
			name: "returns the trimmed scheduled time",
			time: " 16:00 ",
			want: "16:00",
		},
		{
			name: "returns nothing without a scheduled time",
			time: "00:00",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := Event{
				Time: tt.time,
			}

			trimmed := event.TrimmedTime()

			assert.Equal(t, tt.want, trimmed)
		})
	}
}

func TestParseExportDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "returns the iso date",
			value: "26/05/01",
			want:  "2026-05-01",
		},
		{
			name:  "trims the value",
			value: " 26/12/31 ",
			want:  "2026-12-31",
		},
		{
			name:  "returns nothing for an empty value",
			value: "",
			want:  "",
		},
		{
			name:  "keeps another format",
			value: "01.05.2026",
			want:  "01.05.2026",
		},
		{
			name:  "keeps an invalid date",
			value: "26/13/01",
			want:  "26/13/01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			date := ParseExportDate(tt.value)

			assert.Equal(t, tt.want, date)
		})
	}
}
