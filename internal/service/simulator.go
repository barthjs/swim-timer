package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/store"
)

const (
	// simulatedPoolLength is the pool length the simulator assumes with an automatic pool length,
	// which the timing service derives back from the distance and the transmitted lap count.
	simulatedPoolLength = 50
	// replayPassPause is the pause before every pass of a recording, so the timing service has handled the frames
	// of the previous pass before the device clock jumps back.
	replayPassPause = 10 * time.Second
)

// SimulatorService replaces the timing device. It generates the frames of one event of the loaded start list
// with random times, or it replays the data of a recording as it arrived, all of it or the heats of one event.
type SimulatorService struct {
	logger     zerolog.Logger
	options    SimulatorOptions
	store      *store.MeetStore
	recordings *store.RecordingStore
	// onClockJump runs where the replayed device clock jumps, see TimingService.ResetClock.
	onClockJump func()

	// clock is the time source of the replay.
	clock omega.Clock
	// generatorOptions configure the frame generator, for example a clock that advances instantly in tests.
	generatorOptions []omega.Option

	// started guards Run, since two loops would interleave their frames on the same writer.
	started atomic.Bool
}

// SimulatorOptions holds the settings of a SimulatorService.
type SimulatorOptions struct {
	// Event is the event whose heats are generated, or replayed from the recording at ReplayPath.
	// 0 replays the whole recording.
	Event int
	// PoolLength is the configured pool length in meters, 0 for an automatic one.
	// It determines how many laps the simulated swimmers swim.
	PoolLength int
	// ReplayPath is the storage path of the recording to replay, empty to generate the frames of Event.
	ReplayPath string
}

// SimulatorOption configures a SimulatorService.
type SimulatorOption func(*SimulatorService)

// WithSimulatorClock replaces the system time of the generator and the replay,
// for example with a clock that advances instantly in tests. A nil clock is ignored.
func WithSimulatorClock(clock omega.Clock) SimulatorOption {
	return func(s *SimulatorService) {
		if clock == nil {
			return
		}
		s.clock = clock
		s.generatorOptions = append(s.generatorOptions, omega.WithClock(clock))
	}
}

// WithSimulatorRandSeed makes the generated times reproducible.
func WithSimulatorRandSeed(seed int64) SimulatorOption {
	return func(s *SimulatorService) {
		s.generatorOptions = append(s.generatorOptions, omega.WithRandSeed(seed))
	}
}

// NewSimulatorService creates a SimulatorService. onClockJump runs where a replayed recording makes the device clock
// jump, at the start of every pass and every recorded session. It may be nil.
func NewSimulatorService(
	logger zerolog.Logger,
	options SimulatorOptions,
	meetStore *store.MeetStore,
	recordingStore *store.RecordingStore,
	onClockJump func(),
	opts ...SimulatorOption,
) *SimulatorService {
	s := &SimulatorService{
		logger:      logger,
		options:     options,
		store:       meetStore,
		recordings:  recordingStore,
		onClockJump: onClockJump,
		clock:       omega.SystemClock(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run writes the frames of the event's heats, or the data of the replayed recording, to w in a loop until ctx is
// canceled. Generating frames needs the loaded meet data, otherwise there is nothing to simulate.
// It may only be called once per SimulatorService, a second call logs an error and returns.
func (s *SimulatorService) Run(ctx context.Context, w io.Writer) {
	if !s.started.CompareAndSwap(false, true) {
		s.logger.Error().Msg("simulator is already running")
		return
	}

	if s.options.ReplayPath != "" {
		s.replay(ctx, w)
		return
	}
	s.generate(ctx, w)
}

// generate writes the frames of the event's heats with random times to w in a loop until ctx is canceled.
func (s *SimulatorService) generate(ctx context.Context, w io.Writer) {
	starts := s.store.ListStartEntries()
	if len(starts) == 0 {
		s.logger.Error().Msg("no start entries found in meet data")
		return
	}

	event, eventFound := s.store.GetEvent(s.options.Event)

	// Group the lanes by heat. With the event known, only its phase is replayed.
	// Withdrawn entries have no heat and are left out.
	heatsMap := make(map[int][]int)
	for _, start := range starts {
		if start.Withdrawn() {
			continue
		}
		if eventFound && start.PhaseID != event.PhaseID {
			continue
		}
		if start.Event == s.options.Event {
			heatsMap[start.Heat] = append(heatsMap[start.Heat], start.Lane)
		}
	}

	if len(heatsMap) == 0 {
		s.logger.Error().Int("event", s.options.Event).Msg("no start entries found for event in meet data")
		return
	}

	heatNums := slices.Collect(maps.Keys(heatsMap))
	slices.Sort(heatNums)

	// The start list counts heats from 0, the Omega protocol from 1
	heats := make([]omega.SimulatedHeat, 0, len(heatNums))
	for _, heatNum := range heatNums {
		heats = append(heats, omega.SimulatedHeat{
			Number: heatNum + 1,
			Lanes:  heatsMap[heatNum],
		})
	}

	lapCount := s.lapCount(event, eventFound)

	omega.NewSimulator(s.logger, w, s.generatorOptions...).Run(ctx, s.options.Event, lapCount, heats)
}

// lapCount returns the number of laps of the event in the simulated pool,
// at least 1, and 2 if the event or its distance is unknown.
func (s *SimulatorService) lapCount(event *domain.Event, eventFound bool) int {
	if !eventFound || event.Distance == nil || event.Distance.Length <= 0 {
		return 2
	}

	poolLength := s.options.PoolLength
	if poolLength <= 0 {
		poolLength = simulatedPoolLength
	}

	return max(1, event.Distance.Length/poolLength)
}

// replay writes the data of the recording at the replay path to w in a loop until ctx is canceled,
// with the pauses between the reads as they were recorded. With an event, it writes only the time windows
// of the heats of that event, see replayWindows. It stops with an error in the log if the recording
// cannot be read or holds nothing to replay.
func (s *SimulatorService) replay(ctx context.Context, w io.Writer) {
	logger := s.logger.With().Str("recording", s.options.ReplayPath).Int("event", s.options.Event).Logger()
	logger.Info().Msg("replaying the recording")

	entries, err := s.readRecording(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read the recording")
		return
	}

	var selected []bool
	if s.options.Event > 0 {
		selected = replayWindows(entries, s.options.Event)
	} else {
		selected = replayAll(entries)
	}
	first := slices.Index(selected, true)
	if first < 0 {
		logger.Error().Msg("the recording holds no data to replay")
		return
	}
	// The pass ends with the last selected line instead of waiting for the lines after it
	last := len(selected) - 1
	for !selected[last] {
		last--
	}

	for {
		if !s.wait(ctx, replayPassPause) {
			return
		}
		if err := s.replayPass(ctx, logger, w, entries[:last+1], selected, first); err != nil {
			logger.Error().Err(err).Msg("failed to replay the recording")
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// readRecording reads every entry of the recording at the replay path.
func (s *SimulatorService) readRecording(ctx context.Context) ([]recording.Entry, error) {
	file, err := s.recordings.Get(ctx, s.options.ReplayPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := recording.NewReader(file)
	var entries []recording.Entry
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
}

// replayAll selects every data line of a recording.
func replayAll(entries []recording.Entry) []bool {
	selected := make([]bool, len(entries))
	for i, entry := range entries {
		selected[i] = entry.Type == recording.TypeData
	}
	return selected
}

// replayWindows selects the data lines of the heats of an event. A heat spans the reads from the one that
// brought the first byte of its first frame to the one that brought the last byte of its last frame, including noise
// and the frames of other events in between. Every load frame ends the heat, since the device then shows another one,
// so a frame of the heat after it, such as a late correction, starts a short window of its own.
// Every session is decoded on its own, like the timing service does.
func replayWindows(entries []recording.Entry, event int) []bool {
	selected := make([]bool, len(entries))
	source := &windowSource{
		entries: entries,
	}

	for source.beginSession() {
		// open is the window of the heat in progress, if active
		var open [2]int
		heat := 0
		active := false
		closeWindow := func() {
			if !active {
				return
			}
			for i := open[0]; i <= open[1]; i++ {
				selected[i] = entries[i].Type == recording.TypeData
			}
			active = false
		}

		decoder := omega.NewDecoder(source)
		for {
			frame, err := decoder.Next()
			if err != nil {
				break
			}
			if frame.Kind() == omega.KindLoad {
				closeWindow()
			}
			frameEvent, eventErr := frame.Event()
			frameHeat, heatErr := frame.Heat()
			if eventErr != nil || heatErr != nil || frameEvent != event {
				continue
			}

			first := source.lineAt(source.offset - omega.FrameSize)
			last := source.lineAt(source.offset - 1)
			if !active || frameHeat != heat {
				closeWindow()
				open[0] = first
				heat = frameHeat
				active = true
			}
			open[1] = last
		}
		closeWindow()
	}
	return selected
}

// windowSource hands the data lines of one session to a decoder one byte per read, so the position
// of the last byte of a frame is known when the decoder returns it, see replayWindows.
type windowSource struct {
	entries []recording.Entry
	// next is the index of the next entry to read.
	next int
	// chunk holds the unread bytes of the current data line.
	chunk []byte
	// offset counts the bytes of the session read so far.
	offset int
	// lines holds the index of every data line of the session and lineStarts the offset of its first byte.
	lines      []int
	lineStarts []int
}

// beginSession moves past the session line at the current position and reports whether entries are left.
func (s *windowSource) beginSession() bool {
	if s.next >= len(s.entries) {
		return false
	}
	if s.entries[s.next].Type == recording.TypeSession {
		s.next++
	}
	s.chunk = nil
	s.offset = 0
	s.lines = s.lines[:0]
	s.lineStarts = s.lineStarts[:0]
	return true
}

// Read returns the next byte of the session and io.EOF at the next session line or the end of the recording.
func (s *windowSource) Read(p []byte) (int, error) {
	for len(s.chunk) == 0 {
		if s.next >= len(s.entries) || s.entries[s.next].Type == recording.TypeSession {
			return 0, io.EOF
		}
		entry := s.entries[s.next]
		if entry.Type == recording.TypeData {
			s.chunk = entry.Data
			s.lines = append(s.lines, s.next)
			s.lineStarts = append(s.lineStarts, s.offset)
		}
		s.next++
	}
	if len(p) == 0 {
		return 0, nil
	}

	p[0] = s.chunk[0]
	s.chunk = s.chunk[1:]
	s.offset++
	return 1, nil
}

// lineAt returns the index of the data line that holds the byte at an offset of the session.
func (s *windowSource) lineAt(offset int) int {
	position, found := slices.BinarySearch(s.lineStarts, offset)
	if !found {
		position--
	}
	return s.lines[max(position, 0)]
}

// replayPass replays the entries once, writing the selected data lines. The pauses between the entries are kept
// as they were recorded from the first selected entry on, the entries before it run at once.
// A clock set back between two entries counts as no pause.
func (s *SimulatorService) replayPass(
	ctx context.Context,
	logger zerolog.Logger,
	w io.Writer,
	entries []recording.Entry,
	selected []bool,
	first int,
) error {
	start := s.clock.Now()
	previous := entries[first].At
	var due time.Duration
	// meet holds the meet files in effect, which are checked once data is sent under them
	var meet []recording.MeetFile
	meetChecked := true
	warnedMeet := false

	for i, entry := range entries {
		if i > first && entry.At.After(previous) {
			due += entry.At.Sub(previous)
		}
		if i >= first {
			previous = entry.At
		}
		if !s.wait(ctx, due-s.clock.Now().Sub(start)) {
			return nil
		}

		switch entry.Type {
		case recording.TypeSession:
			// Every session started the application anew, which measures the clock offset anew
			if s.onClockJump != nil {
				s.onClockJump()
			}
		case recording.TypeMeet:
			meet = entry.Files
			meetChecked = false
		case recording.TypeData:
			if !selected[i] {
				continue
			}
			if !meetChecked && !warnedMeet && !s.matchesMeet(meet) {
				logger.Warn().Msg("the replayed data was recorded with other meet files than the loaded ones")
				warnedMeet = true
			}
			meetChecked = true
			if _, err := w.Write(entry.Data); err != nil {
				return fmt.Errorf("writing the replayed data: %w", err)
			}
		case recording.TypeGap:
			logger.Debug().Int64("dropped_bytes", entry.DroppedBytes).Msg("the recording lost data here")
		}
	}
	return nil
}

// wait pauses for d and reports whether ctx is still active afterward. A duration of 0 or less does not pause.
func (s *SimulatorService) wait(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}

	select {
	case <-ctx.Done():
		return false
	case <-s.clock.After(d):
		return true
	}
}

// matchesMeet reports whether recorded meet files show the replayed data like the loaded ones. Without an event,
// every file must be the same, compared by its hash. With an event, only what the overlays show of it counts:
// the event with its distance, style, category, and phase, its start entries, and their competitors.
// Recorded files that cannot be read do not match.
func (s *SimulatorService) matchesMeet(files []recording.MeetFile) bool {
	if s.options.Event <= 0 {
		return s.matchesMeetFiles(files)
	}

	contents := make([]store.MeetFileContent, 0, len(files))
	for _, file := range files {
		meetFile, ok := domain.ParseMeetFile(file.Name)
		if !ok {
			continue
		}
		contents = append(contents, store.MeetFileContent{
			File:    meetFile,
			Content: file.Content,
		})
	}
	recorded, err := store.ParseMeetFiles(contents)
	if err != nil {
		return false
	}

	return newEventView(s.store.Snapshot(), s.options.Event).equal(newEventView(recorded, s.options.Event))
}

// matchesMeetFiles reports whether recorded meet files are the ones loaded, compared by their hashes.
func (s *SimulatorService) matchesMeetFiles(files []recording.MeetFile) bool {
	loaded := make(map[string]string)
	for _, content := range s.store.ListFileContents() {
		loaded[string(content.File)] = recording.Hash(content.Content)
	}

	recorded := make(map[string]string, len(files))
	for _, file := range files {
		recorded[file.Name] = file.SHA256
	}
	return maps.Equal(loaded, recorded)
}

// eventView is what the overlays show of an event from the meet data, see matchesMeet.
type eventView struct {
	found   bool
	name    string
	phase   string
	date    string
	time    string
	heats   int
	entries []entryView
}

// entryView is what the overlays show of a start entry.
type entryView struct {
	phase     int
	heat      int
	lane      int
	relay     int
	lastName  string
	firstName string
	birthYear string
	club      string
}

// newEventView collects what the overlays show of an event from a snapshot of the meet data.
func newEventView(snapshot store.MeetSnapshot, number int) eventView {
	view := eventView{}
	for _, event := range snapshot.Events {
		if event.Number != number {
			continue
		}
		view.found = true
		view.name = event.DisplayName()
		view.phase = event.PhaseName()
		view.date = event.Date
		view.time = event.Time
		view.heats = event.HeatCount
	}

	for _, entry := range snapshot.StartEntries {
		if entry.Event != number {
			continue
		}
		item := entryView{
			phase: entry.PhaseID,
			heat:  entry.Heat,
			lane:  entry.Lane,
			relay: entry.Relais,
		}
		if entry.Competitor != nil {
			item.lastName = entry.Competitor.LastName
			item.firstName = entry.Competitor.FirstName
			item.birthYear = entry.Competitor.BirthYear
			item.club = entry.Competitor.Club()
		}
		view.entries = append(view.entries, item)
	}
	// The snapshot keeps the order of the file, which the timing software may change between exports
	slices.SortFunc(view.entries, compareEntryViews)

	return view
}

// compareEntryViews orders start entries by phase, heat, lane, relay position, and name.
func compareEntryViews(a, b entryView) int {
	return cmp.Or(
		cmp.Compare(a.phase, b.phase),
		cmp.Compare(a.heat, b.heat),
		cmp.Compare(a.lane, b.lane),
		cmp.Compare(a.relay, b.relay),
		cmp.Compare(a.lastName, b.lastName),
		cmp.Compare(a.firstName, b.firstName),
	)
}

// equal reports whether two views of an event show the same.
func (v eventView) equal(other eventView) bool {
	return v.found == other.found &&
		v.name == other.name &&
		v.phase == other.phase &&
		v.date == other.date &&
		v.time == other.time &&
		v.heats == other.heats &&
		slices.Equal(v.entries, other.entries)
}
