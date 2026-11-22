package tabular

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteRows(t *testing.T) {
	t.Parallel()

	t.Run("writes the rows under the header", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		var out bytes.Buffer

		err := WriteRows(&out, []string{"title", "size"}, slices.Values([][]string{
			{
				name,
				"S",
			},
		}), Format{
			Separator: ";",
		})

		require.NoError(t, err)
		assert.Equal(t, "title;size\r\n"+name+";S\r\n", out.String())
	})

	t.Run("writes the rejected rows of an import back in the format of its file", func(t *testing.T) {
		t.Parallel()

		imp, err := New(testColumns()).Read(strings.NewReader("name;count\r\nok;1\r\nbad;x\r\n"), Mapping{
			Columns: map[string]string{
				"name":  "name",
				"count": "count",
			},
		})
		require.NoError(t, err)
		_, failed, err := Collect(imp.Rows(t.Context()), 10)
		require.NoError(t, err)
		var out bytes.Buffer

		err = WriteRows(&out, imp.Header(), func(yield func([]string) bool) {
			for _, rowErr := range failed {
				if !yield(rowErr.Values) {
					return
				}
			}
		}, imp.Format())

		require.NoError(t, err)
		assert.Equal(t, "name;count\r\nbad;x\r\n", out.String())
	})

	t.Run("writes windows-1252 with a question mark for a missing character", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := WriteRows(&out, []string{"Größe"}, slices.Values([][]string{
			{
				"Grün ✓",
			},
		}), Format{
			Encoding: EncodingWindows1252,
		})

		require.NoError(t, err)
		assert.Equal(t, encodeWindows1252(t, "Größe\r\nGrün ?\r\n"), out.Bytes())
	})

	t.Run("rejects an unsupported separator", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := WriteRows(&out, []string{"name"}, slices.Values([][]string{}), Format{
			Separator: "|",
		})

		assert.ErrorContains(t, err, "unsupported separator")
	})
}

func TestSchema_Write(t *testing.T) {
	t.Parallel()

	t.Run("writes every exported column under a header of their keys", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		var out bytes.Buffer

		err := New(testColumns()).Write(&out, slices.Values([]*testItem{
			{
				Name:  name,
				Count: 3,
				Kind:  "small",
			},
		}), Format{})

		require.NoError(t, err)
		assert.Equal(t, "name,count,kind\r\n"+name+",3,small\r\n", out.String())
	})

	t.Run("writes the selected columns in their order with the given separator and a byte order mark", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := New(testColumns()).Write(&out, slices.Values([]*testItem{
			{
				Name:  "a;b",
				Count: 1,
			},
		}), Format{
			Separator: ";",
		}, WithColumns("count", "name"), WithByteOrderMark())

		require.NoError(t, err)
		assert.Equal(t, string(bomUTF8)+"count;name\r\n1;\"a;b\"\r\n", out.String())
	})

	t.Run("names the columns in the header with the given function", func(t *testing.T) {
		t.Parallel()

		name := testName(t)
		var out bytes.Buffer

		err := New(testColumns()).Write(&out, slices.Values([]*testItem{
			{
				Name:  name,
				Count: 2,
			},
		}), Format{}, WithColumns("name", "count"), WithHeader(strings.ToUpper))

		require.NoError(t, err)
		assert.Equal(t, "NAME,COUNT\r\n"+name+",2\r\n", out.String())
	})

	t.Run("escapes the values a spreadsheet would read as a formula", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := New(testColumns()).Write(&out, slices.Values([]*testItem{
			{
				Name: "=1+2",
				Kind: "+x",
			},
			{
				Name: "-x",
				Kind: "@x",
			},
			{
				Name: "a=b",
				Kind: "",
			},
		}), Format{}, WithColumns("name", "kind"), WithFormulaEscape())

		require.NoError(t, err)
		assert.Equal(t, "name,kind\r\n'=1+2,'+x\r\n'-x,'@x\r\na=b,\r\n", out.String())
	})

	t.Run("writes a formula as it is without the escape", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		err := New(testColumns()).Write(&out, slices.Values([]*testItem{
			{
				Name: "=1+2",
			},
		}), Format{}, WithColumns("name"))

		require.NoError(t, err)
		assert.Equal(t, "name\r\n=1+2\r\n", out.String())
	})

	t.Run("reads back with the identity mapping", func(t *testing.T) {
		t.Parallel()

		schema := New(testColumns()[:3])
		item := &testItem{
			Name:  testName(t),
			Count: 5,
			Kind:  "large",
		}
		var out bytes.Buffer
		require.NoError(t, schema.Write(&out, slices.Values([]*testItem{item}), Format{}))

		imp, err := schema.Read(strings.NewReader(out.String()), schema.Identity(Format{}))
		require.NoError(t, err)
		rows, failed, err := Collect(imp.Rows(t.Context()), 1)

		require.NoError(t, err)
		require.Empty(t, failed)
		require.Len(t, rows, 1)
		assert.Equal(t, item, rows[0].Value)
	})

	invalid := []struct {
		name    string
		format  Format
		options []WriteOption
		wantErr string
	}{
		{
			name: "rejects an unsupported encoding",
			format: Format{
				Encoding: EncodingUTF16,
			},
			wantErr: "unsupported encoding",
		},
		{
			name:    "rejects a column that is not exported",
			options: []WriteOption{WithColumns("active")},
			wantErr: `"active" cannot be exported`,
		},
		{
			name:    "rejects an unknown column",
			options: []WriteOption{WithColumns("other")},
			wantErr: `"other" cannot be exported`,
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			err := New(testColumns()).Write(&out, slices.Values([]*testItem{}), tt.format, tt.options...)

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
