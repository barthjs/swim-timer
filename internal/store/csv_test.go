package store

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/media"
)

// csvRow is a row type with the column layout of a small export file.
type csvRow struct {
	ID   int    `csv:"id"`
	Name string `csv:"name"`
}

func TestReadMeetFile(t *testing.T) {
	t.Parallel()

	t.Run("returns the content as it is stored", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		content := []byte{'i', 'd', 0xE4, '\r', '\n'}
		dir := filepath.Join(root, meetFolder)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "TEST.TXT"), content, 0o644))

		got, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")

		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("fails for a missing file", func(t *testing.T) {
		t.Parallel()

		storage, _ := newTestStorageWithRoot(t)

		_, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")

		assert.Error(t, err)
	})
}

func TestLoadMap(t *testing.T) {
	t.Parallel()

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "TEST.TXT", "id;name\r\n1;x;\r\n2\r\n")

		err := loadMap(readTestFile(t, storage), map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, 3, parseErr.Line)
	})

	t.Run("rejects a duplicate key with its line in the file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "TEST.TXT", "id;name\r\n1;x;\r\n\r\n2;y;\r\n1;z;\r\n")

		err := loadMap(readTestFile(t, storage), map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, 5, parseErr.Line, "the line in the file, counting the skipped blank row")
		assert.ErrorContains(t, err, "line 5")
	})
}

func TestReadCSV(t *testing.T) {
	t.Parallel()

	t.Run("returns the rows", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(8)

		rows, err := readTestCSV(t, "id;name\r\n1;\""+name+"\";\r\n \r\n;;\r\n2;\"\";\r\n")

		require.NoError(t, err)
		require.Len(t, rows, 2, "rows without any value are skipped")
		assert.Equal(t, name, rows[0].Name)
		assert.Equal(t, 2, rows[1].ID)
	})

	t.Run("reads the special characters of windows-1252", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(4) + "äß"

		rows, err := readTestCSV(t, "id;name\r\n1;\""+name+"\";\r\n")

		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, name, rows[0].Name)
	})

	t.Run("reads a header with a trailing separator", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(8)

		rows, err := readTestCSV(t, "id;name;\r\n1;\""+name+"\";\r\n2;x\r\n")

		require.NoError(t, err)
		require.Len(t, rows, 2, "a row without the trailing separator holds every named column")
		assert.Equal(t, name, rows[0].Name)
		assert.Equal(t, 2, rows[1].ID)
	})

	t.Run("reads a stray quote inside a value", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(4)

		rows, err := readTestCSV(t, "id;name\r\n1;\""+name+"\"x\";\r\n2;"+name+"\"x;\r\n")

		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, name+"\"x", rows[0].Name)
		assert.Equal(t, name+"\"x", rows[1].Name)
	})

	invalid := []struct {
		name     string
		content  string
		wantLine int
		wantErr  string
	}{
		{
			name:     "rejects a missing column",
			content:  "id;title\r\n1;x\r\n",
			wantLine: 1,
			wantErr:  "lacks the columns name",
		},
		{
			name:     "rejects a cut off row",
			content:  "id;name\r\n1;x;\r\n2\r\n",
			wantLine: 3,
			wantErr:  "1 of 2 values",
		},
		{
			name:     "rejects a quote that is not closed",
			content:  "id;name\r\n1;x;\r\n2;\"y;\r\n3;z;\r\n4;w;\r\n",
			wantLine: 3,
			wantErr:  "column name: a quote is not closed",
		},
		{
			name:     "rejects a quote that is not closed in the last row",
			content:  "id;name\r\n1;x;\r\n2;\"y;\r\n",
			wantLine: 3,
			wantErr:  "a quote is not closed",
		},
		{
			name:     "rejects a quote that is not closed at the end of the file",
			content:  "id;name\r\n1;x;\r\n2;\"y",
			wantLine: 3,
			wantErr:  "column name: a quote is not closed",
		},
		{
			name:     "rejects an invalid number",
			content:  "id;name\r\n1;x\r\n\r\nx;y\r\n",
			wantLine: 4,
			wantErr:  "invalid syntax",
		},
		{
			name:    "rejects an empty file",
			content: "",
			wantErr: "empty",
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := readTestCSV(t, tt.content)

			parseErr, ok := errors.AsType[*ParseError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantLine, parseErr.Line)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	marks := []struct {
		name string
		mark []byte
	}{
		{
			name: "rejects a utf-8 byte order mark",
			mark: bomUTF8,
		},
		{
			name: "rejects a utf-16be byte order mark",
			mark: bomUTF16BE,
		},
		{
			name: "rejects a utf-16le byte order mark",
			mark: bomUTF16LE,
		},
	}

	for _, tt := range marks {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := append(slices.Clone(tt.mark), []byte("id;name\r\n1;x\r\n")...)

			_, err := readRawTestCSV(t, content)

			_, ok := errors.AsType[*ParseError](err)
			require.True(t, ok)
			assert.ErrorContains(t, err, "Windows-1252")
		})
	}

	t.Run("rejects utf-8", func(t *testing.T) {
		t.Parallel()

		content := "id;name\r\n1;\"" + apitest.Faker(t).LetterN(4) + "ä\";\r\n"

		_, err := readRawTestCSV(t, []byte(content))

		_, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.ErrorContains(t, err, "UTF-8")
	})
}

// readTestCSV writes content as a Windows-1252 meet file and reads it into rows.
func readTestCSV(t *testing.T, content string) ([]*csvRow, error) {
	t.Helper()

	storage, root := newTestStorageWithRoot(t)
	writeMeetFile(t, root, "TEST.TXT", content)

	var rows []*csvRow
	_, err := readCSV(readTestFile(t, storage), &rows)
	return rows, err
}

// readRawTestCSV writes content unencoded as a meet file and reads it into rows.
func readRawTestCSV(t *testing.T, content []byte) ([]*csvRow, error) {
	t.Helper()

	storage, root := newTestStorageWithRoot(t)
	dir := filepath.Join(root, meetFolder)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "TEST.TXT"), content, 0o644))

	var rows []*csvRow
	_, err := readCSV(readTestFile(t, storage), &rows)
	return rows, err
}

// readTestFile reads the test meet file from the storage.
func readTestFile(t *testing.T, storage media.Storage) []byte {
	t.Helper()

	content, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")
	require.NoError(t, err)
	return content
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	word := apitest.Faker(t).LetterN(8)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes the quotes and the padding inside",
			in:   `"` + word + `     "`,
			want: word,
		},
		{
			name: "removes the spaces around the quotes",
			in:   ` "FR" `,
			want: "FR",
		},
		{
			name: "trims an unquoted value",
			in:   `  100 `,
			want: "100",
		},
		{
			name: "returns nothing for empty quotes",
			in:   `""`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			normalized := normalize(tt.in)

			assert.Equal(t, tt.want, normalized)
		})
	}
}

func TestFillMap(t *testing.T) {
	t.Parallel()

	t.Run("rejects a duplicate key", func(t *testing.T) {
		t.Parallel()

		rows := []*csvRow{
			{
				ID: 1,
			},
			{
				ID: 1,
			},
		}

		err := fillMap(rows, []int{2, 5}, map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, 5, parseErr.Line, "the line of the second occurrence")
		assert.ErrorContains(t, err, "the key 1 occurs more than once")
	})
}
