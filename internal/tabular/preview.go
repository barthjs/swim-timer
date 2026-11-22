package tabular

import (
	"errors"
	"io"
	"slices"
)

const (
	// previewRows bounds the rows a preview reads to collect the values of the columns.
	previewRows = 10_000
	// previewSampleRows is the number of rows a preview shows.
	previewSampleRows = 5
	// previewMaxValues is the number of distinct values up to which a preview lists the values of a column.
	// A column with more holds free text rather than choices.
	previewMaxValues = 50
)

// Preview is the start of a file with a guessed mapping, so the columns and values can be assigned.
type Preview struct {
	Format Format
	Header []string
	Sample [][]string
	// Mapping is guessed from the keys and aliases of the columns and the choices of their values.
	Mapping Mapping
	// Values holds the distinct values of every header with at most previewMaxValues of them, in the order they occur,
	// so the values of any column a header is assigned to can be translated.
	Values map[string][]string
}

// Preview reads the start of a file and guesses its mapping. The values of a preview only cover its first rows,
// so a value that occurs later is reported by the import as a failure of its row.
func (s *Schema[T]) Preview(r io.Reader, format Format, options ...ReadOption) (Preview, error) {
	reader, err := NewReader(r, format, options...)
	if err != nil {
		return Preview{}, err
	}

	header := reader.Header()
	sample := make([][]string, 0, previewSampleRows)
	distinct := make([][]string, len(header))
	full := make([]bool, len(header))

	count := 0
	for row, err := range reader.Rows() {
		if err != nil {
			if _, isRow := errors.AsType[*RowError](err); isRow {
				continue
			}
			return Preview{}, err
		}
		if len(sample) < previewSampleRows {
			sample = append(sample, row.Values)
		}
		collectValues(row.Values, distinct, full)

		count++
		if count >= previewRows {
			break
		}
	}

	values := make(map[string][]string)
	for i, name := range header {
		if name == "" || full[i] || len(distinct[i]) == 0 {
			continue
		}
		values[name] = distinct[i]
	}

	return Preview{
		Format:  reader.Format(),
		Header:  header,
		Sample:  sample,
		Mapping: s.guessMapping(reader.Format(), header, values),
		Values:  values,
	}, nil
}

// collectValues adds the values of a row to the distinct values of their columns, and marks a column as full
// once it has more than previewMaxValues of them.
func collectValues(row []string, distinct [][]string, full []bool) {
	for i, value := range row {
		if i >= len(distinct) || full[i] || value == "" || slices.Contains(distinct[i], value) {
			continue
		}
		if len(distinct[i]) >= previewMaxValues {
			full[i] = true
			distinct[i] = nil
			continue
		}
		distinct[i] = append(distinct[i], value)
	}
}
