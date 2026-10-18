package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileName(t *testing.T) {
	t.Parallel()

	t.Run("returns the name of the item", func(t *testing.T) {
		t.Parallel()

		name, ok := fileName("items", "items/start.v2.json", ".json")

		require.True(t, ok)
		assert.Equal(t, "start.v2", name)
	})

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			desc string
			path string
		}{
			{
				desc: "file in another folder",
				path: "other/start.json",
			},
			{
				desc: "file in a subfolder",
				path: "items/nested/start.json",
			},
			{
				desc: "file with another extension",
				path: "items/notes.txt",
			},
			{
				desc: "file without a name",
				path: "items/.json",
			},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()

				_, ok := fileName("items", tt.path, ".json")

				assert.False(t, ok)
			})
		}
	})
}

func TestFilePath(t *testing.T) {
	t.Parallel()

	t.Run("returns the file of the item", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			desc string
			name string
			want string
		}{
			{
				desc: "keeps the spaces of the name",
				name: "name with spaces",
				want: "items/name with spaces.json",
			},
			{
				desc: "keeps the dots of the name",
				name: "start.v2",
				want: "items/start.v2.json",
			},
			{
				desc: "accepts multibyte characters up to the limit",
				name: strings.Repeat("ä", (maxFileNameBytes-len(".json"))/2),
				want: "items/" + strings.Repeat("ä", (maxFileNameBytes-len(".json"))/2) + ".json",
			},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()

				got, err := filePath("items", tt.name, ".json")

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		for _, name := range invalidFileNames(".json") {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := filePath("items", name, ".json")

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})
}

// invalidFileNames returns names that would leave the folder of a store
// or whose file name with the extension would exceed the file name limit.
func invalidFileNames(extension string) []string {
	tooLong := strings.Repeat("a", maxFileNameBytes-len(extension)+1)
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, "/absolute", tooLong}
}
