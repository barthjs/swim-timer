package tabular

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// detectBytes is how much of the start of a file the detection of its encoding and separator looks at.
const detectBytes = 64 << 10

// Encoding names the character encoding of a file.
type Encoding string

const (
	EncodingUTF8        Encoding = "utf-8"
	EncodingWindows1252 Encoding = "windows-1252"
	// EncodingUTF16 is only reported for a file with its byte order mark, such files cannot be read.
	EncodingUTF16 Encoding = "utf-16"
)

// Encodings returns the encodings a file can be read and written in.
func Encodings() []Encoding {
	return []Encoding{
		EncodingUTF8,
		EncodingWindows1252,
	}
}

// Separators returns the separators a file can use, in the order that decides a tie of the detection.
func Separators() []string {
	return []string{";", ",", "\t"}
}

var (
	bomUTF8    = []byte{0xEF, 0xBB, 0xBF}
	bomUTF16BE = []byte{0xFE, 0xFF}
	bomUTF16LE = []byte{0xFF, 0xFE}
)

// Format describes how a file is encoded and how its values are separated,
// detected when reading and UTF-8 with a comma when writing for an empty field.
type Format struct {
	Encoding  Encoding
	Separator string
}

// ReadOption changes how a file is read.
type ReadOption func(*readSettings)

// readSettings holds the behavior the ReadOption values select.
type readSettings struct {
	strictEncoding bool
	lenient        bool
}

// StrictEncoding rejects a file that is valid in another encoding than the one of its format,
// even where a byte order mark names it.
func StrictEncoding() ReadOption {
	return func(s *readSettings) {
		s.strictEncoding = true
	}
}

// Lenient accepts quotes inside unquoted values and removes the quotes around every value, for files whose writer quotes inconsistently.
// A value may then not span several lines, since a quote that is not closed would otherwise swallow the following rows.
func Lenient() ReadOption {
	return func(s *readSettings) {
		s.lenient = true
	}
}

// RawRow is a row of a file with the line it starts on. Its values are trimmed.
type RawRow struct {
	Line   int
	Values []string
}

// Reader reads a delimited file row by row, so a file of any size is read with constant memory.
// It skips rows without any value and accepts a trailing separator after the last value of a row.
type Reader struct {
	settings readSettings
	format   Format
	counter  *countingReader
	csv      *csv.Reader
	header   []string
	// width is the number of named columns, without the empty ones of trailing separators.
	width int
}

// NewReader reads the header of a file and detects the empty fields of its format.
// It fails with a FileError for an empty file, one in an unexpected encoding, or a header that names a column twice.
func NewReader(r io.Reader, format Format, options ...ReadOption) (*Reader, error) {
	var settings readSettings
	for _, option := range options {
		option(&settings)
	}

	counter := &countingReader{
		r: r,
	}
	buffered := bufio.NewReaderSize(counter, detectBytes)
	start, err := buffered.Peek(detectBytes)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}

	encoding, bomLength, err := detectEncoding(start, format.Encoding, settings.strictEncoding)
	if err != nil {
		return nil, &FileError{
			Err: err,
		}
	}
	if _, err := buffered.Discard(bomLength); err != nil {
		return nil, err
	}
	separator := format.Separator
	if separator == "" {
		separator = detectSeparator(start[bomLength:])
	}
	if !slices.Contains(Separators(), separator) {
		return nil, fmt.Errorf("tabular: unsupported separator %q", separator)
	}

	var decoded io.Reader = buffered
	if encoding == EncodingWindows1252 {
		decoded = charmap.Windows1252.NewDecoder().Reader(buffered)
	}
	// A quote that is not closed in the last row of a file without a final line break ends at the end of the input,
	// which the lenient reader accepts, so a line break is appended to let the value span one in every case.
	terminated := io.MultiReader(decoded, strings.NewReader("\r\n"))

	csvReader := csv.NewReader(terminated)
	csvReader.Comma = rune(separator[0])
	csvReader.LazyQuotes = settings.lenient
	csvReader.FieldsPerRecord = -1

	reader := &Reader{
		settings: settings,
		format: Format{
			Encoding:  encoding,
			Separator: separator,
		},
		counter: counter,
		csv:     csvReader,
	}
	if err := reader.readHeader(); err != nil {
		return nil, err
	}
	return reader, nil
}

// Header returns the names of the columns, including the empty ones of trailing separators.
func (r *Reader) Header() []string {
	return slices.Clone(r.header)
}

// Format returns the format the file is read with, with the detected fields filled in.
func (r *Reader) Format() Format {
	return r.format
}

// Offset returns how many bytes of the file have been consumed, for the progress of a large file.
// It runs ahead of the rows by the buffer of the reader.
func (r *Reader) Offset() int64 {
	return r.counter.n
}

// Rows returns the rows after the header, a RowError for a short row, and ends with a FileError.
func (r *Reader) Rows() iter.Seq2[RawRow, error] {
	return func(yield func(RawRow, error) bool) {
		for {
			values, line, err := r.read()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(RawRow{}, err)
				return
			}
			if isBlank(values) {
				continue
			}

			row := RawRow{
				Line:   line,
				Values: values,
			}
			// A row that ends early was cut off, as in a file that is still being written
			if len(values) < r.width {
				fields := FieldErrors{}
				fields.Add(RowKey, FieldError{
					Tag:   "short_row",
					Param: strconv.Itoa(r.width),
					Value: strconv.Itoa(len(values)),
				})
				if !yield(row, &RowError{
					Line:   line,
					Values: values,
					Fields: fields,
				}) {
					return
				}
				continue
			}
			if !yield(row, nil) {
				return
			}
		}
	}
}

// CompleteFormat returns the format with the encoding detected from the whole content if it has none,
// for a caller that holds a whole file, see DetectEncoding. A file in UTF-16 keeps the format, so the reader rejects it.
func CompleteFormat(format Format, content []byte) Format {
	if format.Encoding != "" {
		return format
	}
	if encoding := DetectEncoding(content); encoding != EncodingUTF16 {
		format.Encoding = encoding
	}
	return format
}

// DetectEncoding returns the encoding of content: UTF-8 for its byte order mark or for special characters in valid UTF-8,
// UTF-16 for its byte order mark, and Windows-1252 for anything else, in which every byte is valid.
// A Reader detects the encoding from the start of a file, so a caller that holds a whole file passes the encoding detected from all of it,
// which a special character late in the file decides as well.
func DetectEncoding(content []byte) Encoding {
	switch {
	case bytes.HasPrefix(content, bomUTF8):
		return EncodingUTF8
	case bytes.HasPrefix(content, bomUTF16BE), bytes.HasPrefix(content, bomUTF16LE):
		return EncodingUTF16
	case !isASCII(string(content)) && utf8.Valid(content):
		return EncodingUTF8
	default:
		return EncodingWindows1252
	}
}

// readHeader reads the first row as the names of the columns.
func (r *Reader) readHeader() error {
	values, line, err := r.read()
	if errors.Is(err, io.EOF) {
		return &FileError{
			Err: ErrEmpty,
		}
	}
	if err != nil {
		return err
	}

	r.header = values
	r.width = len(values)
	for r.width > 0 && values[r.width-1] == "" {
		r.width--
	}
	for i, name := range values[:r.width] {
		if name != "" && slices.Contains(values[:i], name) {
			return &FileError{
				Line: line,
				Err:  fmt.Errorf("the column %q occurs more than once", name),
			}
		}
	}
	return nil
}

// read returns the next row with the line it starts on, checked against the encoding and trimmed.
func (r *Reader) read() ([]string, int, error) {
	values, err := r.csv.Read()
	if errors.Is(err, io.EOF) {
		return nil, 0, io.EOF
	}
	if err != nil {
		if parseErr, ok := errors.AsType[*csv.ParseError](err); ok {
			return nil, 0, &FileError{
				Line: parseErr.StartLine,
				Err:  parseErr.Err,
			}
		}
		return nil, 0, err
	}
	line, _ := r.csv.FieldPos(0)

	for i, value := range values {
		// The lenient reader takes everything up to the next quote or the end of the file as the value,
		// which would swallow the following rows without an error.
		if r.settings.lenient && strings.ContainsAny(value, "\r\n") {
			return nil, 0, &FileError{
				Line:   line,
				Column: r.columnName(i),
				Err:    ErrUnclosedQuote,
			}
		}
		if err := r.checkEncoding(value); err != nil {
			return nil, 0, &FileError{
				Line:   line,
				Column: r.columnName(i),
				Err:    err,
			}
		}
		values[i] = r.trim(value)
	}
	return values, line, nil
}

// columnName returns the name of the column at index i, empty before the header is read or beyond it.
func (r *Reader) columnName(i int) string {
	if i < len(r.header) {
		return r.header[i]
	}
	return ""
}

// checkEncoding reports a value that is not valid in the encoding of the file. In Windows-1252, every byte is valid,
// so with a strict encoding a value whose bytes form UTF-8 with special characters reveals a file saved as UTF-8.
func (r *Reader) checkEncoding(value string) error {
	switch r.format.Encoding {
	case EncodingUTF8:
		if !utf8.ValidString(value) {
			return &EncodingError{
				Expected: EncodingUTF8,
			}
		}
	case EncodingWindows1252:
		if !r.settings.strictEncoding || isASCII(value) {
			return nil
		}
		raw, err := charmap.Windows1252.NewEncoder().String(value)
		if err == nil && utf8.ValidString(raw) {
			return &EncodingError{
				Expected: EncodingWindows1252,
				Found:    EncodingUTF8,
			}
		}
	case EncodingUTF16:
		// A file in UTF-16 is rejected before its first row
	}
	return nil
}

// trim removes the padding around a value, and for a lenient file the quotes around it as well.
func (r *Reader) trim(value string) string {
	if !r.settings.lenient {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "\""))
}

// detectEncoding returns the encoding of a file from its byte order mark or its start, and the length of the mark.
func detectEncoding(start []byte, expected Encoding, strict bool) (Encoding, int, error) {
	if bytes.HasPrefix(start, bomUTF16BE) || bytes.HasPrefix(start, bomUTF16LE) {
		return "", 0, &EncodingError{
			Expected: cmp.Or(expected, EncodingUTF8),
			Found:    EncodingUTF16,
		}
	}
	if bytes.HasPrefix(start, bomUTF8) {
		if strict && expected != "" && expected != EncodingUTF8 {
			return "", 0, &EncodingError{
				Expected: expected,
				Found:    EncodingUTF8,
			}
		}
		return EncodingUTF8, len(bomUTF8), nil
	}

	if expected != "" {
		if !slices.Contains(Encodings(), expected) {
			return "", 0, fmt.Errorf("unsupported encoding %q", expected)
		}
		return expected, 0, nil
	}
	return DetectEncoding(completeRunes(start)), 0, nil
}

// detectSeparator returns the separator that occurs most often in the first line of a file outside quotes,
// a comma if none does.
func detectSeparator(start []byte) string {
	counts := make(map[byte]int)
	quoted := false
	for _, b := range start {
		if b == '\n' && !quoted {
			break
		}
		if b == '"' {
			quoted = !quoted
			continue
		}
		if !quoted {
			counts[b]++
		}
	}

	best := ","
	bestCount := 0
	for _, separator := range Separators() {
		if count := counts[separator[0]]; count > bestCount {
			best = separator
			bestCount = count
		}
	}
	return best
}

// completeRunes returns start without a rune the end of the detected part cuts off.
func completeRunes(start []byte) []byte {
	for i := 1; i < utf8.UTFMax && i <= len(start); i++ {
		if utf8.RuneStart(start[len(start)-i]) {
			if !utf8.FullRune(start[len(start)-i:]) {
				return start[:len(start)-i]
			}
			break
		}
	}
	return start
}

// isASCII reports whether s holds only ASCII characters.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// isBlank reports whether a row holds no value at all.
func isBlank(values []string) bool {
	for _, value := range values {
		if value != "" {
			return false
		}
	}
	return true
}

// countingReader counts the bytes read from r.
type countingReader struct {
	r io.Reader
	n int64
}

// Read reads from the underlying reader and counts the bytes.
func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
