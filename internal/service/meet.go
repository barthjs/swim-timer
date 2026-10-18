package service

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

// MeetOverview is the whole meet in a stable order, with heats counted from 1
// as the device counts them and without withdrawn start entries.
type MeetOverview struct {
	LoadedAt     time.Time
	Events       []*domain.Event
	StartEntries []MeetStartEntry
	Competitors  []*domain.Competitor
}

// MeetStartEntry is a start entry of the overview, with the heat counted from 1.
type MeetStartEntry struct {
	Event         int
	Phase         int
	Heat          int
	Lane          int
	RelayPosition int
	CompetitorID  int
}

// MeetEventList is every event of the meet in the order of the program.
type MeetEventList struct {
	LoadedAt time.Time
	Events   []*domain.Event
}

// MeetCompetitorList is every competitor of the meet, ordered by name.
type MeetCompetitorList struct {
	LoadedAt    time.Time
	Competitors []*domain.Competitor
}

// MeetStartEntryList is the start list of the meet in the order of the heats,
// with heats counted from 1 and without withdrawn start entries.
type MeetStartEntryList struct {
	LoadedAt     time.Time
	StartEntries []MeetStartEntry
}

// MeetEvent is one event of the meet.
type MeetEvent struct {
	LoadedAt time.Time
	Event    *domain.Event
}

// MeetHeat is an event and the start entries of one of its heats, ordered by lane and relay position.
// Each start entry references its competitor, nil if the competitor does not exist.
type MeetHeat struct {
	LoadedAt     time.Time
	Event        *domain.Event
	StartEntries []*domain.StartEntry
}

// MeetDataError reports a meet file whose content cannot be read.
type MeetDataError = store.ParseError

// MeetService manages the Swiss Timing export files the meet data is read from.
type MeetService struct {
	logger      zerolog.Logger
	store       *store.MeetStore
	broadcaster broadcaster
	// onReload runs after every successful reload, nil if nothing has to follow it.
	onReload func()
}

// NewMeetService creates a MeetService. onReload runs after every successful reload,
// so the recording of the timing data keeps the meet files in use. It may be nil.
func NewMeetService(logger zerolog.Logger, broadcaster broadcaster, store *store.MeetStore, onReload func()) *MeetService {
	return &MeetService{
		logger:      logger,
		store:       store,
		broadcaster: broadcaster,
		onReload:    onReload,
	}
}

// ListFiles reports for every meet file the name it is stored under, or an empty name if it is missing.
func (s *MeetService) ListFiles(ctx context.Context) (map[domain.MeetFile]string, error) {
	return s.store.ListFiles(ctx)
}

// ListEvents returns every event in the order of the program, see MeetEventList.
func (s *MeetService) ListEvents() MeetEventList {
	snapshot := s.store.Snapshot()
	return MeetEventList{
		LoadedAt: snapshot.LoadedAt,
		Events:   sortedEvents(snapshot.Events),
	}
}

// ListCompetitors returns every competitor ordered by name, see MeetCompetitorList.
func (s *MeetService) ListCompetitors() MeetCompetitorList {
	snapshot := s.store.Snapshot()
	return MeetCompetitorList{
		LoadedAt:    snapshot.LoadedAt,
		Competitors: sortedCompetitors(snapshot.Competitors),
	}
}

// ListStartEntries returns the start list in the order of the heats, see MeetStartEntryList.
func (s *MeetService) ListStartEntries() MeetStartEntryList {
	snapshot := s.store.Snapshot()
	return MeetStartEntryList{
		LoadedAt:     snapshot.LoadedAt,
		StartEntries: sortedStartEntries(snapshot.StartEntries),
	}
}

// GetOverview returns the whole meet in a stable order, see MeetOverview.
func (s *MeetService) GetOverview() MeetOverview {
	snapshot := s.store.Snapshot()
	return MeetOverview{
		LoadedAt:     snapshot.LoadedAt,
		Events:       sortedEvents(snapshot.Events),
		StartEntries: sortedStartEntries(snapshot.StartEntries),
		Competitors:  sortedCompetitors(snapshot.Competitors),
	}
}

// GetEvent returns the event with the given number, or apperr.ErrNotFound if it does not exist.
func (s *MeetService) GetEvent(number int) (MeetEvent, error) {
	snapshot := s.store.Snapshot()
	event, ok := findEvent(snapshot.Events, number)
	if !ok {
		return MeetEvent{}, apperr.ErrNotFound
	}

	return MeetEvent{
		LoadedAt: snapshot.LoadedAt,
		Event:    event,
	}, nil
}

// GetHeat returns an event and the start entries of its heat counted from 1, see MeetHeat.
// An unknown event or a heat outside the heats of the event returns apperr.ErrNotFound.
func (s *MeetService) GetHeat(number, heat int) (MeetHeat, error) {
	data, ok := s.store.GetHeat(number, heat-1)
	if !ok || heat < 1 || heat > data.Event.HeatCount {
		return MeetHeat{}, apperr.ErrNotFound
	}

	// The store shares its slice, which keeps the order of the file and decides between entries sharing a lane
	entries := make([]*domain.StartEntry, 0, len(data.StartEntries))
	entries = append(entries, data.StartEntries...)
	slices.SortStableFunc(entries, compareHeatStartEntries)

	return MeetHeat{
		LoadedAt:     data.LoadedAt,
		Event:        data.Event,
		StartEntries: entries,
	}, nil
}

// SaveFile stores a meet file without reloading the meet data,
// so several files can be replaced before Reload reads them together.
func (s *MeetService) SaveFile(ctx context.Context, file domain.MeetFile, data []byte) error {
	return s.store.SaveFile(ctx, file, bytes.NewReader(data), int64(len(data)))
}

// Reload reads all meet files again and notifies the open pages, so they show the new data.
// A file that cannot be read fails with a MeetDataError, and the previous meet data stays in use.
func (s *MeetService) Reload(ctx context.Context) error {
	if err := s.store.LoadAll(ctx); err != nil {
		return err
	}

	if s.onReload != nil {
		s.onReload()
	}
	s.notifyReload()
	return nil
}

// notifyReload broadcasts that the meet data changed. A failed notification is only logged,
// since pages still see the new data on their next fetch.
func (s *MeetService) notifyReload() {
	message, err := json.Marshal(domain.NewMessage(domain.MessageMeet, nil))
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to encode meet reload message")
		return
	}
	s.broadcaster.Broadcast(message)
}

// sortedEvents returns the events in the order of the program.
func sortedEvents(snapshotEvents []*domain.Event) []*domain.Event {
	events := slices.Clone(snapshotEvents)
	slices.SortFunc(events, compareEvents)
	return events
}

// sortedStartEntries returns the start entries that are not withdrawn in the order of the heats, with heats counted from 1.
func sortedStartEntries(snapshotEntries []*domain.StartEntry) []MeetStartEntry {
	entries := make([]MeetStartEntry, 0, len(snapshotEntries))
	for _, entry := range snapshotEntries {
		if entry.Withdrawn() {
			continue
		}
		entries = append(entries, MeetStartEntry{
			Event:         entry.Event,
			Phase:         entry.PhaseID,
			Heat:          entry.Heat + 1,
			Lane:          entry.Lane,
			RelayPosition: entry.Relais,
			CompetitorID:  entry.CompetitorID,
		})
	}
	// The snapshot keeps the order of the file, which decides between entries sharing a lane
	slices.SortStableFunc(entries, compareStartEntries)
	return entries
}

// sortedCompetitors returns the competitors ordered by name.
func sortedCompetitors(snapshotCompetitors []*domain.Competitor) []*domain.Competitor {
	competitors := slices.Clone(snapshotCompetitors)
	names := collate.New(language.Und)
	slices.SortFunc(competitors, func(a, b *domain.Competitor) int { return compareCompetitors(names, a, b) })
	return competitors
}

// findEvent returns the event with the given number among events.
func findEvent(events []*domain.Event, number int) (*domain.Event, bool) {
	for _, event := range events {
		if event.Number == number {
			return event, true
		}
	}
	return nil, false
}

// compareEvents orders events by their scheduled date, then by their position in the file,
// which follows the program also for events without a scheduled time.
func compareEvents(a, b *domain.Event) int {
	return cmp.Or(strings.Compare(a.ISODate(), b.ISODate()), cmp.Compare(a.Order, b.Order))
}

// compareStartEntries orders start entries by event, phase, heat, lane, and relay position.
func compareStartEntries(a, b MeetStartEntry) int {
	return cmp.Or(
		cmp.Compare(a.Event, b.Event),
		cmp.Compare(a.Phase, b.Phase),
		cmp.Compare(a.Heat, b.Heat),
		cmp.Compare(a.Lane, b.Lane),
		cmp.Compare(a.RelayPosition, b.RelayPosition),
	)
}

// compareHeatStartEntries orders the start entries of one heat by lane and relay position.
func compareHeatStartEntries(a, b *domain.StartEntry) int {
	return cmp.Or(cmp.Compare(a.Lane, b.Lane), cmp.Compare(a.Relais, b.Relais))
}

// compareCompetitors orders competitors by name and id, sorting letters with accents next to their base letter.
func compareCompetitors(names *collate.Collator, a, b *domain.Competitor) int {
	return cmp.Or(names.CompareString(a.LastName, b.LastName), names.CompareString(a.FirstName, b.FirstName), cmp.Compare(a.ID, b.ID))
}
