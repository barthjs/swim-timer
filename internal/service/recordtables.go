package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/tabular"
)

// recordImportMaxErrors bounds the failed rows an import reports, as many as the dialog that shows them lists.
const recordImportMaxErrors = 100

// RecordTableState is a record table with the number of its records.
type RecordTableState struct {
	Table       domain.RecordTable
	RecordCount int
}

// RecordTableList is every record table in its order.
type RecordTableList struct {
	LoadedAt time.Time
	Tables   []RecordTableState
}

// RecordEntry is a record with the table it belongs to.
type RecordEntry struct {
	Table  domain.RecordTable
	Record domain.Record
}

// RecordList is every record of every table, in the order of the tables and of the records in their files.
type RecordList struct {
	LoadedAt time.Time
	Records  []RecordEntry
}

// RecordTablePreview is the start of a file of records with a guessed mapping and the columns it maps onto.
type RecordTablePreview struct {
	Preview tabular.Preview
	Columns []tabular.ColumnInfo
}

// RecordTableCreateInput is a new record table with the file its records are imported from.
type RecordTableCreateInput struct {
	Name         string
	Abbreviation string
	Enabled      bool
	Content      []byte
	Mapping      tabular.Mapping
}

// RecordTableUpdateInput replaces the name, the abbreviation, and the state of a record table.
type RecordTableUpdateInput struct {
	Name         string
	Abbreviation string
	Enabled      bool
}

// RecordTableImportInput replaces the records of a table with the records of a file.
type RecordTableImportInput struct {
	Content []byte
	Mapping tabular.Mapping
}

// RecordTableExportInput selects the format and the columns of an export of a record table.
type RecordTableExportInput struct {
	Format tabular.Format
	// Columns holds the keys of the exported columns in their order, every column if empty.
	Columns []string
}

// RecordTableService manages the record tables, which the record breaks of the meet are checked against.
type RecordTableService struct {
	store     *store.RecordTableStore
	meetStore *store.MeetStore
}

// NewRecordTableService creates a RecordTableService whose meet data suggests the values of the categories and styles
// a file of records is mapped onto.
func NewRecordTableService(store *store.RecordTableStore, meetStore *store.MeetStore) *RecordTableService {
	return &RecordTableService{
		store:     store,
		meetStore: meetStore,
	}
}

// List returns every record table in its order with the number of its records.
func (s *RecordTableService) List() RecordTableList {
	tables, loadedAt := s.store.List()

	list := RecordTableList{
		LoadedAt: loadedAt,
		Tables:   make([]RecordTableState, 0, len(tables)),
	}
	for _, data := range tables {
		list.Tables = append(list.Tables, toRecordTableState(data))
	}
	return list
}

// Get returns the record table with the given id, or apperr.ErrNotFound if it does not exist.
func (s *RecordTableService) Get(id string) (RecordTableState, error) {
	data, err := s.store.Get(id)
	if err != nil {
		return RecordTableState{}, err
	}
	return toRecordTableState(data), nil
}

// Create adds a record table at the end of the order with the records of a file. A file that cannot be read,
// a mapping that does not fit, or a failed row fails with an apperr.ValidationError, see importRecords, and nothing is stored.
func (s *RecordTableService) Create(ctx context.Context, input RecordTableCreateInput) (RecordTableState, error) {
	records, err := s.importRecords(input.Content, input.Mapping)
	if err != nil {
		return RecordTableState{}, err
	}

	table := domain.RecordTable{
		ID:           strings.ToLower(rand.Text()),
		Name:         input.Name,
		Abbreviation: input.Abbreviation,
		Enabled:      input.Enabled,
	}
	if err := s.store.Save(ctx, table, records); err != nil {
		return RecordTableState{}, err
	}
	return RecordTableState{
		Table:       table,
		RecordCount: len(records),
	}, nil
}

// Update replaces the name, the abbreviation, and the state of the record table with the given id,
// or returns apperr.ErrNotFound if it does not exist.
func (s *RecordTableService) Update(ctx context.Context, id string, input RecordTableUpdateInput) (RecordTableState, error) {
	table := domain.RecordTable{
		ID:           id,
		Name:         input.Name,
		Abbreviation: input.Abbreviation,
		Enabled:      input.Enabled,
	}
	if err := s.store.Update(ctx, table); err != nil {
		return RecordTableState{}, err
	}
	return s.Get(id)
}

// Delete removes the record table with the given id, or returns apperr.ErrNotFound if it does not exist.
// The records set at the meet stay, since they were announced already.
func (s *RecordTableService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListRecords returns every record of every table, see RecordList.
func (s *RecordTableService) ListRecords() RecordList {
	tables, loadedAt := s.store.List()

	list := RecordList{
		LoadedAt: loadedAt,
	}
	for _, data := range tables {
		for _, record := range data.Records {
			list.Records = append(list.Records, RecordEntry{
				Table:  data.Table,
				Record: record,
			})
		}
	}
	return list
}

// Preview reads the start of a file of records and guesses its mapping.
// A file that cannot be read fails with an apperr.ValidationError on the field "file".
func (s *RecordTableService) Preview(content []byte, format tabular.Format) (RecordTablePreview, error) {
	schema := s.importSchema()
	preview, err := schema.Preview(bytes.NewReader(content), tabular.CompleteFormat(format, content))
	if err != nil {
		return RecordTablePreview{}, recordFileError(err)
	}
	return RecordTablePreview{
		Preview: preview,
		Columns: schema.Columns(),
	}, nil
}

// Import replaces the records of the table with the given id with the records of a file,
// or returns apperr.ErrNotFound if it does not exist. It fails like Create and keeps the records on failure.
func (s *RecordTableService) Import(ctx context.Context, id string, input RecordTableImportInput) (RecordTableState, error) {
	if _, err := s.store.Get(id); err != nil {
		return RecordTableState{}, err
	}

	records, err := s.importRecords(input.Content, input.Mapping)
	if err != nil {
		return RecordTableState{}, err
	}
	table, err := s.store.SaveRecords(ctx, id, records)
	if err != nil {
		return RecordTableState{}, err
	}
	return RecordTableState{
		Table:       table,
		RecordCount: len(records),
	}, nil
}

// Reorder puts the record tables in the order of ids, which decides which table a record break counts for.
// ids has to name every table once, otherwise it fails with an apperr.ValidationError on the field "ids".
func (s *RecordTableService) Reorder(ctx context.Context, ids []string) error {
	err := s.store.Reorder(ctx, ids)
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.NewValidationError("ids", "invalid")
	}
	return err
}

// Reload reads every record table again and keeps the previous ones when a file fails with a DataFileError.
func (s *RecordTableService) Reload(ctx context.Context) error {
	return s.store.LoadAll(ctx)
}

// Export writes the records of the table with the given id in the format and with the columns of input,
// or returns apperr.ErrNotFound if it does not exist. An unknown column fails with an apperr.ValidationError.
func (s *RecordTableService) Export(w io.Writer, id string, input RecordTableExportInput) error {
	if _, err := s.store.Get(id); err != nil {
		return err
	}
	columns := s.importSchema().Columns()
	for _, key := range input.Columns {
		if !slices.ContainsFunc(columns, func(column tabular.ColumnInfo) bool { return column.Key == key }) {
			return apperr.NewValidationError("columns", "oneof")
		}
	}
	return s.store.WriteRecords(w, id, input.Format, input.Columns)
}

// importRecords reads every record of a file with a mapping. A file that cannot be read fails on the field "file",
// a mapping that does not fit on "mapping.columns.<key>" and "mapping.values.<key>",
// and failed rows on "rows.<line>.<key>", or "rows.<line>" for a failure of the whole row,
// each with an apperr.ValidationError.
func (s *RecordTableService) importRecords(content []byte, mapping tabular.Mapping) ([]domain.Record, error) {
	mapping.Format = tabular.CompleteFormat(mapping.Format, content)
	imp, err := s.importSchema().Read(bytes.NewReader(content), mapping)
	if err != nil {
		return nil, recordFileError(err)
	}

	// The file is read from memory, so nothing can cancel the reading of its rows
	rows, failed, err := tabular.Collect(imp.Rows(context.Background()), recordImportMaxErrors)
	if err != nil {
		return nil, recordFileError(err)
	}
	if len(failed) > 0 {
		fields := apperr.FieldErrors{}
		for _, rowErr := range failed {
			prefix := "rows." + strconv.Itoa(rowErr.Line)
			for key, errs := range rowErr.Fields {
				field := prefix
				if key != tabular.RowKey {
					field += "." + key
				}
				fields[field] = toAppFieldErrors(errs)
			}
		}
		return nil, &apperr.ValidationError{
			Fields: fields,
		}
	}

	records := make([]domain.Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, *row.Value)
	}
	return records, nil
}

// importSchema returns the schema of the records with the abbreviations of the categories and styles of the meet data as suggested values.
func (s *RecordTableService) importSchema() *tabular.Schema[domain.Record] {
	var genders, strokes []string
	for _, event := range s.meetStore.Snapshot().Events {
		if event.Category != nil && event.Category.Abbreviation != "" && !slices.Contains(genders, event.Category.Abbreviation) {
			genders = append(genders, event.Category.Abbreviation)
		}
		if event.Style != nil && event.Style.Abbreviation != "" && !slices.Contains(strokes, event.Style.Abbreviation) {
			strokes = append(strokes, event.Style.Abbreviation)
		}
	}
	slices.Sort(genders)
	slices.Sort(strokes)

	return store.NewRecordSchema(store.RecordSchemaOptions{
		Genders: genders,
		Strokes: strokes,
	})
}

// recordFileError turns an error of reading a file of records into an apperr.ValidationError,
// see importRecords. Any other error is returned as it is.
func recordFileError(err error) error {
	if mappingErr, ok := errors.AsType[*tabular.MappingError](err); ok {
		fields := apperr.FieldErrors{}
		for key, errs := range mappingErr.Columns {
			fields["mapping.columns."+key] = toAppFieldErrors(errs)
		}
		for key, errs := range mappingErr.Values {
			fields["mapping.values."+key] = toAppFieldErrors(errs)
		}
		return &apperr.ValidationError{
			Fields: fields,
		}
	}
	if fileErr, ok := errors.AsType[*tabular.FileError](err); ok {
		return &apperr.ValidationError{
			Fields: apperr.FieldErrors{
				"file": {
					fileFieldError(fileErr),
				},
			},
		}
	}
	return err
}

// fileFieldError describes a file that cannot be read by a tag the interface translates,
// with the line from which on it cannot be read as the parameter if it is known.
func fileFieldError(fileErr *tabular.FileError) apperr.FieldError {
	if _, ok := errors.AsType[*tabular.EncodingError](fileErr); ok {
		return apperr.FieldError{
			Tag: "encoding",
		}
	}
	if errors.Is(fileErr, tabular.ErrEmpty) {
		return apperr.FieldError{
			Tag: "empty",
		}
	}
	if fileErr.Line == 0 {
		return apperr.FieldError{
			Tag: "unreadable",
		}
	}
	return apperr.FieldError{
		Tag:   "unreadable_line",
		Param: strconv.Itoa(fileErr.Line),
	}
}

// toAppFieldErrors converts the failures of a column into the validation failures of a field.
func toAppFieldErrors(errs []tabular.FieldError) []apperr.FieldError {
	result := make([]apperr.FieldError, 0, len(errs))
	for _, failed := range errs {
		result = append(result, apperr.FieldError{
			Tag:   failed.Tag,
			Param: failed.Param,
		})
	}
	return result
}

// toRecordTableState returns a record table with the number of its records.
func toRecordTableState(data store.RecordTableData) RecordTableState {
	return RecordTableState{
		Table:       data.Table,
		RecordCount: len(data.Records),
	}
}
