package tabular

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// RowKey is the key of the failures of FieldErrors that concern a row as a whole.
const RowKey = ""

var (
	// ErrEmpty is the error of a file without a header.
	ErrEmpty = errors.New("the file is empty")
	// ErrUnclosedQuote is the error of a value whose quote is not closed, which would swallow the following rows.
	ErrUnclosedQuote = errors.New("a quote is not closed")
)

// FieldError is the failure of one value with the tag of its rule, the parameter, and the text that failed.
type FieldError struct {
	Tag   string
	Param string
	Value string
}

// Error describes the rule the value failed.
func (e *FieldError) Error() string {
	if e.Param == "" {
		return fmt.Sprintf("the value %q fails the rule %s", e.Value, e.Tag)
	}
	return fmt.Sprintf("the value %q fails the rule %s=%s", e.Value, e.Tag, e.Param)
}

// FieldErrors groups failures by the key of their column, with RowKey for the row as a whole.
type FieldErrors map[string][]FieldError

// Add adds a failure of the column with the given key.
func (e FieldErrors) Add(key string, err FieldError) {
	e[key] = append(e[key], err)
}

// Keys returns the keys with a failure in sorted order, so a report of them is stable.
func (e FieldErrors) Keys() []string {
	return slices.Sorted(maps.Keys(e))
}

// FileError reports a file that cannot be read any further, with an empty position when it concerns the whole file.
type FileError struct {
	Line   int
	Column string
	Err    error
}

// Error describes the problem and its position in the file.
func (e *FileError) Error() string {
	var position []string
	if e.Line > 0 {
		position = append(position, fmt.Sprintf("line %d", e.Line))
	}
	if e.Column != "" {
		position = append(position, "column "+e.Column)
	}
	if len(position) == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", strings.Join(position, ", "), e.Err)
}

// Unwrap returns the underlying error.
func (e *FileError) Unwrap() error {
	return e.Err
}

// EncodingError reports a file in another encoding than the expected one.
// Found is the encoding the file is in, empty if it is not valid in the expected one but in no other either.
type EncodingError struct {
	Expected Encoding
	Found    Encoding
}

// Error names the expected and the found encoding.
func (e *EncodingError) Error() string {
	if e.Found == "" {
		return fmt.Sprintf("the file is not valid %s", e.Expected)
	}
	return fmt.Sprintf("the file is %s encoded instead of %s", e.Found, e.Expected)
}

// MappingError reports a mapping that does not fit the file or the schema, before any row is read.
// Columns holds the failures by the key of a column, Values the failures of the value mapping by the same key.
type MappingError struct {
	Columns FieldErrors
	Values  FieldErrors
}

// Error names the keys whose mapping failed.
func (e *MappingError) Error() string {
	keys := e.Columns.Keys()
	for _, key := range e.Values.Keys() {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return "the mapping of the columns " + strings.Join(keys, ", ") + " is invalid"
}

// RowError reports a skipped row whose values fail their columns, with the row as it stands in the file,
// so the rejected rows can be written to a file of their own.
type RowError struct {
	Line   int
	Values []string
	Fields FieldErrors
}

// Error names the line and the keys of the failed columns.
func (e *RowError) Error() string {
	keys := e.Fields.Keys()
	for i, key := range keys {
		if key == RowKey {
			keys[i] = "row"
		}
	}
	return fmt.Sprintf("line %d: the values of %s are invalid", e.Line, strings.Join(keys, ", "))
}
