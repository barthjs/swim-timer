package tabular

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	detected := []struct {
		name    string
		content []byte
		want    Format
	}{
		{
			name:    "detects utf-8 with special characters",
			content: []byte("name;city\r\nx;Grün\r\n"),
			want: Format{
				Encoding:  EncodingUTF8,
				Separator: ";",
			},
		},
		{
			name:    "detects windows-1252 with special characters",
			content: encodeWindows1252(t, "name,city\r\nx,Grün\r\n"),
			want: Format{
				Encoding:  EncodingWindows1252,
				Separator: ",",
			},
		},
		{
			name:    "detects utf-8 by its byte order mark",
			content: append(slices.Clone(bomUTF8), []byte("name\tcity\r\n")...),
			want: Format{
				Encoding:  EncodingUTF8,
				Separator: "\t",
			},
		},
		{
			name:    "reads a file of plain ascii as windows-1252 with a comma",
			content: []byte("name\r\n"),
			want: Format{
				Encoding:  EncodingWindows1252,
				Separator: ",",
			},
		},
		{
			name:    "ignores separators inside quotes",
			content: []byte("\"a;b;c\",name\r\n"),
			want: Format{
				Encoding:  EncodingWindows1252,
				Separator: ",",
			},
		},
	}

	for _, tt := range detected {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader, err := NewReader(strings.NewReader(string(tt.content)), Format{})

			require.NoError(t, err)
			assert.Equal(t, tt.want, reader.Format())
		})
	}

	t.Run("keeps the given format", func(t *testing.T) {
		t.Parallel()

		format := Format{
			Encoding:  EncodingUTF8,
			Separator: ",",
		}

		reader, err := NewReader(strings.NewReader("a;b,c\r\n"), format)

		require.NoError(t, err)
		assert.Equal(t, format, reader.Format())
		assert.Equal(t, []string{"a;b", "c"}, reader.Header())
	})

	t.Run("lets a byte order mark select utf-8 without a strict encoding", func(t *testing.T) {
		t.Parallel()

		content := append(slices.Clone(bomUTF8), []byte("name\r\n")...)

		reader, err := NewReader(strings.NewReader(string(content)), Format{
			Encoding: EncodingWindows1252,
		})

		require.NoError(t, err)
		assert.Equal(t, EncodingUTF8, reader.Format().Encoding)
	})

	invalid := []struct {
		name    string
		content []byte
		format  Format
		options []ReadOption
		wantErr error
	}{
		{
			name:    "rejects an empty file",
			content: nil,
			wantErr: ErrEmpty,
		},
		{
			name:    "rejects a file of blank lines",
			content: []byte("\r\n\r\n"),
			wantErr: ErrEmpty,
		},
		{
			name:    "rejects a header that names a column twice",
			content: []byte("name;name\r\n"),
			wantErr: nil,
		},
		{
			name:    "rejects a utf-16 byte order mark",
			content: append(slices.Clone(bomUTF16LE), 'a', 0),
			wantErr: &EncodingError{
				Expected: EncodingUTF8,
				Found:    EncodingUTF16,
			},
		},
		{
			name:    "rejects a utf-8 byte order mark with a strict other encoding",
			content: append(slices.Clone(bomUTF8), []byte("name\r\n")...),
			format: Format{
				Encoding: EncodingWindows1252,
			},
			options: []ReadOption{
				StrictEncoding(),
			},
			wantErr: &EncodingError{
				Expected: EncodingWindows1252,
				Found:    EncodingUTF8,
			},
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewReader(strings.NewReader(string(tt.content)), tt.format, tt.options...)

			fileErr, ok := errors.AsType[*FileError](err)
			require.True(t, ok, "got %v", err)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), fileErr.Err.Error())
			}
		})
	}

	t.Run("rejects an unsupported separator", func(t *testing.T) {
		t.Parallel()

		_, err := NewReader(strings.NewReader("name\r\n"), Format{
			Separator: "|",
		})

		assert.ErrorContains(t, err, "unsupported separator")
	})
}

func TestReader_Header(t *testing.T) {
	t.Parallel()

	t.Run("returns the trimmed names with the empty column of a trailing separator", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader(" id ;name;\r\n"), Format{})
		require.NoError(t, err)

		header := reader.Header()

		assert.Equal(t, []string{"id", "name", ""}, header)
	})
}

func TestReader_Format(t *testing.T) {
	t.Parallel()

	t.Run("returns the detected format", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader("a;b\r\n"), Format{
			Encoding: EncodingUTF8,
		})
		require.NoError(t, err)

		format := reader.Format()

		assert.Equal(t, Format{
			Encoding:  EncodingUTF8,
			Separator: ";",
		}, format)
	})
}

func TestReader_Offset(t *testing.T) {
	t.Parallel()

	t.Run("counts the consumed bytes", func(t *testing.T) {
		t.Parallel()

		content := "id;name\r\n1;x\r\n"
		reader, err := NewReader(strings.NewReader(content), Format{})
		require.NoError(t, err)

		for _, err := range reader.Rows() {
			require.NoError(t, err)
		}

		assert.Equal(t, int64(len(content)), reader.Offset())
	})
}

func TestReader_Rows(t *testing.T) {
	t.Parallel()

	t.Run("returns the rows with their lines and skips blank rows", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(8)
		reader, err := NewReader(strings.NewReader("id;name\r\n1;"+name+"\r\n\r\n;;\r\n2; y \r\n"), Format{})
		require.NoError(t, err)

		rows := collectRaw(t, reader)

		assert.Equal(t, []RawRow{
			{
				Line: 2,
				Values: []string{
					"1",
					name,
				},
			},
			{
				Line: 5,
				Values: []string{
					"2",
					"y",
				},
			},
		}, rows)
	})

	t.Run("reads a value that spans several lines", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader("id;text\r\n1;\"a\r\nb\"\r\n2;c\r\n"), Format{})
		require.NoError(t, err)

		rows := collectRaw(t, reader)

		require.Len(t, rows, 2)
		assert.Equal(t, "a\nb", rows[0].Values[1])
		assert.Equal(t, 4, rows[1].Line)
	})

	t.Run("decodes windows-1252", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader(string(encodeWindows1252(t, "name\r\näß\r\n"))), Format{})
		require.NoError(t, err)

		rows := collectRaw(t, reader)

		require.Len(t, rows, 1)
		assert.Equal(t, "äß", rows[0].Values[0])
	})

	t.Run("removes the quotes of a lenient file and keeps a stray quote inside a value", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader("id;name\r\n1;\"x\"y\";\r\n2;  \"z\"  ;\r\n"), Format{}, Lenient())
		require.NoError(t, err)

		rows := collectRaw(t, reader)

		require.Len(t, rows, 2)
		assert.Equal(t, `x"y`, rows[0].Values[1])
		assert.Equal(t, "z", rows[1].Values[1])
	})

	t.Run("reports a row that ends early and reads the rows after it", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader("id;name;\r\n1\r\n2;x\r\n"), Format{})
		require.NoError(t, err)

		var rows []RawRow
		var errs []error
		for row, err := range reader.Rows() {
			rows = append(rows, row)
			errs = append(errs, err)
		}

		require.Len(t, errs, 2)
		rowErr, ok := errors.AsType[*RowError](errs[0])
		require.True(t, ok)
		assert.Equal(t, 2, rowErr.Line)
		assert.Equal(t, []string{"1"}, rowErr.Values)
		assert.Equal(t, []FieldError{
			{
				Tag:   "short_row",
				Param: "2",
				Value: "1",
			},
		}, rowErr.Fields[RowKey])
		assert.NoError(t, errs[1], "a row without the trailing separator holds every named column")
		assert.Equal(t, 3, rows[1].Line)
	})

	invalid := []struct {
		name       string
		content    string
		options    []ReadOption
		wantLine   int
		wantColumn string
		wantErr    string
	}{
		{
			name:       "rejects a quote of a lenient file that is not closed",
			content:    "id;name\r\n1;x;\r\n2;\"y;\r\n3;z;\r\n",
			options:    []ReadOption{Lenient()},
			wantLine:   3,
			wantColumn: "name",
			wantErr:    ErrUnclosedQuote.Error(),
		},
		{
			name:       "rejects a quote of a lenient file that is not closed at the end of the file",
			content:    "id;name\r\n1;x;\r\n2;\"y",
			options:    []ReadOption{Lenient()},
			wantLine:   3,
			wantColumn: "name",
			wantErr:    ErrUnclosedQuote.Error(),
		},
		{
			name:     "rejects a stray quote of a strict file",
			content:  "id;name\r\n1;x\"y\r\n",
			wantLine: 2,
			wantErr:  "bare",
		},
		{
			name:       "rejects utf-8 in a strict windows-1252 file",
			content:    "id;name\r\n1;x\r\n2;Grün\r\n",
			options:    []ReadOption{StrictEncoding()},
			wantLine:   3,
			wantColumn: "name",
			wantErr:    "utf-8 encoded instead of windows-1252",
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader, err := NewReader(strings.NewReader(tt.content), Format{
				Encoding: EncodingWindows1252,
			}, tt.options...)
			require.NoError(t, err)

			var last error
			for _, err := range reader.Rows() {
				last = err
			}

			fileErr, ok := errors.AsType[*FileError](last)
			require.True(t, ok, "got %v", last)
			assert.Equal(t, tt.wantLine, fileErr.Line)
			assert.Equal(t, tt.wantColumn, fileErr.Column)
			assert.ErrorContains(t, fileErr, tt.wantErr)
		})
	}

	t.Run("rejects invalid utf-8", func(t *testing.T) {
		t.Parallel()

		reader, err := NewReader(strings.NewReader("name\r\nGrün\r\n\xff\r\n"), Format{
			Encoding: EncodingUTF8,
		})
		require.NoError(t, err)

		var last error
		for _, err := range reader.Rows() {
			last = err
		}

		fileErr, ok := errors.AsType[*FileError](last)
		require.True(t, ok, "got %v", last)
		assert.Equal(t, 3, fileErr.Line)
		assert.ErrorContains(t, fileErr, "not valid utf-8")
	})
}

func TestCompleteFormat(t *testing.T) {
	t.Parallel()

	t.Run("fills an empty encoding from the whole content", func(t *testing.T) {
		t.Parallel()

		content := append(bytes.Repeat([]byte("a"), detectBytes+1), []byte("Grün")...)

		format := CompleteFormat(Format{
			Separator: ";",
		}, content)

		assert.Equal(t, Format{
			Encoding:  EncodingUTF8,
			Separator: ";",
		}, format)
	})

	t.Run("keeps a given encoding", func(t *testing.T) {
		t.Parallel()

		format := CompleteFormat(Format{
			Encoding: EncodingWindows1252,
		}, []byte("Grün"))

		assert.Equal(t, EncodingWindows1252, format.Encoding)
	})

	t.Run("leaves the encoding of utf-16 to the reader", func(t *testing.T) {
		t.Parallel()

		format := CompleteFormat(Format{}, append(slices.Clone(bomUTF16LE), 'a', 0))

		assert.Empty(t, format.Encoding)
	})
}

func TestDetectEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		want    Encoding
	}{
		{
			name:    "detects utf-8 by its special characters",
			content: []byte("Grün"),
			want:    EncodingUTF8,
		},
		{
			name:    "detects utf-8 by its byte order mark",
			content: append(slices.Clone(bomUTF8), 'a'),
			want:    EncodingUTF8,
		},
		{
			name:    "detects utf-16 by its byte order mark",
			content: append(slices.Clone(bomUTF16BE), 0, 'a'),
			want:    EncodingUTF16,
		},
		{
			name:    "detects windows-1252 by special characters that are no valid utf-8",
			content: encodeWindows1252(t, "Grün"),
			want:    EncodingWindows1252,
		},
		{
			name:    "reads plain ascii as windows-1252",
			content: []byte("Zurich"),
			want:    EncodingWindows1252,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoding := DetectEncoding(tt.content)

			assert.Equal(t, tt.want, encoding)
		})
	}
}

// collectRaw reads every row of a reader and fails the test on an error.
func collectRaw(t *testing.T, reader *Reader) []RawRow {
	t.Helper()

	var rows []RawRow
	for row, err := range reader.Rows() {
		require.NoError(t, err)
		rows = append(rows, row)
	}
	return rows
}
