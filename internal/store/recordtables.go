package store

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/tabular"
)

const (
	// recordsFolder is the storage folder the record tables and the record breaks are kept in.
	recordsFolder = "records"
	// recordTableExtension is the extension of the file that keeps the records of a table.
	recordTableExtension = ".csv"
	// recordTableIndex is the file that keeps the record tables in their order.
	recordTableIndex = recordsFolder + "/tables.json"
)

// RecordTableFormat is the format the records of a table are kept in, with the byte order mark spreadsheets need
// to recognize UTF-8 and the semicolon they expect as the separator in many languages.
var RecordTableFormat = tabular.Format{
	Encoding:  tabular.EncodingUTF8,
	Separator: ";",
}

// RecordTableData is a record table with its records.
type RecordTableData struct {
	Table   domain.RecordTable
	Records []domain.Record
}

// RecordTableStore keeps every record table as a CSV file in the storage, named after the id of the table,
// and the tables in their order in an index next to them. It holds the tables in memory,
// as read by LoadAll and changed since, and is safe for concurrent use.
type RecordTableStore struct {
	storage media.Storage
	schema  *tabular.Schema[domain.Record]

	// loadMu serializes LoadAll and the changes, so a slower reload never replaces a later change.
	loadMu sync.Mutex
	mu     sync.RWMutex
	tables []RecordTableData
	// loaded reports whether the tables in memory are those of the storage and is guarded by loadMu.
	// It is unset before the first LoadAll and after a failed one, since a change would then write an index
	// that leaves out the tables not read.
	loaded bool
	// loadedAt is when LoadAll last succeeded, zero before that.
	loadedAt time.Time
}

// NewRecordTableStore creates an empty RecordTableStore. LoadAll reads the record tables.
func NewRecordTableStore(storage media.Storage) *RecordTableStore {
	return &RecordTableStore{
		storage: storage,
		schema:  NewRecordSchema(RecordSchemaOptions{}),
	}
}

// List returns every record table with its records in their order, and when they were loaded.
// The records are shared with the store and must not be modified.
func (s *RecordTableStore) List() ([]RecordTableData, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.tables), s.loadedAt
}

// Get returns the record table with the given id and its records, or apperr.ErrNotFound if it does not exist.
// The records are shared with the store and must not be modified.
func (s *RecordTableStore) Get(id string) (RecordTableData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.indexOf(id)
	if index < 0 {
		return RecordTableData{}, apperr.ErrNotFound
	}
	return s.tables[index], nil
}

// Update replaces the name, the abbreviation, and the state of the record table with the given id,
// or returns apperr.ErrNotFound if it does not exist.
func (s *RecordTableStore) Update(ctx context.Context, table domain.RecordTable) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	if err := s.ensureLoaded(ctx); err != nil {
		return err
	}
	index := s.indexOf(table.ID)
	if index < 0 {
		return apperr.ErrNotFound
	}

	tables := s.snapshot()
	tables[index].Table = table
	return s.commit(ctx, tables)
}

// Save writes the record table with its records, replacing an existing one in its place or adding it at the end.
// An id that cannot address a file fails with ErrInvalidName.
func (s *RecordTableStore) Save(ctx context.Context, table domain.RecordTable, records []domain.Record) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	if err := s.ensureLoaded(ctx); err != nil {
		return err
	}
	if err := s.writeRecords(ctx, table.ID, records); err != nil {
		return err
	}

	tables := s.snapshot()
	data := RecordTableData{
		Table:   table,
		Records: slices.Clone(records),
	}
	index := s.indexOf(table.ID)
	if index >= 0 {
		tables[index] = data
	} else {
		tables = append(tables, data)
	}
	if err := s.commit(ctx, tables); err != nil {
		// The file of a new table would return as an unlisted table with the next LoadAll
		if index < 0 {
			_ = s.deleteRecords(ctx, table.ID)
		}
		return err
	}
	return nil
}

// Delete removes the record table with the given id and its file, or returns apperr.ErrNotFound if it does not exist.
func (s *RecordTableStore) Delete(ctx context.Context, id string) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	if err := s.ensureLoaded(ctx); err != nil {
		return err
	}
	index := s.indexOf(id)
	if index < 0 {
		return apperr.ErrNotFound
	}

	// The file goes first, since a file left without its entry in the index would return as an unlisted table
	if err := s.deleteRecords(ctx, id); err != nil {
		return err
	}
	return s.commit(ctx, slices.Delete(s.snapshot(), index, index+1))
}

// SaveRecords replaces the records of the record table with the given id and keeps the table as it is now,
// or returns apperr.ErrNotFound if it does not exist.
func (s *RecordTableStore) SaveRecords(ctx context.Context, id string, records []domain.Record) (domain.RecordTable, error) {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	if err := s.ensureLoaded(ctx); err != nil {
		return domain.RecordTable{}, err
	}
	index := s.indexOf(id)
	if index < 0 {
		return domain.RecordTable{}, apperr.ErrNotFound
	}
	if err := s.writeRecords(ctx, id, records); err != nil {
		return domain.RecordTable{}, err
	}

	tables := s.snapshot()
	tables[index].Records = slices.Clone(records)
	return tables[index].Table, s.commit(ctx, tables)
}

// Reorder puts the record tables in the order of ids, which has to name every table once,
// or fails with apperr.ErrNotFound.
func (s *RecordTableStore) Reorder(ctx context.Context, ids []string) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	if err := s.ensureLoaded(ctx); err != nil {
		return err
	}
	current := s.snapshot()
	if len(ids) != len(current) {
		return apperr.ErrNotFound
	}
	tables := make([]RecordTableData, 0, len(ids))
	for _, id := range ids {
		index := s.indexOf(id)
		if index < 0 || slices.ContainsFunc(tables, func(t RecordTableData) bool { return t.Table.ID == id }) {
			return apperr.ErrNotFound
		}
		tables = append(tables, current[index])
	}
	return s.commit(ctx, tables)
}

// LoadAll reads the index and every record table and replaces the tables at once,
// or keeps the previous ones when a file fails with a DataFileError.
func (s *RecordTableStore) LoadAll(ctx context.Context) error {
	// A CSV file the index does not list is added as a disabled table named after the file,
	// so a table copied into the folder by hand appears. Every change reads the tables again until they load.
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	return s.load(ctx)
}

// WriteRecords writes the records of a table in the given format, with the columns of the given keys,
// or every column without any, and escapes the values a spreadsheet would read as a formula.
// It fails with apperr.ErrNotFound for an unknown table.
func (s *RecordTableStore) WriteRecords(w io.Writer, id string, format tabular.Format, columns []string) error {
	data, err := s.Get(id)
	if err != nil {
		return err
	}

	// The file is meant for a spreadsheet, which must not run a name or a place as a formula
	options := []tabular.WriteOption{
		tabular.WithColumns(columns...),
		tabular.WithFormulaEscape(),
	}
	if format.Encoding == tabular.EncodingUTF8 {
		options = append(options, tabular.WithByteOrderMark())
	}
	return s.schema.Write(w, recordPointers(data.Records), format, options...)
}

// ensureLoaded reads the tables again while the caller holds loadMu, unless the last LoadAll succeeded.
func (s *RecordTableStore) ensureLoaded(ctx context.Context) error {
	if s.loaded {
		return nil
	}
	return s.load(ctx)
}

// load reads the tables as LoadAll describes while the caller holds loadMu.
func (s *RecordTableStore) load(ctx context.Context) error {
	tables, err := s.readTables(ctx)
	if err != nil {
		s.loaded = false
		return err
	}

	s.mu.Lock()
	s.tables = tables
	s.loadedAt = time.Now()
	s.mu.Unlock()
	s.loaded = true
	return nil
}

// readTables reads the index, adds the unlisted files, and reads the records of every table.
func (s *RecordTableStore) readTables(ctx context.Context) ([]RecordTableData, error) {
	index, err := s.readIndex(ctx)
	if err != nil {
		return nil, err
	}
	index, err = s.addUnlisted(ctx, index)
	if err != nil {
		return nil, err
	}

	tables := make([]RecordTableData, 0, len(index))
	for _, table := range index {
		records, err := s.readRecords(ctx, table.ID)
		if err != nil {
			return nil, err
		}
		tables = append(tables, RecordTableData{
			Table:   table,
			Records: records,
		})
	}
	return tables, nil
}

// writeRecords writes the records of a table to its file while the caller holds loadMu.
// An id that cannot address a file fails with ErrInvalidName.
func (s *RecordTableStore) writeRecords(ctx context.Context, id string, records []domain.Record) error {
	p, err := filePath(recordsFolder, id, recordTableExtension)
	if err != nil {
		return err
	}

	var content bytes.Buffer
	if err := s.schema.Write(&content, recordPointers(records), RecordTableFormat, tabular.WithByteOrderMark()); err != nil {
		return fmt.Errorf("store: encoding record table %q: %w", id, err)
	}
	if err := s.storage.Put(ctx, p, bytes.NewReader(content.Bytes()), int64(content.Len())); err != nil {
		return fmt.Errorf("store: saving record table %q: %w", id, err)
	}
	return nil
}

// deleteRecords removes the file of a table while the caller holds loadMu. A missing file is no error.
func (s *RecordTableStore) deleteRecords(ctx context.Context, id string) error {
	p, err := filePath(recordsFolder, id, recordTableExtension)
	if err != nil {
		return err
	}
	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting record table %q: %w", id, err)
	}
	return nil
}

// indexOf returns the position of the table with the given id, -1 if it does not exist.
func (s *RecordTableStore) indexOf(id string) int {
	return slices.IndexFunc(s.tables, func(t RecordTableData) bool { return t.Table.ID == id })
}

// snapshot returns a copy of the tables in memory, which a change modifies before commit.
func (s *RecordTableStore) snapshot() []RecordTableData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.tables)
}

// commit writes the index of the tables and replaces the tables in memory.
func (s *RecordTableStore) commit(ctx context.Context, tables []RecordTableData) error {
	index := make([]domain.RecordTable, 0, len(tables))
	for _, data := range tables {
		index = append(index, data.Table)
	}

	content, err := json.Marshal(index, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("store: encoding record table index: %w", err)
	}
	if err := s.storage.Put(ctx, recordTableIndex, bytes.NewReader(content), int64(len(content))); err != nil {
		return fmt.Errorf("store: saving record table index: %w", err)
	}

	s.mu.Lock()
	s.tables = tables
	s.mu.Unlock()
	return nil
}

// readIndex returns the record tables of the index in their order, none if it does not exist.
func (s *RecordTableStore) readIndex(ctx context.Context) ([]domain.RecordTable, error) {
	content, err := media.ReadAll(ctx, s.storage, recordTableIndex)
	if errors.Is(err, media.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: reading record table index: %w", err)
	}

	var index []domain.RecordTable
	if err := json.Unmarshal(content, &index); err != nil {
		return nil, &DataFileError{
			File: path.Base(recordTableIndex),
			Err:  err,
		}
	}
	return index, nil
}

// addUnlisted appends a disabled table for every CSV file of the folder the index does not list.
func (s *RecordTableStore) addUnlisted(ctx context.Context, index []domain.RecordTable) ([]domain.RecordTable, error) {
	paths, err := s.storage.List(ctx, recordsFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing record tables: %w", err)
	}
	slices.Sort(paths)

	for _, p := range paths {
		id, ok := fileName(recordsFolder, p, recordTableExtension)
		if !ok || slices.ContainsFunc(index, func(t domain.RecordTable) bool { return t.ID == id }) {
			continue
		}
		index = append(index, domain.RecordTable{
			ID:   id,
			Name: id,
		})
	}
	return index, nil
}

// readRecords reads the records of a table, none if its file does not exist.
func (s *RecordTableStore) readRecords(ctx context.Context, id string) ([]domain.Record, error) {
	p, err := filePath(recordsFolder, id, recordTableExtension)
	if err != nil {
		return nil, &DataFileError{
			File: path.Base(recordTableIndex),
			Err:  fmt.Errorf("the id %q cannot name a file", id),
		}
	}

	content, err := media.ReadAll(ctx, s.storage, p)
	if errors.Is(err, media.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: reading record table %q: %w", id, err)
	}

	records, err := readRecordFile(content, s.schema)
	if err != nil {
		if dataErr, ok := errors.AsType[*DataFileError](err); ok {
			dataErr.File = id + recordTableExtension
		}
		return nil, err
	}
	return records, nil
}

// readRecordFile reads every record of a stored table or fails as a whole with a DataFileError that names the position.
func readRecordFile(content []byte, schema *tabular.Schema[domain.Record]) ([]domain.Record, error) {
	// A file copied into the folder by hand may leave out the optional columns
	// and use another encoding and separator, which are detected, the encoding from the whole file.
	imp, err := schema.ReadByHeader(bytes.NewReader(content), tabular.CompleteFormat(tabular.Format{}, content))
	if err != nil {
		return nil, recordDataFileError(err)
	}

	var records []domain.Record
	// The file is read from memory, so nothing can cancel the reading of its rows
	for row, err := range imp.Rows(context.Background()) {
		if err != nil {
			return nil, recordDataFileError(err)
		}
		records = append(records, *row.Value)
	}
	return records, nil
}

// recordDataFileError describes an error of reading a record file with its position.
func recordDataFileError(err error) error {
	if fileErr, ok := errors.AsType[*tabular.FileError](err); ok {
		return &DataFileError{
			Line:   fileErr.Line,
			Column: fileErr.Column,
			Err:    fileErr.Err,
		}
	}
	if mappingErr, ok := errors.AsType[*tabular.MappingError](err); ok {
		return &DataFileError{
			Line: 1,
			Err:  fmt.Errorf("the header lacks the columns %s", strings.Join(mappingErr.Columns.Keys(), ", ")),
		}
	}
	if rowErr, ok := errors.AsType[*tabular.RowError](err); ok {
		column := rowErr.Fields.Keys()[0]
		failed := rowErr.Fields[column][0]
		return &DataFileError{
			Line:   rowErr.Line,
			Column: column,
			Err:    &failed,
		}
	}
	return err
}

// recordPointers yields a pointer to every record, as the schema writes them.
func recordPointers(records []domain.Record) func(yield func(*domain.Record) bool) {
	return func(yield func(*domain.Record) bool) {
		for i := range records {
			if !yield(&records[i]) {
				return
			}
		}
	}
}

// RecordSchemaOptions holds the suggested values of the columns that depend on the meet data,
// which accept any value, since a stored table must load with any meet.
type RecordSchemaOptions struct {
	// Genders are the abbreviations of the categories, Strokes the abbreviations of the styles.
	Genders []string
	Strokes []string
}

// courseAliases are the common English and German names of the courses, which the guess of a mapping translates.
var courseAliases = map[string][]string{
	string(domain.CourseShort): {"short", "short course", "kurz", "kurzbahn", "25", "25 m", "25m"},
	string(domain.CourseLong):  {"long", "long course", "lang", "langbahn", "50", "50 m", "50m"},
}

// strokeAliases are the common English and German names and letters of the styles,
// by the abbreviation of the style in the catalog of the timing software, which the guess of a mapping translates.
var strokeAliases = map[string][]string{
	"FR": {"F", "free", "freestyle", "freistil", "kraul"},
	"BA": {"R", "back", "backstroke", "rücken", "ruecken"},
	"BR": {"B", "breast", "breaststroke", "brust"},
	"BU": {"S", "fly", "butterfly", "schmetterling", "delphin"},
	"ME": {"L", "IM", "medley", "lagen"},
}

// boolAliases are the common English and German words for yes and no, which the guess of a mapping translates.
var boolAliases = map[string][]string{
	"true":  {"yes", "y", "ja", "j", "x", "1"},
	"false": {"no", "n", "nein", "0"},
}

// recordDateLayouts are the layouts a date of a record is read in, the first one is the one it is written in.
var recordDateLayouts = []string{time.DateOnly, "02.01.2006", "2.1.2006"}

// NewRecordSchema returns the schema of the records of a table. Its keys name the columns of the stored files,
// and its aliases guess the columns of a file from a spreadsheet or another program in English or German.
func NewRecordSchema(opts RecordSchemaOptions) *tabular.Schema[domain.Record] {
	genderOptions := []tabular.ColumnOption{
		tabular.Required(),
		tabular.Aliases("Geschlecht", "Sex"),
		tabular.Normalize(strings.ToUpper),
	}
	if len(opts.Genders) > 0 {
		genderOptions = append(genderOptions, tabular.Suggestions(opts.Genders...))
	}
	// Without meet data, the styles of the catalog of the timing software are suggested
	strokes := opts.Strokes
	if len(strokes) == 0 {
		strokes = slices.Sorted(maps.Keys(strokeAliases))
	}
	strokeOptions := []tabular.ColumnOption{
		tabular.Required(),
		tabular.Aliases("Lage", "Style"),
		tabular.Normalize(strings.ToUpper),
		tabular.Suggestions(strokes...),
		tabular.ValueAliases(strokeAliases),
	}

	return tabular.New([]tabular.Column[domain.Record]{
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "course",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Course = domain.Course(v) },
			Get:    func(r *domain.Record) string { return string(r.Course) },
			Format: identity,
		}, tabular.Required(), tabular.Aliases("Bahn", "Pool"), tabular.Normalize(strings.ToUpper),
			tabular.Choices(string(domain.CourseShort), string(domain.CourseLong)), tabular.ValueAliases(courseAliases)),
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "gender",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Gender = v },
			Get:    func(r *domain.Record) string { return r.Gender },
			Format: identity,
		}, genderOptions...),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "relayCount",
			Parse:  tabular.ParseInt,
			Set:    func(r *domain.Record, v int) { r.RelayCount = v },
			Get:    func(r *domain.Record) int { return r.RelayCount },
			Format: strconv.Itoa,
		}, tabular.Aliases("Anzahl", "Relay"), tabular.Rules("min=1,max=20")),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "distance",
			Parse:  tabular.ParseInt,
			Set:    func(r *domain.Record, v int) { r.Distance = v },
			Get:    func(r *domain.Record) int { return r.Distance },
			Format: strconv.Itoa,
		}, tabular.Required(), tabular.Aliases("Laenge", "Länge", "Strecke"), tabular.Rules("min=1,max=100000")),
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "stroke",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Stroke = v },
			Get:    func(r *domain.Record) string { return r.Stroke },
			Format: identity,
		}, strokeOptions...),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "ageFrom",
			Parse:  tabular.ParseInt,
			Set:    func(r *domain.Record, v int) { r.AgeFrom = v },
			Get:    func(r *domain.Record) int { return r.AgeFrom },
			Format: strconv.Itoa,
		}, tabular.Aliases("VonAlter", "Von"), tabular.Rules("min=0,max=150")),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "ageTo",
			Parse:  tabular.ParseInt,
			Set:    func(r *domain.Record, v int) { r.AgeTo = v },
			Get:    func(r *domain.Record) int { return r.AgeTo },
			Format: strconv.Itoa,
		}, tabular.Aliases("BisAlter", "Bis"), tabular.Rules("min=0,max=150")),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "time",
			Parse:  ParseSwimTime,
			Set:    func(r *domain.Record, v int) { r.Time = v },
			Get:    func(r *domain.Record) int { return r.Time },
			Format: FormatSwimTime,
		}, tabular.Required(), tabular.Aliases("Zeit"), tabular.Rules("min=1")),
		tabular.Field(tabular.FieldOptions[domain.Record, bool]{
			Key:    "approved",
			Parse:  tabular.ParseBool,
			Set:    func(r *domain.Record, v bool) { r.Approved = v },
			Get:    func(r *domain.Record) bool { return r.Approved },
			Format: tabular.FormatBool,
		}, tabular.Aliases("Anerkannt"), tabular.Normalize(strings.ToLower), tabular.Choices("true", "false"),
			tabular.ValueAliases(boolAliases)),
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "name",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Name = v },
			Get:    func(r *domain.Record) string { return r.Name },
			Format: identity,
		}, tabular.Aliases("Holder"), tabular.Normalize(trimName), tabular.Rules("max=200")),
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "club",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Club = v },
			Get:    func(r *domain.Record) string { return r.Club },
			Format: identity,
		}, tabular.Aliases("Verein", "Nation"), tabular.Rules("max=200")),
		tabular.Field(tabular.FieldOptions[domain.Record, string]{
			Key:    "place",
			Parse:  tabular.ParseString,
			Set:    func(r *domain.Record, v string) { r.Place = v },
			Get:    func(r *domain.Record) string { return r.Place },
			Format: identity,
		}, tabular.Aliases("Ort", "City"), tabular.Rules("max=200")),
		tabular.Field(tabular.FieldOptions[domain.Record, time.Time]{
			Key:    "date",
			Parse:  tabular.ParseDate(recordDateLayouts...),
			Set:    func(r *domain.Record, v time.Time) { r.Date = v.Format(time.DateOnly) },
			Get:    recordDate,
			Format: formatRecordDate,
		}, tabular.Aliases("Datum")),
		tabular.Field(tabular.FieldOptions[domain.Record, int]{
			Key:    "birthYear",
			Parse:  tabular.ParseInt,
			Set:    func(r *domain.Record, v int) { r.BirthYear = v },
			Get:    func(r *domain.Record) int { return r.BirthYear },
			Format: formatOptionalInt,
		}, tabular.Aliases("Jg", "Jahrgang", "YearOfBirth"), tabular.Rules("min=0,max=9999")),
	},
		// A file without a column for relays holds individual records
		tabular.WithResolve(func(context.Context, map[string]string) (*domain.Record, error) {
			return &domain.Record{
				RelayCount: 1,
			}, nil
		}),
		tabular.WithValidate(func(_ context.Context, r *domain.Record) tabular.FieldErrors {
			if r.AgeFrom > 0 && r.AgeTo > 0 && r.AgeTo < r.AgeFrom {
				return tabular.FieldErrors{
					"ageTo": {
						{
							Tag:   "gtefield",
							Param: "ageFrom",
						},
					},
				}
			}
			return nil
		}),
	)
}

// identity returns a text as it is.
func identity(value string) string {
	return value
}

// trimName removes the spaces and a trailing comma around a name, which a file may leave after an empty list of names.
func trimName(value string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(value), ","))
}

// recordDate returns the date of a record, the zero time if it has none.
func recordDate(r *domain.Record) time.Time {
	date, err := time.Parse(time.DateOnly, r.Date)
	if err != nil {
		return time.Time{}
	}
	return date
}

// formatRecordDate writes a date as YYYY-MM-DD, empty for the zero time.
func formatRecordDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.DateOnly)
}

// formatOptionalInt writes a number, empty for zero.
func formatOptionalInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

// ParseSwimTime reads a swim time written as [[hours:]minutes:]seconds with a decimal point or comma,
// with up to two digits of hundredths, into hundredths of a second. It fails with the tag "swimtime".
func ParseSwimTime(raw string) (int, error) {
	invalid := &tabular.FieldError{
		Tag: "swimtime",
	}

	parts := strings.Split(strings.Replace(raw, ",", ".", 1), ":")
	if len(parts) > 3 {
		return 0, invalid
	}

	seconds, fraction, _ := strings.Cut(parts[len(parts)-1], ".")
	if len(fraction) > 2 || !isDigits(seconds) || (fraction != "" && !isDigits(fraction)) {
		return 0, invalid
	}
	hundredths, _ := strconv.Atoi((fraction + "00")[:2])
	total, _ := strconv.Atoi(seconds)
	if len(parts) > 1 && total >= 60 {
		return 0, invalid
	}

	multiplier := 60
	for i := len(parts) - 2; i >= 0; i-- {
		if !isDigits(parts[i]) {
			return 0, invalid
		}
		value, _ := strconv.Atoi(parts[i])
		// Only the leading part may exceed its unit
		if i > 0 && value >= 60 {
			return 0, invalid
		}
		total += value * multiplier
		multiplier *= 60
	}
	return total*100 + hundredths, nil
}

// FormatSwimTime writes hundredths of a second as mm:ss.hh, with the hours in front of a time of an hour or more.
func FormatSwimTime(hundredths int) string {
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	fraction := hundredths % 100
	if minutes >= 60 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", minutes/60, minutes%60, seconds, fraction)
	}
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, fraction)
}

// isDigits reports whether s is a non-empty run of ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
