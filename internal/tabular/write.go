package tabular

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// WriteOption changes how a file is written.
type WriteOption func(*writeSettings)

// writeSettings holds the behavior the WriteOption values select.
type writeSettings struct {
	columns       []string
	header        func(key string) string
	bom           bool
	escapeFormula bool
}

// WithColumns writes only the columns with the given keys, in their order.
func WithColumns(keys ...string) WriteOption {
	return func(s *writeSettings) {
		s.columns = keys
	}
}

// WithHeader names the columns in the header with the given function instead of their keys.
// A file with such a header reads back with a mapping of the names instead of Identity.
func WithHeader(name func(key string) string) WriteOption {
	return func(s *writeSettings) {
		s.header = name
	}
}

// WithByteOrderMark starts a UTF-8 file with a byte order mark, which spreadsheets need to recognize the encoding.
func WithByteOrderMark() WriteOption {
	return func(s *writeSettings) {
		s.bom = true
	}
}

// WithFormulaEscape puts an apostrophe in front of every value of a row that starts with =, +, -, or @,
// so a spreadsheet shows it as text instead of running it as a formula.
// A file written with it is meant to be opened, since reading it back keeps the apostrophes.
func WithFormulaEscape() WriteOption {
	return func(s *writeSettings) {
		s.escapeFormula = true
	}
}

// WriteRows writes rows of text under the given header, a character Windows-1252 lacks as a question mark.
func WriteRows(w io.Writer, header []string, rows iter.Seq[[]string], format Format, options ...WriteOption) error {
	return writeValues(w, header, rows, format, writeSettingsOf(options))
}

// Write writes the targets as rows under a header of the column keys, so the file reads back with Identity,
// and a character Windows-1252 lacks as a question mark.
func (s *Schema[T]) Write(w io.Writer, rows iter.Seq[*T], format Format, options ...WriteOption) error {
	settings := writeSettingsOf(options)

	columns, err := s.exportColumns(settings.columns)
	if err != nil {
		return err
	}

	header := make([]string, 0, len(columns))
	for _, column := range columns {
		name := column.key
		if settings.header != nil {
			name = settings.header(column.key)
		}
		header = append(header, name)
	}

	return writeValues(w, header, formatRows(columns, rows), format, settings)
}

// writeSettingsOf applies the options to empty settings.
func writeSettingsOf(options []WriteOption) writeSettings {
	var settings writeSettings
	for _, option := range options {
		option(&settings)
	}
	return settings
}

// exportColumns returns the exported columns with the given keys in their order, or every exported column.
func (s *Schema[T]) exportColumns(keys []string) ([]Column[T], error) {
	if len(keys) == 0 {
		var columns []Column[T]
		for _, column := range s.columns {
			if column.format != nil {
				columns = append(columns, column)
			}
		}
		return columns, nil
	}

	columns := make([]Column[T], 0, len(keys))
	for _, key := range keys {
		index := slices.IndexFunc(s.columns, func(column Column[T]) bool { return column.key == key })
		if index < 0 || s.columns[index].format == nil {
			return nil, fmt.Errorf("tabular: the column %q cannot be exported", key)
		}
		columns = append(columns, s.columns[index])
	}
	return columns, nil
}

// formatRows returns the text of the columns of every target. The rows share one slice,
// since the writer writes a row before the next one is formatted.
func formatRows[T any](columns []Column[T], targets iter.Seq[*T]) iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		row := make([]string, len(columns))
		for target := range targets {
			for i, column := range columns {
				row[i] = column.format(target)
			}
			if !yield(row) {
				return
			}
		}
	}
}

// writeValues writes the header and the rows in the encoding and with the separator of the format.
func writeValues(w io.Writer, header []string, rows iter.Seq[[]string], format Format, settings writeSettings) error {
	encodingName := cmp.Or(format.Encoding, EncodingUTF8)
	separator := cmp.Or(format.Separator, ",")
	if !slices.Contains(Separators(), separator) {
		return fmt.Errorf("tabular: unsupported separator %q", separator)
	}

	out := w
	switch encodingName {
	case EncodingUTF8:
		if settings.bom {
			if _, err := w.Write(bomUTF8); err != nil {
				return err
			}
		}
	case EncodingWindows1252:
		out = charmap.Windows1252.NewEncoder().Writer(w)
	default:
		return fmt.Errorf("tabular: unsupported encoding %q", encodingName)
	}

	writer := csv.NewWriter(out)
	writer.Comma = rune(separator[0])
	writer.UseCRLF = true

	write := func(row []string, escape bool) error {
		if encodingName == EncodingWindows1252 || escape {
			row = slices.Clone(row)
			for i, value := range row {
				if escape {
					value = escapeFormula(value)
				}
				if encodingName == EncodingWindows1252 {
					value = replaceUnsupported(value)
				}
				row[i] = value
			}
		}
		return writer.Write(row)
	}

	if err := write(header, false); err != nil {
		return err
	}
	for values := range rows {
		if err := write(values, settings.escapeFormula); err != nil {
			return err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	// The encoder buffers an incomplete character until it is closed, which leaves the underlying writer open
	if closer, ok := out.(io.Closer); ok && out != w {
		return closer.Close()
	}
	return nil
}

// escapeFormula puts an apostrophe in front of a value that a spreadsheet would read as a formula.
func escapeFormula(value string) string {
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}

// replaceUnsupported replaces every character Windows-1252 lacks with a question mark.
func replaceUnsupported(value string) string {
	return strings.Map(func(r rune) rune {
		if _, ok := charmap.Windows1252.EncodeRune(r); ok {
			return r
		}
		return '?'
	}, value)
}
