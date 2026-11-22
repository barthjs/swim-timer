package tabular

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldError_Error(t *testing.T) {
	t.Parallel()

	t.Run("names the value and the rule", func(t *testing.T) {
		t.Parallel()

		err := &FieldError{
			Tag:   "number",
			Value: "x",
		}

		assert.Equal(t, `the value "x" fails the rule number`, err.Error())
	})

	t.Run("names the parameter of the rule", func(t *testing.T) {
		t.Parallel()

		err := &FieldError{
			Tag:   "max",
			Param: "5",
			Value: "9",
		}

		assert.Equal(t, `the value "9" fails the rule max=5`, err.Error())
	})
}

func TestFieldErrors_Add(t *testing.T) {
	t.Parallel()

	t.Run("collects the failures of a key", func(t *testing.T) {
		t.Parallel()

		errs := FieldErrors{}

		errs.Add("name", FieldError{
			Tag: "required",
		})
		errs.Add("name", FieldError{
			Tag: "max",
		})

		assert.Len(t, errs["name"], 2)
	})
}

func TestFieldErrors_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns the keys in sorted order", func(t *testing.T) {
		t.Parallel()

		errs := FieldErrors{
			"b":    nil,
			"a":    nil,
			RowKey: nil,
		}

		keys := errs.Keys()

		assert.Equal(t, []string{RowKey, "a", "b"}, keys)
	})
}

func TestFileError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *FileError
		want string
	}{
		{
			name: "names the line and the column",
			err: &FileError{
				Line:   3,
				Column: "name",
				Err:    ErrUnclosedQuote,
			},
			want: "line 3, column name: a quote is not closed",
		},
		{
			name: "names only the line without a column",
			err: &FileError{
				Line: 2,
				Err:  ErrUnclosedQuote,
			},
			want: "line 2: a quote is not closed",
		},
		{
			name: "describes a problem of the whole file without a position",
			err: &FileError{
				Err: ErrEmpty,
			},
			want: "the file is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestFileError_Unwrap(t *testing.T) {
	t.Parallel()

	t.Run("returns the underlying error", func(t *testing.T) {
		t.Parallel()

		err := &FileError{
			Line: 1,
			Err:  ErrEmpty,
		}

		assert.True(t, errors.Is(err, ErrEmpty))
	})
}

func TestEncodingError_Error(t *testing.T) {
	t.Parallel()

	t.Run("names the found and the expected encoding", func(t *testing.T) {
		t.Parallel()

		err := &EncodingError{
			Expected: EncodingWindows1252,
			Found:    EncodingUTF8,
		}

		assert.Equal(t, "the file is utf-8 encoded instead of windows-1252", err.Error())
	})

	t.Run("names the expected encoding without a found one", func(t *testing.T) {
		t.Parallel()

		err := &EncodingError{
			Expected: EncodingUTF8,
		}

		assert.Equal(t, "the file is not valid utf-8", err.Error())
	})
}

func TestMappingError_Error(t *testing.T) {
	t.Parallel()

	t.Run("names every failed key once", func(t *testing.T) {
		t.Parallel()

		err := &MappingError{
			Columns: FieldErrors{
				"name": nil,
				"kind": nil,
			},
			Values: FieldErrors{
				"kind":  nil,
				"count": nil,
			},
		}

		assert.Equal(t, "the mapping of the columns kind, name, count is invalid", err.Error())
	})
}

func TestRowError_Error(t *testing.T) {
	t.Parallel()

	t.Run("names the line and the failed columns", func(t *testing.T) {
		t.Parallel()

		err := &RowError{
			Line: 4,
			Fields: FieldErrors{
				"name": nil,
				RowKey: nil,
			},
		}

		assert.Equal(t, "line 4: the values of row, name are invalid", err.Error())
	})
}
