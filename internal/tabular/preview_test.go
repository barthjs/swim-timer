package tabular

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Preview(t *testing.T) {
	t.Parallel()

	t.Run("returns the start of the file with a guessed mapping", func(t *testing.T) {
		t.Parallel()

		content := "Title;Kind;Note;\r\na;Small;x;\r\nb;Large;y;\r\nc;small;;\r\n"

		preview, err := New(testColumns()).Preview(strings.NewReader(content), Format{})

		require.NoError(t, err)
		assert.Equal(t, Format{
			Encoding:  EncodingWindows1252,
			Separator: ";",
		}, preview.Format)
		assert.Equal(t, []string{"Title", "Kind", "Note", ""}, preview.Header)
		assert.Equal(t, [][]string{
			{
				"a",
				"Small",
				"x",
				"",
			},
			{
				"b",
				"Large",
				"y",
				"",
			},
			{
				"c",
				"small",
				"",
				"",
			},
		}, preview.Sample)
		assert.Equal(t, map[string][]string{
			"Title": {
				"a",
				"b",
				"c",
			},
			"Kind": {
				"Small",
				"Large",
				"small",
			},
			"Note": {
				"x",
				"y",
			},
		}, preview.Values)
		assert.Equal(t, Mapping{
			Format: preview.Format,
			Columns: map[string]string{
				"name": "Title",
				"kind": "Kind",
			},
			Values: map[string]map[string]string{
				"kind": {
					"Small": "small",
					"Large": "large",
					"small": "small",
				},
			},
		}, preview.Mapping)
	})

	t.Run("shows only the first rows", func(t *testing.T) {
		t.Parallel()

		var content strings.Builder
		_, _ = content.WriteString("name\r\n")
		for i := range previewSampleRows + 3 {
			_, _ = fmt.Fprintf(&content, "%d\r\n", i)
		}

		preview, err := New(testColumns()).Preview(strings.NewReader(content.String()), Format{})

		require.NoError(t, err)
		assert.Len(t, preview.Sample, previewSampleRows)
	})

	t.Run("leaves out the values of a column with many of them", func(t *testing.T) {
		t.Parallel()

		var content strings.Builder
		_, _ = content.WriteString("name;kind\r\n")
		for i := range previewMaxValues + 1 {
			_, _ = fmt.Fprintf(&content, "%d;small\r\n", i)
		}

		preview, err := New(testColumns()).Preview(strings.NewReader(content.String()), Format{})

		require.NoError(t, err)
		assert.NotContains(t, preview.Values, "name")
		assert.Equal(t, []string{"small"}, preview.Values["kind"])
	})

	t.Run("skips a row that ends early", func(t *testing.T) {
		t.Parallel()

		preview, err := New(testColumns()).Preview(strings.NewReader("name;kind\r\na\r\nb;small\r\n"), Format{})

		require.NoError(t, err)
		assert.Len(t, preview.Sample, 1)
	})

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		_, err := New(testColumns()).Preview(strings.NewReader(""), Format{})

		_, ok := errors.AsType[*FileError](err)
		assert.True(t, ok)
	})
}

func TestCollectValues(t *testing.T) {
	t.Parallel()

	t.Run("adds new values and ignores empty and known ones", func(t *testing.T) {
		t.Parallel()

		distinct := [][]string{
			{
				"a",
			},
			nil,
		}
		full := []bool{false, false}

		collectValues([]string{"a", "", "extra"}, distinct, full)
		collectValues([]string{"b", "c"}, distinct, full)

		assert.Equal(t, [][]string{
			{
				"a",
				"b",
			},
			{
				"c",
			},
		}, distinct)
	})
}
