package tabular

import (
	"context"
	"errors"
	"io"
	"iter"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Row is a target read from a row of a file, with the line the row starts on.
type Row[T any] struct {
	Line  int
	Value *T
}

// Import reads the rows of a file into targets of a schema.
type Import[T any] struct {
	schema  *Schema[T]
	reader  *Reader
	mapping Mapping
	// sources holds the index in the header each column is read from by the key of the column,
	// -1 for a column the mapping leaves out.
	sources map[string]int
}

// Read reads the header of a file in the format of the mapping and checks the mapping against the header and the schema.
// It fails with a FileError for a file that cannot be read and a MappingError for a mapping
// that names an unknown column or header, leaves out a required column, or translates into a value that is no choice of its column.
func (s *Schema[T]) Read(r io.Reader, mapping Mapping, options ...ReadOption) (*Import[T], error) {
	reader, err := NewReader(r, mapping.Format, options...)
	if err != nil {
		return nil, err
	}

	sources, err := s.resolveMapping(mapping, reader.Header())
	if err != nil {
		return nil, err
	}
	return &Import[T]{
		schema:  s,
		reader:  reader,
		mapping: mapping,
		sources: sources,
	}, nil
}

// ReadByHeader maps the columns of a file whose header names them by their keys or aliases,
// and fails with a MappingError for a missing required column.
func (s *Schema[T]) ReadByHeader(r io.Reader, format Format, options ...ReadOption) (*Import[T], error) {
	reader, err := NewReader(r, format, options...)
	if err != nil {
		return nil, err
	}

	mapping := s.guessMapping(reader.Format(), reader.Header(), nil)
	mapping.Values = nil
	sources, err := s.resolveMapping(mapping, reader.Header())
	if err != nil {
		return nil, err
	}
	return &Import[T]{
		schema:  s,
		reader:  reader,
		mapping: mapping,
		sources: sources,
	}, nil
}

// resolveMapping returns the index in the header of every column, see Import.sources.
func (s *Schema[T]) resolveMapping(mapping Mapping, header []string) (map[string]int, error) {
	failed := &MappingError{
		Columns: FieldErrors{},
		Values:  FieldErrors{},
	}
	sources := make(map[string]int, len(s.columns))

	for key := range mapping.Columns {
		if !slices.ContainsFunc(s.columns, func(column Column[T]) bool { return column.key == key }) {
			failed.Columns.Add(key, FieldError{
				Tag: "invalid",
			})
		}
	}
	for _, column := range s.columns {
		source := mapping.Columns[column.key]
		if source == "" {
			if column.required {
				failed.Columns.Add(column.key, FieldError{
					Tag: "required",
				})
			}
			sources[column.key] = -1
			continue
		}
		index := slices.Index(header, source)
		if index < 0 {
			failed.Columns.Add(column.key, FieldError{
				Tag:   "missing",
				Value: source,
			})
			continue
		}
		sources[column.key] = index

		for value, choice := range mapping.Values[column.key] {
			if choice != "" && len(column.choices) > 0 && !slices.Contains(column.choices, choice) {
				failed.Values.Add(column.key, FieldError{
					Tag:   "oneof",
					Param: strings.Join(column.choices, " "),
					Value: value,
				})
			}
		}
	}

	if len(failed.Columns) > 0 || len(failed.Values) > 0 {
		return nil, failed
	}
	return sources, nil
}

// Header returns the names of the columns of the file, see Reader.Header.
func (i *Import[T]) Header() []string {
	return i.reader.Header()
}

// Format returns the format the file is read with, with the detected fields filled in.
func (i *Import[T]) Format() Format {
	return i.reader.Format()
}

// Offset returns how many bytes of the file have been consumed, see Reader.Offset.
func (i *Import[T]) Offset() int64 {
	return i.reader.Offset()
}

// Rows returns the targets of the rows and a RowError for each row that fails, until a FileError,
// an error of the resolve hook, or the end of ctx.
func (i *Import[T]) Rows(ctx context.Context) iter.Seq2[Row[T], error] {
	return func(yield func(Row[T], error) bool) {
		for raw, err := range i.reader.Rows() {
			if ctxErr := ctx.Err(); ctxErr != nil {
				yield(Row[T]{}, ctxErr)
				return
			}
			if err != nil {
				if _, isRow := errors.AsType[*RowError](err); isRow {
					if !yield(Row[T]{
						Line: raw.Line,
					}, err) {
						return
					}
					continue
				}
				yield(Row[T]{}, err)
				return
			}

			row, err := i.read(ctx, raw)
			if err != nil {
				if _, isRow := errors.AsType[*RowError](err); isRow {
					if !yield(Row[T]{
						Line: raw.Line,
					}, err) {
						return
					}
					continue
				}
				yield(Row[T]{}, err)
				return
			}
			if !yield(row, nil) {
				return
			}
		}
	}
}

// read turns one row of the file into a target, see Rows.
func (i *Import[T]) read(ctx context.Context, raw RawRow) (Row[T], error) {
	values := i.values(raw)

	failed := FieldErrors{}
	if i.schema.prepare != nil {
		for key, errs := range i.schema.prepare(ctx, values) {
			for _, err := range errs {
				failed.Add(key, err)
			}
		}
		if len(failed) > 0 {
			return Row[T]{}, &RowError{
				Line:   raw.Line,
				Values: raw.Values,
				Fields: failed,
			}
		}
	}

	target := new(T)
	if i.schema.resolve != nil {
		resolved, err := i.schema.resolve(ctx, values)
		if err != nil {
			return Row[T]{}, err
		}
		target = resolved
	}

	for _, column := range i.schema.columns {
		if err := setColumn(ctx, i.schema.rules, column, target, values[column.key]); err != nil {
			failed.Add(column.key, *err)
		}
	}
	if len(failed) == 0 && i.schema.validate != nil {
		failed = i.schema.validate(ctx, target)
	}
	if len(failed) > 0 {
		return Row[T]{}, &RowError{
			Line:   raw.Line,
			Values: raw.Values,
			Fields: failed,
		}
	}

	return Row[T]{
		Line:  raw.Line,
		Value: target,
	}, nil
}

// values returns the text of every column of a row by its key, translated by the mapping and normalized.
func (i *Import[T]) values(raw RawRow) map[string]string {
	values := make(map[string]string, len(i.schema.columns))
	for _, column := range i.schema.columns {
		index := i.sources[column.key]
		if index < 0 || index >= len(raw.Values) {
			values[column.key] = ""
			continue
		}

		value := raw.Values[index]
		if translated, ok := i.mapping.Values[column.key][value]; ok {
			value = translated
		}
		if column.normalize != nil {
			value = column.normalize(value)
		}
		values[column.key] = value
	}
	return values
}

// setColumn checks the text of a column and assigns its value to the target.
// An empty text leaves the target as it is, unless the column is required.
func setColumn[T any](ctx context.Context, rules *validator.Validate, column Column[T], target *T, raw string) *FieldError {
	if raw == "" {
		if column.required {
			return &FieldError{
				Tag: "required",
			}
		}
		return nil
	}
	if len(column.choices) > 0 && !slices.Contains(column.choices, raw) {
		return &FieldError{
			Tag:   "oneof",
			Param: strings.Join(column.choices, " "),
			Value: raw,
		}
	}
	return column.set(ctx, rules, target, raw)
}

// Batches groups the targets of rows into slices of up to size rows, with the errors in between in their order.
func Batches[T any](rows iter.Seq2[Row[T], error], size int) iter.Seq2[[]Row[T], error] {
	return func(yield func([]Row[T], error) bool) {
		batch := make([]Row[T], 0, size)
		for row, err := range rows {
			if err != nil {
				if !yield(nil, err) {
					return
				}
				continue
			}
			batch = append(batch, row)
			if len(batch) < size {
				continue
			}
			if !yield(batch, nil) {
				return
			}
			batch = make([]Row[T], 0, size)
		}
		if len(batch) > 0 {
			yield(batch, nil)
		}
	}
}

// Collect reads the rows of a small file into the targets of the valid rows and the failures of up to maxErrors others.
func Collect[T any](rows iter.Seq2[Row[T], error], maxErrors int) ([]Row[T], []*RowError, error) {
	var valid []Row[T]
	var failed []*RowError
	for row, err := range rows {
		if err == nil {
			valid = append(valid, row)
			continue
		}
		rowErr, isRow := errors.AsType[*RowError](err)
		if !isRow {
			return nil, nil, err
		}
		failed = append(failed, rowErr)
		if len(failed) >= maxErrors {
			break
		}
	}
	return valid, failed, nil
}
