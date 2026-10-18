package service

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/serial"
	"github.com/barthjs/swim-timer/internal/store"
)

// recentHeatCount bounds the heats kept with their times, so a frame for a heat shown shortly before,
// such as a late correction, shows it again with its times instead of starting it empty.
const recentHeatCount = 8

const (
	// baselineMatch is how close to the baseline the offset of a start lies when it arrived without delay.
	baselineMatch = 500 * time.Millisecond
	// baselineConfirmations is the number of starts without delay after which the baseline counts as confirmed.
	baselineConfirmations = 2
	// baselineDrift bounds how far a start may lower a confirmed baseline. The clocks drift by milliseconds per hour,
	// so a larger drop comes from a wrong time of day or a clock set during the meet.
	baselineDrift = 2 * time.Second
)

// broadcaster publishes a message to every connected overlay.
type broadcaster interface {
	Broadcast(message []byte)
}

// TimingDevice keeps the connection to the timing device open, hands its byte stream to the timing service,
// and reports the port in use and whether it is reachable. serial.Supervisor implements it,
// and tests replace it, since it reads a serial port.
type TimingDevice interface {
	Run(ctx context.Context, handle serial.Handler)
	Port() string
	Ping(ctx context.Context) error
}

// TimingOptions holds the settings of a TimingService.
type TimingOptions struct {
	// PoolLength is the pool length in meters, 0 derives it from the distance and the lap count.
	PoolLength int
	// ConfiguredPort is the port from the configuration, empty if every port found is tried.
	ConfiguredPort string
	// SimulateEvent is the event the simulator replays, 0 when a real device is read.
	SimulateEvent int
	// ReplayFile is the name of the recording the simulator replays, empty when no recording is replayed.
	ReplayFile string
}

// TimingStatus describes the connection to the timing device.
type TimingStatus struct {
	// Connected reports whether a port is currently open.
	Connected bool
	// Port is the name of the open port, empty while disconnected.
	Port string
	// ConfiguredPort is the port from the configuration, empty if every port found is tried.
	ConfiguredPort string
	// SimulateEvent is the event the simulator replays, 0 when a real device is read.
	SimulateEvent int
	// ReplayFile is the name of the recording the simulator replays, empty when no recording is replayed.
	ReplayFile string
}

// TimingService reads the frames of the timing device, turns them into the live heat shown by the overlays,
// and publishes every change. It is the only writer of the live heat:
// frames are handled one after another in the goroutine that runs Run.
type TimingService struct {
	logger      zerolog.Logger
	store       *store.MeetStore
	broadcaster broadcaster
	device      TimingDevice
	options     TimingOptions
	// recorder records the received data, nil without recording.
	recorder *recording.Writer

	// Only the frame handling goroutine accesses the fields up to started. current is the heat on display,
	// and recent holds the last heats with their times, the current one last.
	current *heatState
	recent  []*heatState
	// clock corrects the start times, see correctStart.
	clock *startClock
	// lastStart is the last start applied to a heat, nil if it carried no time of day, see startHeat.
	lastStart *appliedStart
	// heldStart is a start held back until a time of its heat confirms it, nil without one, see startHeat.
	heldStart *heldStart

	// clockReset asks the frame handling goroutine to measure the clock offset anew before the next start.
	clockReset atomic.Bool
	// started guards Run, since the live heat must only have one writer.
	started atomic.Bool

	mu   sync.RWMutex
	heat *domain.LiveHeat
}

// NewTimingService creates a TimingService. With a recorder, it also records the received data. It may be nil.
func NewTimingService(
	logger zerolog.Logger,
	options TimingOptions,
	broadcaster broadcaster,
	device TimingDevice,
	recorder *recording.Writer,
	store *store.MeetStore,
) *TimingService {
	return &TimingService{
		logger:      logger,
		store:       store,
		broadcaster: broadcaster,
		device:      device,
		options:     options,
		recorder:    recorder,
		clock:       newStartClock(logger, fixedZone(time.Now())),
		heat: &domain.LiveHeat{
			// Before the first heat there are no lanes to map, so the usual numbering is assumed
			FirstLane: 1,
			Lanes:     make(map[int]*domain.Lane),
		},
	}
}

// GetLiveHeat returns a deep copy of the live heat.
func (s *TimingService) GetLiveHeat() *domain.LiveHeat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.heat.Copy()
}

// GetStatus returns the current state of the connection to the timing device.
func (s *TimingService) GetStatus() TimingStatus {
	port := s.device.Port()

	return TimingStatus{
		Connected:      port != "",
		Port:           port,
		ConfiguredPort: s.options.ConfiguredPort,
		SimulateEvent:  s.options.SimulateEvent,
		ReplayFile:     s.options.ReplayFile,
	}
}

// Run reads and handles frames from the port the device keeps open until ctx is canceled.
// It may only be called once per TimingService, a second call logs an error and returns.
func (s *TimingService) Run(ctx context.Context) {
	if !s.started.CompareAndSwap(false, true) {
		s.logger.Error().Msg("timing service is already running")
		return
	}

	if s.recorder != nil {
		s.recorder.Start()
		defer s.recorder.Close()
		s.RecordMeet()
	}

	s.device.Run(ctx, func(ctx context.Context, r io.Reader) error {
		if s.recorder != nil {
			r = s.recorder.Reader(r)
		}
		decoder := omega.NewDecoder(r)
		for {
			frame, err := decoder.Next()
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return nil //nolint:nilerr // A canceled context ends the connection without an error.
			}
			s.handleFrame(frame)
		}
	})
}

// Snapshot returns the live heat as the init message for a newly connected overlay.
func (s *TimingService) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.Marshal(domain.NewMessage(domain.MessageInit, s.heat))
}

// RecordMeet records the meet files in use from now on, so the following data can be assigned to its meet.
// The meet service calls it after every successful reload. Without recording it does nothing.
func (s *TimingService) RecordMeet() {
	if s.recorder == nil {
		return
	}

	contents := s.store.ListFileContents()
	files := make([]recording.MeetFile, 0, len(contents))
	for _, content := range contents {
		files = append(files, recording.NewMeetFile(string(content.File), content.Content))
	}
	s.recorder.Meet(time.Now(), files)
}

// ResetClock makes the next start measure the clock offset anew, as after a start of the application.
// The replay of a recording calls it where the device clock jumps, at the start of every pass and every session.
func (s *TimingService) ResetClock() {
	s.clockReset.Store(true)
}

// handleFrame updates the live heat with a frame. Frames of a kind the live heat does not use are only logged.
func (s *TimingService) handleFrame(frame omega.Frame) {
	s.logger.Debug().Func(func(e *zerolog.Event) { e.Str("hex", frame.Hex()) }).Msg("raw frame")

	switch frame.Kind() {
	case omega.KindLoad:
		s.updateStatus(frame, domain.HeatStatusLoad)
	case omega.KindOfficial:
		s.updateStatus(frame, domain.HeatStatusOfficial)
	case omega.KindStart:
		s.startHeat(frame, s.correctStart(frame, time.Now()))
	case omega.KindIntermediate:
		s.updateTime(frame, timeIntermediate)
	case omega.KindFinish:
		s.updateTime(frame, timeFinish)
	case omega.KindButton:
		s.updateTime(frame, timeButton)
	case omega.KindUnknown:
		// Reaction times and other messages do not change the live heat
	}
}

// correctStart determines the actual start time of a heat from the device's time of day in the start frame,
// see startClock. A pending clock reset takes effect first.
func (s *TimingService) correctStart(frame omega.Frame, now time.Time) startCorrection {
	if s.clockReset.Swap(false) {
		s.clock.reset()
	}
	return s.clock.correct(frame, now)
}

// updateStatus applies a load or official frame to its heat and shows that heat. An official frame for a heat
// not on display confirms an earlier result late, so it marks the heat as official without showing it.
func (s *TimingService) updateStatus(frame omega.Frame, status domain.HeatStatus) {
	event, heat, lapCount, ok := s.parseHeat(frame)
	if !ok {
		return
	}
	key := heatKey{
		event: event,
		heat:  heat,
	}
	if status == domain.HeatStatusOfficial {
		if s.heldStart != nil && s.heldStart.key == key {
			s.heldStart = nil
		}
		if !s.onDisplay(event, heat) {
			s.recordLateOfficial(frame, key, lapCount)
			return
		}
	}

	state := s.show(event, heat)
	state.applyStatus(frame, lapCount, status)

	s.publish(state.liveHeat())
}

// recordLateOfficial marks a heat kept in recent as official without showing it,
// so recalling it later shows the official result.
func (s *TimingService) recordLateOfficial(frame omega.Frame, key heatKey, lapCount int) {
	s.logger.Debug().Int("event", key.event).Int("heat", key.heat).Msg("recording official frame of a heat not on display")
	for _, state := range s.recent {
		if state.key == key {
			state.applyStatus(frame, lapCount, domain.HeatStatusOfficial)
			return
		}
	}
}

// startHeat applies a start frame to its heat and shows that heat. A start for an official heat
// and a repetition of the last start of the heat are ignored. After loading the next heat, the device sometimes
// repeats the previous start under the new heat number, so such a start is held back until a time confirms it.
func (s *TimingService) startHeat(frame omega.Frame, start startCorrection) {
	event, heat, lapCount, ok := s.parseHeat(frame)
	if !ok {
		return
	}
	key := heatKey{
		event: event,
		heat:  heat,
	}
	if s.isOfficial(key) {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("ignoring start frame for an official heat")
		return
	}
	if s.repeatsOwnStart(key, start) {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("ignoring start that repeats the last start of the heat")
		return
	}
	if s.repeatsLastStart(key, start) {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("holding back start that repeats the previous start")
		s.heldStart = &heldStart{
			key:      key,
			lapCount: lapCount,
			start:    start,
		}
		return
	}

	s.heldStart = nil
	state := s.show(event, heat)
	state.applyUsedLanes(frame)
	s.applyStart(state, lapCount, start)

	s.publish(state.liveHeat())
}

// isOfficial reports whether the heat is kept in recent and official there.
func (s *TimingService) isOfficial(key heatKey) bool {
	for _, state := range s.recent {
		if state.key == key {
			return state.heat.Status == domain.HeatStatusOfficial
		}
	}
	return false
}

// repeatsOwnStart reports whether a start carries the same time of day as the last start applied to its heat,
// which would clear the times recorded since.
func (s *TimingService) repeatsOwnStart(key heatKey, start startCorrection) bool {
	return start.valid && s.lastStart != nil && s.lastStart.key == key && s.lastStart.omegaMS == start.omegaMS
}

// repeatsLastStart reports whether a start carries the same time of day as the last start applied to another heat.
func (s *TimingService) repeatsLastStart(key heatKey, start startCorrection) bool {
	return start.valid && s.lastStart != nil && s.lastStart.key != key && s.lastStart.omegaMS == start.omegaMS
}

// updateTime records the time of a lane at a lap and shows its heat, so a heat whose load and start frames
// were missed still appears. A time confirms a held back start of its heat and adds a lane without a start entry.
// A button time only fills a lap without a time, since it backs up a missed touch.
func (s *TimingService) updateTime(frame omega.Frame, kind timeKind) {
	event, heat, lapCount, ok := s.parseHeat(frame)
	if !ok {
		return
	}
	frameLane, laneOK := frame.Lane()
	lap, lapOK := frame.Lap()
	ms, timeOK := frame.TimeMS()
	if !laneOK || !lapOK || !timeOK || lap < 1 {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("ignoring time frame without a valid lane, lap, or time")
		return
	}

	state := s.show(event, heat)
	if s.heldStart != nil && s.heldStart.key == state.key {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("applying start confirmed by a time")
		s.applyStart(state, s.heldStart.lapCount, s.heldStart.start)
		s.heldStart = nil
	}
	if state.heat.Status == "" {
		// The heat appears through a time, so it is running
		state.heat.Status = domain.HeatStatusStart
	}
	state.heat.LapCount = lapCount
	state.applyUsedLanes(frame)
	lane := laneNumber(state.heat.FirstLane, frameLane)
	if kind == timeButton && state.hasTime(lane, lap) {
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("ignoring button time for a lap with a time")
	} else {
		state.record(lane, lap, lapResult{
			time:   ms,
			finish: kind == timeFinish,
		})
	}
	state.updateLanes(s.getPoolLength(event, lapCount))
	state.updateStopTime()

	s.publish(state.liveHeat())
}

// parseHeat returns the event, the heat, and the number of laps a frame belongs to,
// or false if one of them cannot be parsed or the heat is below 1.
func (s *TimingService) parseHeat(frame omega.Frame) (event, heat, lapCount int, ok bool) {
	lapCount, err := frame.LapCount()
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to parse lap count")
		return 0, 0, 0, false
	}
	event, err = frame.Event()
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to parse event")
		return 0, 0, 0, false
	}
	heat, err = frame.Heat()
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to parse heat")
		return 0, 0, 0, false
	}
	if heat < 1 {
		// Heats are counted from 1, and the meet data keeps withdrawn entries below that
		s.logger.Debug().Int("event", event).Int("heat", heat).Msg("ignoring frame for a heat below 1")
		return 0, 0, 0, false
	}
	return event, heat, lapCount, true
}

// show makes a heat the one on display and returns it, with its times if it is kept in recent.
// It takes the meet data on every frame, so a heat shows reloaded meet data right away.
func (s *TimingService) show(event, heat int) *heatState {
	key := heatKey{
		event: event,
		heat:  heat,
	}
	if !s.onDisplay(event, heat) {
		s.current = s.recall(key)
	}

	data, ok := s.store.GetHeat(event, heat-1)
	if !ok && !s.current.warned {
		s.current.warned = true
		s.logger.Warn().Int("event", event).Int("heat", heat).Msg("event not found in meet data")
	}
	s.current.applyMeetData(data, ok)
	return s.current
}

// onDisplay reports whether the heat is the one on display.
func (s *TimingService) onDisplay(event, heat int) bool {
	return s.current != nil && s.current.key == (heatKey{
		event: event,
		heat:  heat,
	})
}

// recall returns the heat from recent and moves it to the end, or adds a new heat,
// dropping the oldest one beyond recentHeatCount.
func (s *TimingService) recall(key heatKey) *heatState {
	for i, state := range s.recent {
		if state.key == key {
			s.recent = append(slices.Delete(s.recent, i, i+1), state)
			return state
		}
	}

	state := &heatState{
		key: key,
		heat: &domain.LiveHeat{
			Event: key.event,
			Heat:  key.heat,
			Lanes: make(map[int]*domain.Lane),
		},
		results: make(map[int]map[int]lapResult),
		listed:  make(map[int]bool),
	}
	s.recent = append(s.recent, state)
	if len(s.recent) > recentHeatCount {
		s.recent = slices.Delete(s.recent, 0, len(s.recent)-recentHeatCount)
	}
	return state
}

// applyStart starts the heat at the corrected start time, clears its times, and remembers the start,
// so a repetition of it under another heat number is recognized.
func (s *TimingService) applyStart(state *heatState, lapCount int, start startCorrection) {
	state.heat.LapCount = lapCount
	state.heat.Status = domain.HeatStatusStart
	state.heat.StartTime = new(start.at)
	state.clearResults()

	s.lastStart = nil
	if start.valid {
		s.lastStart = &appliedStart{
			key:     state.key,
			omegaMS: start.omegaMS,
		}
	}
}

// publish stores a new live heat and broadcasts it. The message is encoded under the lock, so the stored
// and the broadcast state match, but sent after releasing it, so a slow broadcast never blocks readers.
func (s *TimingService) publish(state *domain.LiveHeat) {
	s.mu.Lock()
	message, err := json.Marshal(domain.NewMessage(domain.MessageUpdate, state))
	if err != nil {
		s.mu.Unlock()
		s.logger.Error().Err(err).Msg("failed to marshal live heat")
		return
	}
	s.heat = state
	s.mu.Unlock()

	s.broadcaster.Broadcast(message)
}

// getPoolLength returns the pool length of an event in meters: the configured one, or else the distance
// of the event divided by the lap count the device transmits, with 50 m if either is unknown.
func (s *TimingService) getPoolLength(event, lapCount int) int {
	if s.options.PoolLength > 0 {
		return s.options.PoolLength
	}
	meetEvent, ok := s.store.GetEvent(event)
	if !ok || meetEvent.Distance == nil || meetEvent.Distance.Length <= 0 || lapCount <= 0 {
		return 50
	}
	return meetEvent.Distance.Length / lapCount
}

// fixedZone returns a zone with the name and offset in effect at t, which a daylight saving change does not move.
func fixedZone(t time.Time) *time.Location {
	name, offset := t.Zone()
	return time.FixedZone(name, offset)
}

// timeKind is the kind of time a time frame carries.
type timeKind int

const (
	// timeIntermediate is a split, or at the last lap a withdrawn finish.
	timeIntermediate timeKind = iota
	// timeFinish is the finish of a lane.
	timeFinish
	// timeButton is the time of a button pressed at the finish, a backup for a missed touch.
	timeButton
)

// heatKey identifies a heat by its event and heat number.
type heatKey struct {
	event int
	heat  int
}

// lapResult is the time of a lane at a lap.
type lapResult struct {
	time int
	// finish reports whether the device sent the time as the finish of the lane.
	// An intermediate time at the last lap replaces a finish, since it withdraws a touch taken as finish.
	finish bool
}

// heatState is a heat together with the time of every lap it is built from,
// so a correction replaces only its lap and the ranks of every lap can be computed.
type heatState struct {
	key  heatKey
	heat *domain.LiveHeat
	// results holds the times by lane and lap.
	results map[int]map[int]lapResult
	// listed holds the lanes with a start entry in the meet data.
	listed map[int]bool
	// usedLanes holds the lanes the device last reported in use for the heat, as single-digit lane numbers.
	usedLanes []int
	// warned reports whether the missing meet data of the heat was logged.
	warned bool
}

// applyMeetData takes the details of the event and the competitors of the heat from the meet data.
// Lanes keep their times, and without meet data the heat keeps what it shows.
func (h *heatState) applyMeetData(data store.HeatData, found bool) {
	h.heat.FirstLane = data.FirstLane
	if !found {
		return
	}

	h.heat.HeatCount = data.Event.HeatCount
	h.heat.Name = data.Event.DisplayName()
	h.heat.Phase = data.Event.PhaseID
	h.heat.PhaseName = data.Event.PhaseName()

	// Entries sharing a lane show the one listed last
	entries := make(map[int]*domain.StartEntry, len(data.StartEntries))
	for _, entry := range data.StartEntries {
		entries[entry.Lane] = entry
	}

	clear(h.listed)
	for number, entry := range entries {
		h.listed[number] = true
		lane := h.lane(number)
		lane.FirstName, lane.LastName, lane.BirthYear, lane.Club = "", "", "", ""
		if entry.Competitor != nil {
			lane.FirstName = entry.Competitor.FirstName
			lane.LastName = entry.Competitor.LastName
			lane.BirthYear = entry.Competitor.BirthYear
			lane.Club = entry.Competitor.Club()
		}
	}
	for number, lane := range h.heat.Lanes {
		if !h.listed[number] {
			lane.FirstName, lane.LastName, lane.BirthYear, lane.Club = "", "", "", ""
		}
	}
}

// applyStatus applies a load or official frame to the heat.
func (h *heatState) applyStatus(frame omega.Frame, lapCount int, status domain.HeatStatus) {
	h.heat.LapCount = lapCount
	h.heat.Status = status
	h.applyUsedLanes(frame)
	h.updateStopTime()
}

// applyUsedLanes takes the lanes in use from a frame of the heat. A frame without a valid mask keeps the earlier one.
func (h *heatState) applyUsedLanes(frame omega.Frame) {
	lanes, ok := frame.UsedLanes()
	if !ok {
		return
	}
	h.usedLanes = lanes
}

// clearResults removes every time of the heat and the lanes without a start entry,
// so a restart after a false start or test touches does not show the times of the previous attempt.
func (h *heatState) clearResults() {
	clear(h.results)
	h.heat.StopTime = 0
	for number, lane := range h.heat.Lanes {
		if !h.listed[number] {
			delete(h.heat.Lanes, number)
			continue
		}
		lane.Lap = 0
		lane.Meter = 0
		lane.Time = 0
		lane.Rank = 0
		lane.Finished = false
	}
}

// hasTime reports whether the lane has a time at the lap.
func (h *heatState) hasTime(lane, lap int) bool {
	_, ok := h.results[lane][lap]
	return ok
}

// record stores the time of a lane at a lap, replacing an earlier time of the same lane and lap.
func (h *heatState) record(lane, lap int, result lapResult) {
	laps, ok := h.results[lane]
	if !ok {
		laps = make(map[int]lapResult)
		h.results[lane] = laps
	}
	laps[lap] = result
}

// updateLanes shows for every lane the time of its highest lap with its rank at that lap. The ranks are computed
// from the times instead of taken from the device, which only sends the rank of the lane of the frame.
func (h *heatState) updateLanes(poolLength int) {
	ranks := lapRanks(h.results, h.heat.LapCount)

	for number, laps := range h.results {
		highest := 0
		for lap := range laps {
			highest = max(highest, lap)
		}
		result := laps[highest]

		lane := h.lane(number)
		lane.Lap = highest
		lane.Meter = highest * poolLength
		lane.Time = result.time
		lane.Rank = ranks[highest][number]
		lane.Finished = result.finish
	}
}

// updateStopTime sets the stop time of the heat to the highest finish time once every lane the device reports
// in use has finished, and clears it otherwise.
func (h *heatState) updateStopTime() {
	h.heat.StopTime = 0
	if len(h.usedLanes) == 0 {
		return
	}

	stopTime := 0
	for _, frameLane := range h.usedLanes {
		lane, ok := h.heat.Lanes[laneNumber(h.heat.FirstLane, frameLane)]
		if !ok || !lane.Finished {
			return
		}
		stopTime = max(stopTime, lane.Time)
	}
	h.heat.StopTime = stopTime
}

// liveHeat returns a copy of the heat as the overlays receive it, without the lanes that have no time
// and that the device does not report in use. A lane with a time stays, since the mask also drops lanes in use.
func (h *heatState) liveHeat() *domain.LiveHeat {
	heat := h.heat.Copy()
	if len(h.usedLanes) == 0 {
		return heat
	}

	inUse := make(map[int]bool, len(h.usedLanes))
	for _, frameLane := range h.usedLanes {
		inUse[laneNumber(heat.FirstLane, frameLane)] = true
	}
	for number := range heat.Lanes {
		if !inUse[number] && len(h.results[number]) == 0 {
			delete(heat.Lanes, number)
		}
	}
	return heat
}

// laneNumber maps the single-digit lane of a frame to a lane of the pool,
// where 0 is lane 10 unless the pool is numbered from 0.
func laneNumber(firstLane, frameLane int) int {
	if frameLane != 0 || firstLane == 0 {
		return frameLane
	}
	return 10
}

// lane returns the lane with the given number, adding it if the heat has none yet.
func (h *heatState) lane(number int) *domain.Lane {
	lane, ok := h.heat.Lanes[number]
	if !ok {
		lane = &domain.Lane{
			Number: number,
		}
		h.heat.Lanes[number] = lane
	}
	return lane
}

// lapRanks returns the rank of every lane at every lap, by lap and lane, with a shared rank for equal times.
// A withdrawn finish takes no part in the ranks of the last lap.
func lapRanks(results map[int]map[int]lapResult, lapCount int) map[int]map[int]int {
	type laneTime struct {
		lane int
		time int
	}
	byLap := make(map[int][]laneTime)
	for lane, laps := range results {
		for lap, result := range laps {
			if lapCount > 0 && lap >= lapCount && !result.finish {
				continue
			}
			byLap[lap] = append(byLap[lap], laneTime{
				lane: lane,
				time: result.time,
			})
		}
	}

	ranks := make(map[int]map[int]int, len(byLap))
	for lap, times := range byLap {
		slices.SortFunc(times, func(a, b laneTime) int { return cmp.Or(cmp.Compare(a.time, b.time), cmp.Compare(a.lane, b.lane)) })
		ranks[lap] = make(map[int]int, len(times))
		for i, t := range times {
			rank := i + 1
			if i > 0 && t.time == times[i-1].time {
				rank = ranks[lap][times[i-1].lane]
			}
			ranks[lap][t.lane] = rank
		}
	}
	return ranks
}

// appliedStart is a start applied to a heat with the device's time of day it carried.
type appliedStart struct {
	key     heatKey
	omegaMS int
}

// heldStart is a start held back until a time of its heat confirms it.
type heldStart struct {
	key      heatKey
	lapCount int
	start    startCorrection
}

// startClock corrects the start time of a heat by the delay of its start signal. A start signal is only ever delayed,
// so the smallest offset between the local time of day and the device's time of day of any start is the baseline
// of an on-time signal, and a late start is shifted back by its delay. It is not safe for concurrent use.
type startClock struct {
	logger zerolog.Logger
	// zone is the zone offset of the host when the clock started, see localTimeOfDayMS.
	zone *time.Location
	// baseline is the smallest clock offset of any start since the clock started or was reset,
	// nil before the first start with a time of day.
	baseline *time.Duration
	// confirmations counts the starts that arrived without delay against the current baseline.
	confirmations int
}

// newStartClock creates a startClock that reads the local time of day in zone.
func newStartClock(logger zerolog.Logger, zone *time.Location) *startClock {
	return &startClock{
		logger: logger,
		zone:   zone,
	}
}

// correct returns the start time of a start frame that arrived at now, with the clock values it is based on.
// Without a valid time of day, the heat starts at arrival.
func (c *startClock) correct(frame omega.Frame, now time.Time) startCorrection {
	omegaMS, ok := frame.TimeMS()
	if !ok {
		return startCorrection{
			at: now,
		}
	}

	localMS := localTimeOfDayMS(now.In(c.zone))
	offset := clockOffset(localMS, omegaMS)

	baseline, plausible := c.updateBaseline(offset)
	delay := offset - baseline
	if !plausible {
		c.logger.Warn().
			Int("omega_time_ms", omegaMS).
			Int("local_day_ms", localMS).
			Dur("clock_offset", offset).
			Dur("baseline", baseline).
			Msg("start signal lies before the baseline, a clock was probably set, starting at its arrival")
		delay = 0
	}

	c.logger.Info().
		Int("omega_time_ms", omegaMS).
		Int("local_day_ms", localMS).
		Dur("clock_offset", offset).
		Dur("baseline", baseline).
		Dur("delay", delay).
		Msg("start signal received")

	return startCorrection{
		at:       now.Add(-delay),
		valid:    true,
		omegaMS:  omegaMS,
		localMS:  localMS,
		offset:   offset,
		baseline: baseline,
		delay:    delay,
	}
}

// localTimeOfDayMS returns the milliseconds since midnight of t in its zone. Callers pass the fixed zone
// the host had at startup, since the device clock does not follow a daylight saving change.
func localTimeOfDayMS(t time.Time) int {
	h, m, sec := t.Clock()
	ms := t.Nanosecond() / 1_000_000
	return (h*3600+m*60+sec)*1000 + ms
}

// clockOffset returns the difference between the local and the device's time of day, wrapped into twelve hours
// in both directions, so it stays continuous when only one of the clocks has passed midnight.
func clockOffset(localDayMS, omegaTimeMS int) time.Duration {
	const (
		dayMS     = 24 * 60 * 60 * 1000
		halfDayMS = dayMS / 2
	)

	offset := localDayMS - omegaTimeMS
	if offset > halfDayMS {
		offset -= dayMS
	}
	if offset <= -halfDayMS {
		offset += dayMS
	}
	return time.Duration(offset) * time.Millisecond
}

// updateBaseline adjusts the baseline to the offset of a start and returns the baseline the start is measured
// against. It reports false for an offset further below a confirmed baseline than the clocks can drift.
func (c *startClock) updateBaseline(offset time.Duration) (time.Duration, bool) {
	if c.baseline == nil {
		c.baseline = &offset
		c.confirmations = 0
		return offset, true
	}

	baseline := *c.baseline
	if offset >= baseline {
		if offset-baseline <= baselineMatch {
			c.confirmations++
		}
		return baseline, true
	}

	drop := baseline - offset
	if c.confirmations >= baselineConfirmations && drop > baselineDrift {
		return baseline, false
	}

	// A drop beyond the drift corrects a baseline set by a late start, which has to be confirmed anew
	if drop > baselineMatch {
		c.confirmations = 0
	}
	c.baseline = &offset
	return offset, true
}

// reset forgets the baseline, so the next start measures the clock offset anew.
func (c *startClock) reset() {
	c.baseline = nil
	c.confirmations = 0
}

// startCorrection is the corrected start time of a start frame with the clock values it is based on.
type startCorrection struct {
	// at is the start time of the heat.
	at time.Time
	// valid reports whether the frame carried a time of day, without one the other fields are empty.
	valid    bool
	omegaMS  int
	localMS  int
	offset   time.Duration
	baseline time.Duration
	delay    time.Duration
}
