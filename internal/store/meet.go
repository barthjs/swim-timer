package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/tabular"
)

const (
	// meetFolder is the storage folder the meet files are kept in.
	meetFolder = "meet"
	// meetExtension is the extension the timing software exports the meet files with,
	// and the one an uploaded meet file is stored with.
	meetExtension = ".TXT"
	// meetSeparator separates the values of the meet files.
	meetSeparator = ";"
)

var (
	// meetExtensions are the extensions a meet file is looked up with, in order.
	// The storage can be case-sensitive, and a file copied into the folder by hand may be lowercase.
	meetExtensions = []string{meetExtension, ".txt"}
)

// MeetStore holds the meet data read from the Swiss Timing export files in memory and is safe for concurrent use.
type MeetStore struct {
	storage media.Storage

	// loadMu serializes LoadAll, so a slower reload never replaces the data of a later one.
	loadMu sync.Mutex
	mu     sync.RWMutex
	data   *meetData
	// loadedAt is when LoadAll last succeeded, zero before that.
	loadedAt time.Time
}

// MeetSnapshot is one consistent view of all meet data, as read by a single LoadAll.
// Its slices are new, but the entries are shared with the store and must not be modified.
type MeetSnapshot struct {
	LoadedAt     time.Time
	Events       []*domain.Event
	StartEntries []*domain.StartEntry
	Competitors  []*domain.Competitor
}

// meetData is one consistent set of meet data, read from the files together.
type meetData struct {
	categories  map[string]*domain.Category // by Abbreviation
	distances   map[int]*domain.Distance    // by ID
	competitors map[int]*domain.Competitor  // by ID
	events      map[int]*domain.Event       // by Number
	phases      map[int]*domain.Phase       // by ID
	starts      []*domain.StartEntry
	startsIdx   map[string][]*domain.StartEntry // by "event-phase-heat"
	styles      map[int]*domain.Style           // by ID
	// firstLane is the number of the first lane of the pool, 0 if a start entry uses lane 0, otherwise 1.
	firstLane int
	// files holds the content of every loaded file as it was read.
	files map[domain.MeetFile][]byte
}

// NewMeetStore creates an empty MeetStore. LoadAll reads the meet files.
func NewMeetStore(storage media.Storage) *MeetStore {
	return &MeetStore{
		storage: storage,
		data:    newMeetData(),
	}
}

// ListStartEntries returns a slice of all start entries.
func (s *MeetStore) ListStartEntries() []*domain.StartEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.starts
}

// ListFiles reports for every meet file the name it is stored under,
// including the extension it is read with, or an empty name if it does not exist.
func (s *MeetStore) ListFiles(ctx context.Context) (map[domain.MeetFile]string, error) {
	names := make(map[domain.MeetFile]string, len(domain.MeetFiles()))
	for _, file := range domain.MeetFiles() {
		p, ok, err := s.findFile(ctx, file)
		if err != nil {
			return nil, err
		}
		if ok {
			names[file] = path.Base(p)
		}
	}
	return names, nil
}

// MeetFileContent is a loaded meet file with its content as it was read.
type MeetFileContent struct {
	File    domain.MeetFile
	Content []byte
}

// ListFileContents returns the content of every meet file the current meet data was read from,
// in the order of domain.MeetFiles. The contents are shared with the store and must not be modified.
func (s *MeetStore) ListFileContents() []MeetFileContent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contents := make([]MeetFileContent, 0, len(s.data.files))
	for _, file := range domain.MeetFiles() {
		content, ok := s.data.files[file]
		if !ok {
			continue
		}
		contents = append(contents, MeetFileContent{
			File:    file,
			Content: content,
		})
	}
	return contents
}

// HeatData is an event and the start entries of one of its heats, read together from one load.
type HeatData struct {
	LoadedAt     time.Time
	Event        *domain.Event
	StartEntries []*domain.StartEntry
	// FirstLane is the number of the first lane of the pool, 0 if any start entry uses lane 0, otherwise 1.
	FirstLane int
}

// GetHeat returns an event and the start entries of its 0-based heat in the phase of the event.
// Both are read under one lock, so they always belong to the same LoadAll, whose time LoadedAt holds.
func (s *MeetStore) GetHeat(event, heat int) (HeatData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meetEvent, ok := s.data.events[event]
	if !ok {
		return HeatData{
			FirstLane: s.data.firstLane,
		}, false
	}
	return HeatData{
		LoadedAt:     s.loadedAt,
		Event:        meetEvent,
		StartEntries: s.data.startsIdx[startsKey(event, meetEvent.PhaseID, heat)],
		FirstLane:    s.data.firstLane,
	}, true
}

// GetEvent returns the event with the given number if it exists.
func (s *MeetStore) GetEvent(number int) (*domain.Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data.events[number]
	return e, ok
}

// SaveFile stores the content of r as the given meet file, byte for byte, with size -1 when unknown.
// It first removes copies with another extension, so LoadAll reads the stored file.
func (s *MeetStore) SaveFile(ctx context.Context, file domain.MeetFile, r io.Reader, size int64) error {
	filePath := meetFolder + "/" + string(file) + meetExtension
	for _, ext := range meetExtensions {
		other := meetFolder + "/" + string(file) + ext
		if other == filePath {
			continue
		}
		if err := s.storage.Delete(ctx, other); err != nil {
			return fmt.Errorf("store: removing %s: %w", other, err)
		}
	}
	if err := s.storage.Put(ctx, filePath, r, size); err != nil {
		return fmt.Errorf("store: saving %s: %w", filePath, err)
	}
	return nil
}

// Snapshot returns all events, start entries, and competitors of the current meet data at once,
// so they always belong to the same LoadAll. The order of the slices is unspecified.
func (s *MeetStore) Snapshot() MeetSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data.snapshot(s.loadedAt)
}

// LoadAll reads all meet files and replaces the meet data with them at once, a missing file leaving its part empty.
// A file that fails to parse returns a DataFileError and keeps the previous meet data, so exports are never mixed.
func (s *MeetStore) LoadAll(ctx context.Context) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	data := newMeetData()

	for _, file := range domain.MeetFiles() {
		filePath, ok, err := s.findFile(ctx, file)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		if err := data.load(ctx, s.storage, file, filePath); err != nil {
			if dataErr, ok := errors.AsType[*DataFileError](err); ok {
				dataErr.File = path.Base(filePath)
				return dataErr
			}
			return fmt.Errorf("store: loading %s: %w", filePath, err)
		}
	}

	data.link()

	s.mu.Lock()
	s.data = data
	s.loadedAt = time.Now()
	s.mu.Unlock()

	return nil
}

// ParseMeetFiles reads meet data from the contents of meet files without changing the store,
// and fails with a DataFileError that names a file that does not parse.
func ParseMeetFiles(contents []MeetFileContent) (MeetSnapshot, error) {
	data := newMeetData()
	for _, content := range contents {
		if err := data.parse(content.File, content.Content); err != nil {
			if dataErr, ok := errors.AsType[*DataFileError](err); ok {
				dataErr.File = string(content.File)
				return MeetSnapshot{}, dataErr
			}
			return MeetSnapshot{}, fmt.Errorf("store: parsing %s: %w", content.File, err)
		}
	}
	data.link()

	return data.snapshot(time.Time{}), nil
}

// newMeetData returns empty meet data with every map allocated.
func newMeetData() *meetData {
	return &meetData{
		categories:  make(map[string]*domain.Category),
		distances:   make(map[int]*domain.Distance),
		competitors: make(map[int]*domain.Competitor),
		events:      make(map[int]*domain.Event),
		phases:      make(map[int]*domain.Phase),
		starts:      make([]*domain.StartEntry, 0),
		startsIdx:   make(map[string][]*domain.StartEntry),
		styles:      make(map[int]*domain.Style),
		firstLane:   1,
		files:       make(map[domain.MeetFile][]byte),
	}
}

// snapshot returns all events, start entries, and competitors of the meet data, see MeetSnapshot.
func (d *meetData) snapshot(loadedAt time.Time) MeetSnapshot {
	snapshot := MeetSnapshot{
		LoadedAt:     loadedAt,
		Events:       make([]*domain.Event, 0, len(d.events)),
		StartEntries: make([]*domain.StartEntry, len(d.starts)),
		Competitors:  make([]*domain.Competitor, 0, len(d.competitors)),
	}
	for _, event := range d.events {
		snapshot.Events = append(snapshot.Events, event)
	}
	copy(snapshot.StartEntries, d.starts)
	for _, competitor := range d.competitors {
		snapshot.Competitors = append(snapshot.Competitors, competitor)
	}
	return snapshot
}

// findFile returns the storage path of a meet file, trying every known extension.
func (s *MeetStore) findFile(ctx context.Context, file domain.MeetFile) (string, bool, error) {
	for _, ext := range meetExtensions {
		filePath := meetFolder + "/" + string(file) + ext
		exists, err := s.storage.Exists(ctx, filePath)
		if err != nil {
			return "", false, fmt.Errorf("store: checking %s: %w", filePath, err)
		}
		if exists {
			return filePath, true, nil
		}
	}
	return "", false, nil
}

// load reads one meet file into its part of the meet data, see parse.
func (d *meetData) load(ctx context.Context, storage media.Storage, file domain.MeetFile, filePath string) error {
	content, err := readMeetFile(ctx, storage, filePath)
	if err != nil {
		return err
	}
	return d.parse(file, content)
}

// parse reads the content of one meet file into its part of the meet data and keeps the content.
func (d *meetData) parse(file domain.MeetFile, content []byte) error {
	d.files[file] = content

	switch file {
	case domain.MeetFileCategories:
		return loadMap(content, categorySchema, d.categories, func(c *domain.Category) string { return c.Abbreviation })
	case domain.MeetFileDistances:
		return loadMap(content, distanceSchema, d.distances, func(dist *domain.Distance) int { return dist.ID })
	case domain.MeetFileCompetitors:
		return loadMap(content, competitorSchema, d.competitors, func(c *domain.Competitor) int { return c.ID })
	case domain.MeetFileEvents:
		return d.loadEvents(content)
	case domain.MeetFilePhases:
		return loadMap(content, phaseSchema, d.phases, func(p *domain.Phase) int { return p.ID })
	case domain.MeetFileStartEntries:
		starts, _, err := readCSV(content, startEntrySchema)
		if err != nil {
			return err
		}
		d.starts = append(d.starts[:0], starts...)
		return nil
	case domain.MeetFileStyles:
		return loadMap(content, styleSchema, d.styles, func(st *domain.Style) int { return st.ID })
	default:
		return nil
	}
}

// loadEvents reads the events and keeps their order in the file, which is the order of the program.
func (d *meetData) loadEvents(content []byte) error {
	events, lines, err := readCSV(content, eventSchema)
	if err != nil {
		return err
	}
	for i, event := range events {
		event.Order = i
	}
	return fillMap(events, lines, d.events, func(e *domain.Event) int { return e.Number })
}

// link resolves the references between the files and builds the starts index.
func (d *meetData) link() {
	for _, e := range d.events {
		e.Distance = d.distances[e.DistanceID]
		e.Style = d.styles[e.StyleID]
		e.Category = d.categories[e.AbCat]
		e.Phase = d.phases[e.PhaseID]
	}

	d.firstLane = 1
	for _, start := range d.starts {
		start.Competitor = d.competitors[start.CompetitorID]
		if start.Lane == 0 && !start.Withdrawn() {
			d.firstLane = 0
		}
	}

	d.buildStartsIndex()
}

// buildStartsIndex indexes the start entries by event, phase, and heat.
// The phase keeps the entries apart for an event that has several phases.
func (d *meetData) buildStartsIndex() {
	clear(d.startsIdx)
	for _, start := range d.starts {
		key := startsKey(start.Event, start.PhaseID, start.Heat)
		d.startsIdx[key] = append(d.startsIdx[key], start)
	}
}

// startsKey builds the lookup key for the starts index.
func startsKey(event, phase, heat int) string {
	return fmt.Sprintf("%d-%d-%d", event, phase, heat)
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

// loadMap loads the rows of a meet file into dest by the key keyFunc returns,
// reporting a key that occurs twice, since one of the entries would be lost.
func loadMap[T any, K comparable](content []byte, schema *tabular.Schema[T], dest map[K]*T, keyFunc func(*T) K) error {
	items, lines, err := readCSV(content, schema)
	if err != nil {
		return err
	}
	return fillMap(items, lines, dest, keyFunc)
}

// readCSV reads the rows of a meet file with values and returns the line of every row, so later checks can name it.
func readCSV[T any](content []byte, schema *tabular.Schema[T]) ([]*T, []int, error) {
	// The timing software exports Windows-1252, and a file saved again by an editor is often UTF-8,
	// so the encoding is detected from the whole file. The writer of the timing software quotes inconsistently,
	// pads the values, and ends every row with a separator, so the files are read leniently.
	format := tabular.CompleteFormat(tabular.Format{
		Separator: meetSeparator,
	}, content)

	imp, err := schema.Read(bytes.NewReader(content), schema.Identity(format), tabular.Lenient())
	if err != nil {
		return nil, nil, toDataFileError(err)
	}

	var items []*T
	var lines []int
	// The meet files are read from memory, so nothing can cancel the reading of their rows
	for row, err := range imp.Rows(context.Background()) {
		if err != nil {
			return nil, nil, toDataFileError(err)
		}
		items = append(items, row.Value)
		lines = append(lines, row.Line)
	}
	return items, lines, nil
}

// toDataFileError describes an error of reading a meet file with its position, so the admin can fix the export.
func toDataFileError(err error) error {
	if fileErr, ok := errors.AsType[*tabular.FileError](err); ok {
		return &DataFileError{
			Line:   fileErr.Line,
			Column: fileErr.Column,
			Err:    describeFileError(fileErr.Err),
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
		return &DataFileError{
			Line:   rowErr.Line,
			Column: column,
			Err:    describeFieldError(rowErr.Fields[column][0]),
		}
	}
	return err
}

// describeFileError explains a file in an encoding that cannot be read.
func describeFileError(err error) error {
	if _, ok := errors.AsType[*tabular.EncodingError](err); ok {
		return errors.New("the file is UTF-16 encoded, but a meet file has to be UTF-8 or Windows-1252")
	}
	return err
}

// describeFieldError explains the failure of a value of a meet file.
func describeFieldError(failed tabular.FieldError) error {
	switch failed.Tag {
	case "number":
		return fmt.Errorf("the value %q is not a number", failed.Value)
	case "short_row":
		return fmt.Errorf("the row has %s of %s values", failed.Value, failed.Param)
	default:
		return &failed
	}
}

// fillMap puts the items into dest by their key and reports a key that occurs twice
// with the line of its second occurrence, taken from lines, which holds the line in the file of every item.
func fillMap[T any, K comparable](items []*T, lines []int, dest map[K]*T, keyFunc func(*T) K) error {
	clear(dest)
	for i, item := range items {
		key := keyFunc(item)
		if _, exists := dest[key]; exists {
			result := &DataFileError{
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

// The schemas of the meet files name every column as the header of the export does.
var (
	categorySchema = tabular.New([]tabular.Column[domain.Category]{
		meetString("Category", func(c *domain.Category, v string) { c.Name = v }),
		meetString("AbrevCat", func(c *domain.Category, v string) { c.Abbreviation = v }),
	})
	competitorSchema = tabular.New([]tabular.Column[domain.Competitor]{
		meetInt("id", func(c *domain.Competitor, v int) { c.ID = v }),
		meetInt("bib", func(c *domain.Competitor, v int) { c.BIB = v }),
		meetString("lastname", func(c *domain.Competitor, v string) { c.LastName = v }),
		meetString("firstname", func(c *domain.Competitor, v string) { c.FirstName = v }),
		meetString("birthyear", func(c *domain.Competitor, v string) { c.BirthYear = v }),
		meetString("abNat", func(c *domain.Competitor, v string) { c.AbNat = v }),
		meetString("abCat", func(c *domain.Competitor, v string) { c.AbCat = v }),
	})
	distanceSchema = tabular.New([]tabular.Column[domain.Distance]{
		meetInt("idLength", func(d *domain.Distance, v int) { d.ID = v }),
		meetString("Longueur", func(d *domain.Distance, v string) { d.Name = v }),
		meetInt("Mlongueur", func(d *domain.Distance, v int) { d.Length = v }),
		meetInt("Relais", func(d *domain.Distance, v int) { d.Relay = v }),
	})
	eventSchema = tabular.New([]tabular.Column[domain.Event]{
		meetInt("event", func(e *domain.Event, v int) { e.Number = v }),
		meetInt("round", func(e *domain.Event, v int) { e.PhaseID = v }),
		meetInt("nbHeat", func(e *domain.Event, v int) { e.HeatCount = v }),
		meetInt("idLen", func(e *domain.Event, v int) { e.DistanceID = v }),
		meetInt("idStyle", func(e *domain.Event, v int) { e.StyleID = v }),
		meetString("abCat", func(e *domain.Event, v string) { e.AbCat = v }),
		meetString("date", func(e *domain.Event, v string) { e.Date = v }),
		meetString("time", func(e *domain.Event, v string) { e.Time = v }),
	})
	phaseSchema = tabular.New([]tabular.Column[domain.Phase]{
		meetInt("idRound", func(p *domain.Phase, v int) { p.ID = v }),
		meetString("TITLE", func(p *domain.Phase, v string) { p.Name = v }),
		meetString("RoundAbrev", func(p *domain.Phase, v string) { p.Abbreviation = v }),
	})
	startEntrySchema = tabular.New([]tabular.Column[domain.StartEntry]{
		meetInt("event", func(s *domain.StartEntry, v int) { s.Event = v }),
		meetInt("round", func(s *domain.StartEntry, v int) { s.PhaseID = v }),
		meetInt("heat", func(s *domain.StartEntry, v int) { s.Heat = v }),
		meetInt("lane", func(s *domain.StartEntry, v int) { s.Lane = v }),
		meetInt("relais", func(s *domain.StartEntry, v int) { s.Relais = v }),
		meetInt("idBib", func(s *domain.StartEntry, v int) { s.CompetitorID = v }),
	})
	styleSchema = tabular.New([]tabular.Column[domain.Style]{
		meetInt("idStyle", func(s *domain.Style, v int) { s.ID = v }),
		meetString("Style", func(s *domain.Style, v string) { s.Name = v }),
		meetString("StyleAbrév", func(s *domain.Style, v string) { s.Abbreviation = v }),
	})
)

// meetInt returns a column of a meet file that holds a whole number, zero if it is empty.
func meetInt[T any](key string, set func(*T, int)) tabular.Column[T] {
	return tabular.Field(tabular.FieldOptions[T, int]{
		Key:   key,
		Parse: tabular.ParseInt,
		Set:   set,
	})
}

// meetString returns a column of a meet file that holds text.
func meetString[T any](key string, set func(*T, string)) tabular.Column[T] {
	return tabular.Field(tabular.FieldOptions[T, string]{
		Key:   key,
		Parse: tabular.ParseString,
		Set:   set,
	})
}
