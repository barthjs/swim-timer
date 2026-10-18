package cmdline

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestPrompter_Line(t *testing.T) {
	t.Parallel()

	t.Run("reads the lines until the end of the input", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		wantFirst := faker.LetterN(8)
		wantSecond := faker.LetterN(8)
		var out bytes.Buffer
		p := newPrompter(strings.NewReader(wantFirst+"\r\n"+wantSecond), &out)

		first, err := p.Line("Name: ")

		require.NoError(t, err)
		assert.Equal(t, wantFirst, first)

		second, err := p.Line("Name: ")

		require.NoError(t, err)
		assert.Equal(t, wantSecond, second, "input without a final line break still counts")

		_, err = p.Line("Name: ")

		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, "Name: Name: Name: ", out.String(), "labels go to the error output")
	})
}

func TestPrompter_Secret(t *testing.T) {
	t.Parallel()

	t.Run("reads the secret from a pipe", func(t *testing.T) {
		t.Parallel()

		want := apitest.Faker(t).LetterN(12)
		p := newPrompter(strings.NewReader(want+"\n"), io.Discard)

		secret, err := p.Secret("Password: ")

		require.NoError(t, err)
		assert.Equal(t, want, secret)
	})
}

func TestPrompter_NewPassword(t *testing.T) {
	t.Parallel()

	t.Run("skips the confirmation for a pipe", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		first := faker.LetterN(12)
		second := faker.LetterN(12)
		p := newPrompter(strings.NewReader(first+"\n"+second+"\n"), io.Discard)

		password, err := p.NewPassword("Password: ", "Confirm: ")

		require.NoError(t, err)
		assert.Equal(t, first, password)
	})
}

func TestPrompter_Confirm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "accepts a short yes",
			input: "y\n",
			want:  true,
		},
		{
			name:  "accepts an uppercase yes",
			input: "YES\n",
			want:  true,
		},
		{
			name:  "accepts a yes with surrounding spaces",
			input: " yes \n",
			want:  true,
		},
		{
			name:  "declines a no",
			input: "n\n",
			want:  false,
		},
		{
			name:  "declines an empty answer",
			input: "\n",
			want:  false,
		},
		{
			name:  "declines any other answer",
			input: "maybe\n",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			ok, err := newPrompter(strings.NewReader(tt.input), &out).Confirm("Delete?")

			require.NoError(t, err)
			assert.Equal(t, tt.want, ok)
			assert.Equal(t, "Delete? [y/N] ", out.String())
		})
	}

	t.Run("returns the read error", func(t *testing.T) {
		t.Parallel()

		_, err := newPrompter(failingReader{}, io.Discard).Confirm("Delete?")

		assert.EqualError(t, err, "read failed")
	})
}

// failingReader fails every read.
type failingReader struct{}

// Read returns an error without reading.
func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
