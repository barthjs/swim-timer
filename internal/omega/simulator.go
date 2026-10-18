package omega

import (
	"cmp"
	"context"
	"io"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/rs/zerolog"
)

const (
	// simulatorHeatPause is the pause before each heat and between its load and start frames.
	simulatorHeatPause = 10 * time.Second
)

// SimulatedHeat is one heat the simulator replays.
type SimulatedHeat struct {
	// Number is the heat number as the device transmits it, counted from 1.
	Number int
	// Lanes are the occupied lanes. Duplicates are ignored.
	Lanes []int
}

// Clock is the time source of the simulator. Tests replace it, so a whole heat runs without waiting.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// SystemClock returns the clock of the system.
func SystemClock() Clock {
	return realClock{}
}

// realClock is the system time.
type realClock struct{}

// Now returns the current time.
func (realClock) Now() time.Time { return time.Now() }

// After waits for the duration to elapse and then sends the current time on the returned channel.
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Option configures a Simulator.
type Option func(*Simulator)

// WithClock replaces the system time, for example with a clock that advances instantly in tests.
// A nil clock is ignored.
func WithClock(clock Clock) Option {
	return func(s *Simulator) {
		if clock != nil {
			s.clock = clock
		}
	}
}

// WithRandSeed makes the swimmers' speeds reproducible.
func WithRandSeed(seed int64) Option {
	return func(s *Simulator) {
		s.rng = rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
	}
}

// Simulator writes realistic timing frames for an event to a writer,
// so the application can be demonstrated and tested without a timing device.
type Simulator struct {
	logger zerolog.Logger
	writer io.Writer
	clock  Clock
	rng    *rand.Rand
}

// NewSimulator creates a Simulator that writes to w.
func NewSimulator(logger zerolog.Logger, w io.Writer, opts ...Option) *Simulator {
	s := &Simulator{
		logger: logger.With().Str("component", "simulator").Logger(),
		writer: w,
		clock:  realClock{},
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.rng == nil {
		s.rng = rand.New(rand.NewPCG(uint64(s.clock.Now().UnixNano()), 0))
	}
	return s
}

// Run replays the heats of an event in a loop until ctx is canceled.
// Without heats there is nothing to replay, and it returns at once.
func (s *Simulator) Run(ctx context.Context, event, lapCount int, heats []SimulatedHeat) {
	if len(heats) == 0 {
		return
	}

	for {
		for _, heat := range heats {
			if !s.wait(ctx, simulatorHeatPause) {
				return
			}
			s.simulateHeat(ctx, event, lapCount, heat)
		}
	}
}

// simulateHeat runs a single heat: a load frame, a start frame with the time of day, one time per swimmer and lap
// at the moment it is reached, and an official frame. Swimmers get a random speed of 20 to 30 seconds per 50 meters
// plus a penalty for their distance to the middle lane.
func (s *Simulator) simulateHeat(ctx context.Context, event, lapCount int, heat SimulatedHeat) {
	s.logger.Info().Int("event", event).Int("heat", heat.Number).Msg("simulating heat")

	laneSet := make(map[int]bool)
	for _, lane := range heat.Lanes {
		laneSet[lane] = true
	}
	lanes := slices.Collect(maps.Keys(laneSet))
	// Sorting keeps the order of the random speeds independent of the map iteration order
	slices.Sort(lanes)

	middleLane := 0.0
	if len(lanes) > 0 {
		middleLane = float64(lanes[0]+lanes[len(lanes)-1]) / 2.0
	}

	s.write(FrameData{
		Kind:      KindLoad,
		UsedLanes: lanes,
		LapCount:  lapCount,
		Event:     event,
		Heat:      heat.Number,
	})

	if !s.wait(ctx, simulatorHeatPause) {
		return
	}

	startTime := s.clock.Now()
	s.write(FrameData{
		Kind:      KindStart,
		UsedLanes: lanes,
		LapCount:  lapCount,
		Event:     event,
		Heat:      heat.Number,
		Time:      timeOfDay(startTime),
	})

	type participant struct {
		lane  int
		speed time.Duration // time per 50m
		time  time.Duration
		rank  int
	}

	parts := make([]*participant, len(lanes))
	for i, lane := range lanes {
		baseSpeed := 20*time.Second + s.rng.N(10*time.Second)
		distFromMiddle := math.Abs(float64(lane) - middleLane)
		penalty := time.Duration(distFromMiddle * float64(500*time.Millisecond))

		parts[i] = &participant{
			lane:  lane,
			speed: baseSpeed + penalty,
		}
	}

	for lap := 1; lap <= lapCount; lap++ {
		for _, p := range parts {
			p.time = p.speed * time.Duration(lap)
		}

		sortedParts := slices.Clone(parts)
		slices.SortFunc(sortedParts, func(a, b *participant) int { return cmp.Compare(a.time, b.time) })

		for i, p := range sortedParts {
			p.rank = i + 1
		}

		// Send the times in order of arrival, each at the moment it is reached
		for _, p := range sortedParts {
			remaining := p.time - s.clock.Now().Sub(startTime)
			if remaining > 0 && !s.wait(ctx, remaining) {
				return
			}

			kind := KindIntermediate
			if lap == lapCount {
				kind = KindFinish
			}
			s.write(FrameData{
				Kind:      kind,
				UsedLanes: lanes,
				LapCount:  lapCount,
				Event:     event,
				Heat:      heat.Number,
				Rank:      p.rank,
				Lane:      p.lane,
				Lap:       lap,
				Time:      p.time,
			})
		}
	}

	s.write(FrameData{
		Kind:      KindOfficial,
		UsedLanes: lanes,
		LapCount:  lapCount,
		Event:     event,
		Heat:      heat.Number,
	})
}

// wait waits for d on the clock of the simulator and reports whether the run goes on.
// A canceled ctx ends the run even if d has elapsed as well, which select alone would pick at random.
func (s *Simulator) wait(ctx context.Context, d time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-s.clock.After(d):
		return true
	}
}

// timeOfDay returns the time elapsed since local midnight, as the simulated device transmits it in a start frame.
// The device clock runs in local time, like the clock the start time correction compares it with.
func timeOfDay(t time.Time) time.Duration {
	local := t.Local()
	hours, minutes, seconds := local.Clock()
	clock := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	return clock + time.Duration(local.Nanosecond())
}

// write encodes a frame and writes it. A failed write is logged, the simulation continues.
func (s *Simulator) write(d FrameData) {
	if _, err := s.writer.Write(Encode(d)); err != nil {
		s.logger.Warn().Err(err).Msg("failed to write simulated frame")
	}
}
