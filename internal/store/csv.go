package store

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/media"
)

// csvTag is the struct tag that maps a field to its column in an export file.
const csvTag = "csv"

var (
	// bomUTF8 starts a file that was saved as UTF-8 by an editor.
	bomUTF8 = []byte{0xEF, 0xBB, 0xBF}
	// bomUTF16BE and bomUTF16LE start a file that was saved as UTF-16.
	bomUTF16BE = []byte{0xFE, 0xFF}
	bomUTF16LE = []byte{0xFF, 0xFE}
)

// ParseError reports a meet file whose content cannot be read.
// Line and Column are empty if the error concerns the whole file.
type ParseError struct {
	File   string
	Line   int
	Column string
	Err    error
}

// Error describes the problem with the file and its position in it.
func (e *ParseError) Error() string {
	position := e.File
	if e.Line > 0 {
		position += fmt.Sprintf(", line %d", e.Line)
	}
	if e.Column != "" {
		position += ", column " + e.Column
	}
	return fmt.Sprintf("%s: %v", position, e.Err)
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// readMeetFile reads a meet file from the storage. Export files are small, so the whole content is read at once.
func readMeetFile(ctx context.Context, storage media.Storage, path string) ([]byte, error) {
	reader, err := storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

// loadMap loads the content of a CSV file into dest by the key keyFunc returns,
// reporting a key that occurs twice, since one of the entries would be lost.
func loadMap[T any, K comparable](content []byte, dest map[K]*T, keyFunc func(*T) K) error {
	var items []*T
	lines, err := readCSV(content, &items)
	if err != nil {
		return err
	}
	return fillMap(items, lines, dest, keyFunc)
}

// readCSV decodes the content of a Windows-1252 encoded CSV file into target and returns the line of every item
// in the file, so later checks can name it. It skips rows without any value and rejects invalid files.
func readCSV[T any](content []byte, target *[]*T) ([]int, error) {
	// The whole content is checked before decoding, since a file saved as UTF-8
	// can only be recognized after its first multibyte sequence.
	if err := checkEncoding(content); err != nil {
		return nil, err
	}

	rows, err := readRows(bytes.NewReader(content), csvColumns[T]())
	if err != nil {
		return nil, err
	}

	if err := gocsv.UnmarshalCSV(rows, target); err != nil {
		return nil, rows.translate(err)
	}
	return rows.lines, nil
}

// checkEncoding rejects files in another encoding than Windows-1252. The timing software writes no byte order mark
// and practically never valid UTF-8 with special characters, so such a file was saved again by an editor.
func checkEncoding(content []byte) error {
	if bytes.HasPrefix(content, bomUTF8) || bytes.HasPrefix(content, bomUTF16BE) || bytes.HasPrefix(content, bomUTF16LE) {
		return &ParseError{
			Err: errors.New("the file is not Windows-1252 encoded as exported by the timing software"),
		}
	}
	if !isASCII(content) && utf8.Valid(content) {
		return &ParseError{
			Err: errors.New("the file is UTF-8 encoded, but the timing software exports it in Windows-1252"),
		}
	}
	return nil
}

// isASCII reports whether content holds only ASCII characters.
func isASCII(content []byte) bool {
	for _, b := range content {
		if b >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// csvColumns returns the column names the fields of T are read from.
func csvColumns[T any]() []string {
	var columns []string
	t := reflect.TypeFor[T]()
	for field := range t.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get(csvTag), ",")
		if name == "" || name == "-" {
			continue
		}
		columns = append(columns, name)
	}
	return columns
}

// readRows reads and checks every row of an export file. The timing software separates the values
// with semicolons, terminates every row with a trailing separator, and quotes values inconsistently,
// so the reader is lenient about quotes and removes them together with the padding of the values.
// A value never spans several lines, so a value with a line break comes from a quote that is not closed.
func readRows(in io.Reader, columns []string) (*csvRows, error) {
	// A quote that is not closed in the last row of a file without a final line break ends at the end of the input,
	// which the lenient reader accepts, so a line break is appended to let the value span one in every case.
	terminated := io.MultiReader(in, strings.NewReader("\r\n"))
	r := csv.NewReader(charmap.Windows1252.NewDecoder().Reader(terminated))
	r.LazyQuotes = true
	r.Comma = ';'
	r.FieldsPerRecord = -1

	rows := &csvRows{}
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line, _ := r.FieldPos(0)
		if err != nil {
			return nil, &ParseError{
				Line: line,
				Err:  err,
			}
		}
		// The lenient reader takes everything up to the next quote or the end of the file as the value,
		// which would swallow the following rows without an error.
		if i := slices.IndexFunc(record, isMultiline); i >= 0 {
			result := &ParseError{
				Line: line,
				Err:  errors.New("a quote is not closed"),
			}
			if i < len(rows.header) {
				result.Column = rows.header[i]
			}
			return nil, result
		}
		for i := range record {
			record[i] = normalize(record[i])
		}

		if rows.header == nil {
			if err := checkHeader(record, columns); err != nil {
				return nil, &ParseError{
					Line: line,
					Err:  err,
				}
			}
			rows.header = record
			continue
		}

		if isBlank(record) {
			continue
		}
		// A row that ends early was cut off, for example by reading a file that is still being written
		if want := headerLength(rows.header); len(record) < want {
			return nil, &ParseError{
				Line: line,
				Err:  fmt.Errorf("the row has %d of %d values", len(record), want),
			}
		}
		rows.body = append(rows.body, record)
		rows.lines = append(rows.lines, line)
	}

	if rows.header == nil {
		return nil, &ParseError{
			Err: errors.New("the file is empty"),
		}
	}
	return rows, nil
}

// isMultiline reports whether a value contains a line break.
func isMultiline(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

// normalize removes all quotes and surrounding whitespace.
func normalize(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "\""))
}

// checkHeader reports columns that the header of an export file lacks.
func checkHeader(header, columns []string) error {
	var missing []string
	for _, column := range columns {
		if !slices.Contains(header, column) {
			missing = append(missing, column)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the header lacks the columns %s", strings.Join(missing, ", "))
	}
	return nil
}

// isBlank reports whether a row holds no value at all, such as an empty line with separators only.
func isBlank(record []string) bool {
	for _, value := range record {
		if value != "" {
			return false
		}
	}
	return true
}

// headerLength returns the number of named columns, ignoring the empty column of a trailing separator.
func headerLength(header []string) int {
	length := len(header)
	for length > 0 && header[length-1] == "" {
		length--
	}
	return length
}

// csvRows holds the checked rows of an export file and serves them to gocsv.
type csvRows struct {
	header []string
	body   [][]string
	// lines holds the line in the file of every row of body.
	lines []int
}

// Read is not used by gocsv.UnmarshalCSV, which reads all rows at once.
func (c *csvRows) Read() ([]string, error) {
	return nil, errors.New("store: csvRows only supports ReadAll")
}

// ReadAll returns the header followed by every row.
func (c *csvRows) ReadAll() ([][]string, error) {
	return append([][]string{c.header}, c.body...), nil
}

// translate turns an error of gocsv into a ParseError with the line in the file and the column name,
// since gocsv counts the rows after skipping the blank ones.
func (c *csvRows) translate(err error) error {
	parseErr, ok := errors.AsType[*csv.ParseError](err)
	if !ok {
		return err
	}

	result := &ParseError{
		Err: parseErr.Err,
	}
	if row := parseErr.Line - 2; row >= 0 && row < len(c.lines) {
		result.Line = c.lines[row]
	}
	if column := parseErr.Column - 1; column >= 0 && column < len(c.header) {
		result.Column = c.header[column]
	}
	return result
}

// fillMap puts the items into dest by their key and reports a key that occurs twice
// with the line of its second occurrence, taken from lines, which holds the line in the file of every item.
func fillMap[T any, K comparable](items []*T, lines []int, dest map[K]*T, keyFunc func(*T) K) error {
	clear(dest)
	for i, item := range items {
		key := keyFunc(item)
		if _, exists := dest[key]; exists {
			result := &ParseError{
				Err: fmt.Errorf("the key %v occurs more than once", key),
			}
			if i < len(lines) {
				result.Line = lines[i]
			}
			return result
		}
		dest[key] = item
	}
	return nil
}
