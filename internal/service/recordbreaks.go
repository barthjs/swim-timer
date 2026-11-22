package service

import (
	"cmp"
	"context"
	"crypto/rand"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

// RecordCheckInput is a heat whose times are checked for records.
type RecordCheckInput struct {
	Event *domain.Event
	Heat  int
	// PoolLength is the pool length in meters, which selects the course of the records.
	PoolLength int
	Lanes      []RecordCheckLane
}

// RecordCheckLane is a lane of a heat with a finish and a rank.
type RecordCheckLane struct {
	Lane int
	// Time is the finish time in milliseconds.
	Time       int
	Competitor *domain.Competitor
}

// RecordBreakService checks the heats for records and manages the records set at the meet.
type RecordBreakService struct {
	logger     zerolog.Logger
	store      *store.RecordBreakStore
	tableStore *store.RecordTableStore
	// onSet receives the records an official heat set that it did not store before, nil if nothing has to follow.
	onSet func(breaks []domain.RecordBreak)
	now   func() time.Time
	// writes tracks the writes of the checked heats, which Wait waits for.
	writes sync.WaitGroup
}

// NewRecordBreakService creates a RecordBreakService whose optional onSet receives the records
// an official heat set or equalled that it had not stored before.
func NewRecordBreakService(
	logger zerolog.Logger,
	store *store.RecordBreakStore,
	tableStore *store.RecordTableStore,
	onSet func(breaks []domain.RecordBreak),
) *RecordBreakService {
	return &RecordBreakService{
		logger:     logger,
		store:      store,
		tableStore: tableStore,
		onSet:      onSet,
		now:        time.Now,
	}
}

// List returns every record set at the meet, the most recent first.
func (s *RecordBreakService) List(ctx context.Context) ([]domain.RecordBreak, error) {
	return s.store.List(ctx)
}

// Delete removes the record break with the given id, or returns apperr.ErrNotFound if it does not exist.
// The next heats are then checked against the record that stood before it.
func (s *RecordBreakService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Clear removes every record break.
func (s *RecordBreakService) Clear(ctx context.Context) error {
	return s.store.Clear(ctx)
}

// Reload reads the records set at the meet again and keeps the previous ones when the file fails with a DataFileError.
func (s *RecordBreakService) Reload(ctx context.Context) error {
	return s.store.LoadAll(ctx)
}

// Check returns the records the lanes of an official heat broke or equalled by lane
// and stores them in place of earlier ones in the background, see Wait.
func (s *RecordBreakService) Check(ctx context.Context, input RecordCheckInput) (map[int][]domain.LaneRecord, error) {
	// A record the heat stored before keeps its id and time, and only the others go to onSet
	if input.Event == nil {
		return map[int][]domain.LaneRecord{}, nil
	}

	var results map[int][]domain.LaneRecord
	var fresh []domain.RecordBreak
	changed, err := s.store.ReplaceHeat(ctx, input.Event.Number, input.Heat, func(previous, others []domain.RecordBreak) []domain.RecordBreak {
		var added []domain.RecordBreak
		results, added = s.evaluate(input, earlierBreaks(previous, others, s.now()))
		fresh = nil
		for i, b := range added {
			index := slices.IndexFunc(previous, func(p domain.RecordBreak) bool { return sameRecordBreak(p, b) })
			if index < 0 {
				fresh = append(fresh, b)
				continue
			}
			added[i].ID = previous[index].ID
			added[i].BrokenAt = previous[index].BrokenAt
		}
		return added
	})
	if err != nil {
		return nil, err
	}

	if changed {
		s.flush()
	}
	if s.onSet != nil && len(fresh) > 0 {
		s.onSet(fresh)
	}
	return results, nil
}

// Preview finds the records the finished lanes of a running heat broke or equalled like Check,
// without storing them, so the overlays can show a record as soon as its lane touches.
func (s *RecordBreakService) Preview(ctx context.Context, input RecordCheckInput) (map[int][]domain.LaneRecord, error) {
	if input.Event == nil {
		return map[int][]domain.LaneRecord{}, nil
	}
	breaks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	var previous, others []domain.RecordBreak
	for _, b := range breaks {
		if b.Event == input.Event.Number && b.Heat == input.Heat {
			previous = append(previous, b)
		} else {
			others = append(others, b)
		}
	}
	results, _ := s.evaluate(input, earlierBreaks(previous, others, s.now()))
	return results, nil
}

// Wait blocks until the record breaks of the checked heats are written.
func (s *RecordBreakService) Wait() {
	s.writes.Wait()
}

// flush writes the record breaks in the background. The writes run one after the other in the store,
// and one that finds a newer state written already does nothing.
func (s *RecordBreakService) flush() {
	s.writes.Go(func() {
		// The write outlives the check that started it, so a canceled check never loses its records
		if err := s.store.Flush(context.Background()); err != nil {
			s.logger.Error().Err(err).Msg("failed to save the record breaks")
		}
	})
}

// earlierBreaks returns the record breaks of the other heats that were set before the records of a heat,
// so a heat checked again after a correction is not checked against the records later heats set against its own.
// A heat without stored records is checked against every record set so far.
func earlierBreaks(previous, others []domain.RecordBreak, now time.Time) []domain.RecordBreak {
	cutoff := now
	for _, b := range previous {
		if b.BrokenAt.Before(cutoff) {
			cutoff = b.BrokenAt
		}
	}
	return slices.DeleteFunc(slices.Clone(others), func(b domain.RecordBreak) bool { return !b.BrokenAt.Before(cutoff) })
}

// evaluate returns the records the lanes of a heat broke or equalled by lane, and the record breaks they set,
// see the page concepts/records.
func (s *RecordBreakService) evaluate(
	input RecordCheckInput,
	others []domain.RecordBreak,
) (map[int][]domain.LaneRecord, []domain.RecordBreak) {
	// Only individual events in a pool of 25 or 50 m are checked.
	// A lane counts for the first enabled table in which it breaks or equals a record of its discipline
	// for the age of the competitor, the year of the event minus the year of birth, and there for every age range it breaks.
	// A record set earlier at the meet replaces the record of its table while it is faster. The lanes are checked
	// from the fastest time on, so of two lanes under a record only the faster one sets it, and an equal time equals it.
	results := make(map[int][]domain.LaneRecord)
	course, ok := domain.CourseOf(input.PoolLength)
	if !ok || input.Event == nil || input.Event.Distance == nil || input.Event.Style == nil || input.Event.Distance.Relay > 1 {
		return results, nil
	}

	tables, _ := s.tableStore.List()
	year := eventYear(input.Event, s.now())
	lanes := slices.Clone(input.Lanes)
	slices.SortFunc(lanes, func(a, b RecordCheckLane) int {
		return cmp.Or(cmp.Compare(a.Time, b.Time), cmp.Compare(a.Lane, b.Lane))
	})

	var added []domain.RecordBreak
	for _, lane := range lanes {
		if lane.Competitor == nil || lane.Time <= 0 {
			continue
		}
		discipline := domain.RecordDiscipline{
			Course:     course,
			Gender:     strings.ToUpper(lane.Competitor.AbCat),
			RelayCount: 1,
			Distance:   input.Event.Distance.Length / max(input.Event.Distance.Relay, 1),
			Stroke:     strings.ToUpper(input.Event.Style.Abbreviation),
		}
		age := competitorAge(lane.Competitor, year)

		for _, data := range tables {
			if !data.Table.Enabled {
				continue
			}
			breaks := s.checkTable(data, discipline, age, lane, input, slices.Concat(others, added))
			if len(breaks) == 0 {
				continue
			}
			added = append(added, breaks...)
			for _, b := range breaks {
				results[lane.Lane] = append(results[lane.Lane], toLaneRecord(b))
			}
			break
		}
	}
	return results, added
}

// checkTable returns the record breaks of a lane in one table, one per age range of its discipline
// that covers the age of the competitor and whose record the lane broke or equalled.
func (s *RecordBreakService) checkTable(
	data store.RecordTableData,
	discipline domain.RecordDiscipline,
	age int,
	lane RecordCheckLane,
	input RecordCheckInput,
	earlier []domain.RecordBreak,
) []domain.RecordBreak {
	// The fastest record of every age range of the discipline, a table may list a range more than once
	var current []domain.Record
	consider := func(record domain.Record) {
		if record.Discipline() != discipline || !record.Covers(age) {
			return
		}
		index := slices.IndexFunc(current, func(r domain.Record) bool { return r.SameRange(&record) })
		if index < 0 {
			current = append(current, record)
			return
		}
		if record.Time < current[index].Time {
			current[index] = record
		}
	}
	for _, record := range data.Records {
		consider(record)
	}
	for _, b := range earlier {
		if b.TableID == data.Table.ID {
			consider(b.Record)
		}
	}

	hundredths := lane.Time / 10
	var breaks []domain.RecordBreak
	for _, previous := range current {
		if hundredths > previous.Time {
			continue
		}
		record := previous
		record.Time = hundredths
		record.Approved = false
		record.Name = competitorName(lane.Competitor)
		record.Club = lane.Competitor.Club()
		record.Place = ""
		record.Date = input.Event.ISODate()
		record.BirthYear, _ = strconv.Atoi(strings.TrimSpace(lane.Competitor.BirthYear))

		breaks = append(breaks, domain.RecordBreak{
			ID:           strings.ToLower(rand.Text()),
			TableID:      data.Table.ID,
			TableName:    data.Table.Name,
			Abbreviation: data.Table.Abbreviation,
			Event:        input.Event.Number,
			Heat:         input.Heat,
			Lane:         lane.Lane,
			Equalled:     hundredths == previous.Time,
			Record:       record,
			Previous:     previous,
			BrokenAt:     s.now(),
		})
	}
	return breaks
}

// sameRecordBreak reports whether two record breaks of a heat are the same record of the same lane and time.
func sameRecordBreak(a, b domain.RecordBreak) bool {
	return a.TableID == b.TableID &&
		a.Lane == b.Lane &&
		a.Record.SameRange(&b.Record) &&
		a.Record.Time == b.Record.Time &&
		a.Equalled == b.Equalled
}

// eventYear returns the year of the scheduled date of an event, or the year of now without one.
func eventYear(event *domain.Event, now time.Time) int {
	date, err := time.Parse(time.DateOnly, event.ISODate())
	if err != nil {
		return now.Year()
	}
	return date.Year()
}

// competitorAge returns the age of a competitor in the given year, 0 without a year of birth.
func competitorAge(competitor *domain.Competitor, year int) int {
	birthYear, err := strconv.Atoi(strings.TrimSpace(competitor.BirthYear))
	if err != nil || birthYear <= 0 || birthYear > year {
		return 0
	}
	return year - birthYear
}

// competitorName returns the name of a competitor as the record tables write it, last name first.
func competitorName(competitor *domain.Competitor) string {
	if competitor.FirstName == "" {
		return competitor.LastName
	}
	return competitor.LastName + ", " + competitor.FirstName
}

// toLaneRecord returns a record break as the overlays show it.
func toLaneRecord(b domain.RecordBreak) domain.LaneRecord {
	return domain.LaneRecord{
		Abbreviation: b.Abbreviation,
		TableName:    b.TableName,
		AgeFrom:      b.Record.AgeFrom,
		AgeTo:        b.Record.AgeTo,
		Equalled:     b.Equalled,
		// The times of the lanes are milliseconds, while a record keeps hundredths of a second
		PreviousTime: b.Previous.Time * 10,
		PreviousName: b.Previous.Name,
	}
}
