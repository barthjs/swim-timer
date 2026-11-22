package tabular

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestParseString(t *testing.T) {
	t.Parallel()

	t.Run("returns the text as it is", func(t *testing.T) {
		t.Parallel()

		text := testName(t)

		value, err := ParseString(text)

		require.NoError(t, err)
		assert.Equal(t, text, value)
	})
}

func TestParseInt(t *testing.T) {
	t.Parallel()

	t.Run("parses a whole number", func(t *testing.T) {
		t.Parallel()

		value, err := ParseInt("-42")

		require.NoError(t, err)
		assert.Equal(t, -42, value)
	})

	t.Run("rejects a decimal number", func(t *testing.T) {
		t.Parallel()

		_, err := ParseInt("1.5")

		assert.Equal(t, &FieldError{
			Tag: "number",
		}, err)
	})
}

func TestParseDecimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want float64
	}{
		{
			name: "parses a decimal point",
			raw:  "1.25",
			want: 1.25,
		},
		{
			name: "parses a decimal comma",
			raw:  "1,25",
			want: 1.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, err := ParseDecimal(tt.raw)

			require.NoError(t, err)
			assert.InDelta(t, tt.want, value, 1e-9)
		})
	}

	t.Run("rejects text", func(t *testing.T) {
		t.Parallel()

		_, err := ParseDecimal("x")

		assert.Equal(t, &FieldError{
			Tag: "number",
		}, err)
	})
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	t.Run("parses true and false", func(t *testing.T) {
		t.Parallel()

		yes, errYes := ParseBool("true")
		no, errNo := ParseBool("false")

		require.NoError(t, errYes)
		require.NoError(t, errNo)
		assert.True(t, yes)
		assert.False(t, no)
	})

	t.Run("rejects another text", func(t *testing.T) {
		t.Parallel()

		_, err := ParseBool("1")

		assert.Equal(t, &FieldError{
			Tag: "boolean",
		}, err)
	})
}

func TestFormatBool(t *testing.T) {
	t.Parallel()

	t.Run("returns the text ParseBool reads", func(t *testing.T) {
		t.Parallel()

		value, err := ParseBool(FormatBool(true))

		require.NoError(t, err)
		assert.True(t, value)
	})
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	t.Run("parses the first layout that fits", func(t *testing.T) {
		t.Parallel()

		parse := ParseDate(time.DateOnly, "02.01.2006")
		date := apitest.Faker(t).Date().UTC().Truncate(24 * time.Hour)

		value, err := parse(date.Format("02.01.2006"))

		require.NoError(t, err)
		assert.Equal(t, date, value)
	})

	t.Run("rejects a date in no layout with the first one", func(t *testing.T) {
		t.Parallel()

		parse := ParseDate(time.DateOnly, "02.01.2006")

		_, err := parse(apitest.Faker(t).Date().Format("2006/01/02"))

		assert.Equal(t, &FieldError{
			Tag:   "date",
			Param: time.DateOnly,
		}, err)
	})
}
