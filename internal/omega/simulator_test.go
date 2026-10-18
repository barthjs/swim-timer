package omega_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/omega"
)

// fakeClock advances instantly: every wait moves the time forward by the waited duration,
// so a heat of several minutes runs in microseconds and the frames still carry realistic timing.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// After advances the fake time by d and returns a channel that already holds the new time.
func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	c.mu.Unlock()

	ch := make(chan time.Time, 1)
	ch <- now
	return ch
}

// heatRecorder collects the frames the simulator writes
// and cancels the run once the given number of official frames arrived,
// so a test sees exactly the heats it asked for.
type heatRecorder struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	officials int
	wantHeats int
	cancel    context.CancelFunc
	clock     *fakeClock
	writtenAt []time.Time
}

// Write records the frame and cancels the run once the wanted number of heats is official.
func (r *heatRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.officials >= r.wantHeats {
		return len(p), nil
	}
	_, _ = r.buf.Write(p)
	r.writtenAt = append(r.writtenAt, r.clock.Now())

	frame, err := omega.NewDecoder(bytes.NewReader(p)).Next()
	if err == nil && frame.Kind() == omega.KindOfficial {
		r.officials++
		if r.officials == r.wantHeats {
			r.cancel()
		}
	}
	return len(p), nil
}

func TestSystemClock(t *testing.T) {
	t.Parallel()

	t.Run("returns the time of the system", func(t *testing.T) {
		t.Parallel()

		before := time.Now()

		got := omega.SystemClock().Now()

		assert.False(t, got.Before(before))
		assert.WithinDuration(t, time.Now(), got, time.Second)
	})
}

func TestWithClock(t *testing.T) {
	t.Parallel()

	t.Run("ignores a nil clock", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		assert.NotPanics(t, func() {
			omega.NewSimulator(zerolog.Nop(), io.Discard, omega.WithClock(nil)).
				Run(ctx, 1, 1, []omega.SimulatedHeat{
					{
						Number: 1,
						Lanes:  []int{1},
					},
				})
		})
	})
}

func TestWithRandSeed(t *testing.T) {
	t.Parallel()

	t.Run("reproduces the heats", func(t *testing.T) {
		t.Parallel()

		heats := []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{1, 2, 3, 4, 5, 6, 7, 8},
			},
		}
		first, _ := simulate(t, heats, 2, omega.WithRandSeed(42))
		second, _ := simulate(t, heats, 2, omega.WithRandSeed(42))

		require.Len(t, second, len(first))
		for i := range first {
			assert.Equal(t, first[i].Hex(), second[i].Hex())
		}
	})
}

func TestSimulator_Run(t *testing.T) {
	t.Parallel()

	t.Run("replays a heat", func(t *testing.T) {
		t.Parallel()

		frames, writtenAt := simulate(t, []omega.SimulatedHeat{
			{
				Number: 2,
				Lanes:  []int{3, 4, 5, 4},
			},
		}, 2, omega.WithRandSeed(1))

		// load, start, three swimmers on two laps, official; the duplicate lane 4 counts once
		require.Len(t, frames, 9)
		assert.Equal(t, omega.KindLoad, frames[0].Kind())
		assert.Equal(t, omega.KindStart, frames[1].Kind())
		assert.Equal(t, omega.KindOfficial, frames[8].Kind())

		for _, frame := range frames {
			event, err := frame.Event()
			require.NoError(t, err)
			assert.Equal(t, 12, event)
			heat, err := frame.Heat()
			require.NoError(t, err)
			assert.Equal(t, 2, heat)
			lapCount, err := frame.LapCount()
			require.NoError(t, err)
			assert.Equal(t, 2, lapCount)
		}

		for i, lap := range []int{1, 2} {
			laneFrames := frames[2+i*3 : 5+i*3]
			wantKind := omega.KindIntermediate
			if lap == 2 {
				wantKind = omega.KindFinish
			}

			lanes := map[int]bool{}
			for rank, frame := range laneFrames {
				assert.Equal(t, wantKind, frame.Kind())
				assert.Equal(t, lap, mustLap(t, frame))
				assert.Equal(t, rank+1, mustRank(t, frame), "times arrive in the order of their ranks")
				lanes[mustLane(t, frame)] = true
				if rank > 0 {
					assert.GreaterOrEqual(t, mustTimeMS(t, frame), mustTimeMS(t, laneFrames[rank-1]))
				}
			}
			assert.Equal(t, map[int]bool{3: true, 4: true, 5: true}, lanes)
		}

		// Every time is written at the moment it is reached, counted from the start frame
		start := writtenAt[1]
		for i := 2; i < 8; i++ {
			elapsed := writtenAt[i].Sub(start)
			assert.Equal(t, time.Duration(mustTimeMS(t, frames[i]))*time.Millisecond, elapsed.Truncate(10*time.Millisecond))
		}

		// Swimmers need 20 to 30 seconds per lap plus up to a second for an outer lane
		for _, frame := range frames[2:5] {
			assert.GreaterOrEqual(t, mustTimeMS(t, frame), 20_000)
			assert.Less(t, mustTimeMS(t, frame), 31_000)
		}
	})

	t.Run("carries the time of day in the start frame", func(t *testing.T) {
		t.Parallel()

		frames, writtenAt := simulate(t, []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{4},
			},
		}, 1, omega.WithRandSeed(6))

		require.Equal(t, omega.KindStart, frames[1].Kind())
		local := writtenAt[1].Local()
		hours, minutes, seconds := local.Clock()
		want := (hours*3600+minutes*60+seconds)*1000 + local.Nanosecond()/int(time.Millisecond)/10*10
		assert.Equal(t, want, mustTimeMS(t, frames[1]), "the local time of day at the start")
		assert.Equal(t, 1, mustLane(t, frames[1]))
		assert.Equal(t, 0, mustLap(t, frames[1]))
		assert.Equal(t, 0, mustRank(t, frames[1]))
	})

	t.Run("sends the used lanes", func(t *testing.T) {
		t.Parallel()

		frames, _ := simulate(t, []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{3, 4, 5},
			},
		}, 1, omega.WithRandSeed(7))

		for _, frame := range frames {
			assert.Equal(t, "2720", frame.Fields().UsedLanes)
		}
	})

	t.Run("counts the penalty from the middle of the used lanes", func(t *testing.T) {
		t.Parallel()

		// The same seed and number of lanes give the same base speeds, so only the penalty could differ
		low := lapTimesByPosition(t, []int{1, 2, 3})
		high := lapTimesByPosition(t, []int{6, 7, 8})

		assert.Equal(t, low, high)
	})

	t.Run("pauses before the start", func(t *testing.T) {
		t.Parallel()

		_, writtenAt := simulate(t, []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{4},
			},
		}, 1, omega.WithRandSeed(2))

		assert.Equal(t, 10*time.Second, writtenAt[1].Sub(writtenAt[0]), "between load and start")
	})

	t.Run("replays the heats in order", func(t *testing.T) {
		t.Parallel()

		frames, _ := simulate(t, []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{1, 2},
			},
			{
				Number: 2,
				Lanes:  []int{3},
			},
		}, 1, omega.WithRandSeed(3))

		var loaded []int
		for _, frame := range frames {
			if frame.Kind() == omega.KindLoad {
				heat, err := frame.Heat()
				require.NoError(t, err)
				loaded = append(loaded, heat)
			}
		}
		assert.Equal(t, []int{1, 2}, loaded)
	})

	t.Run("returns without heats", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})
		go func() {
			omega.NewSimulator(zerolog.Nop(), io.Discard).Run(t.Context(), 1, 1, nil)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Run without heats did not return")
		}
	})

	t.Run("stops when canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		var out bytes.Buffer
		omega.NewSimulator(zerolog.Nop(), &out).Run(ctx, 1, 1, []omega.SimulatedHeat{
			{
				Number: 1,
				Lanes:  []int{1},
			},
		})

		assert.Zero(t, out.Len())
	})

	t.Run("stops during a heat", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		clock := newFakeClock()
		writer := &cancelAfter{
			frames: 2,
			cancel: cancel,
		}

		done := make(chan struct{})
		go func() {
			omega.NewSimulator(zerolog.Nop(), writer, omega.WithClock(clock), omega.WithRandSeed(4)).
				Run(ctx, 1, 4, []omega.SimulatedHeat{
					{
						Number: 1,
						Lanes:  []int{1, 2, 3},
					},
				})
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the simulator did not stop")
		}
		assert.Equal(t, 2, writer.count(), "the heat ends at its first wait after the cancel, before any lap time")
	})

	t.Run("logs write errors", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		var logs bytes.Buffer
		writer := &failingWriter{
			cancel: cancel,
		}

		omega.NewSimulator(zerolog.New(&logs), writer, omega.WithClock(newFakeClock()), omega.WithRandSeed(5)).
			Run(ctx, 1, 1, []omega.SimulatedHeat{
				{
					Number: 1,
					Lanes:  []int{1},
				},
			})

		assert.Contains(t, logs.String(), "failed to write simulated frame")
	})
}

// lapTimesByPosition simulates a heat of one lap and returns each swimmer's time
// by the position of the lane among the lanes in use.
func lapTimesByPosition(t *testing.T, lanes []int) map[int]int {
	t.Helper()

	frames, _ := simulate(t, []omega.SimulatedHeat{
		{
			Number: 1,
			Lanes:  lanes,
		},
	}, 1, omega.WithRandSeed(8))

	times := map[int]int{}
	for _, frame := range frames {
		if frame.Kind() != omega.KindFinish {
			continue
		}
		times[mustLane(t, frame)-lanes[0]] = mustTimeMS(t, frame)
	}
	require.Len(t, times, len(lanes))
	return times
}

// simulate runs the simulator until heats official frames were written and returns the decoded frames
// with the fake time each one was written at.
func simulate(t *testing.T, heats []omega.SimulatedHeat, lapCount int, opts ...omega.Option) ([]omega.Frame, []time.Time) {
	t.Helper()

	clock := newFakeClock()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	recorder := &heatRecorder{
		wantHeats: len(heats),
		cancel:    cancel,
		clock:     clock,
	}

	done := make(chan struct{})
	go func() {
		options := append([]omega.Option{omega.WithClock(clock)}, opts...)
		simulator := omega.NewSimulator(zerolog.Nop(), recorder, options...)
		simulator.Run(ctx, 12, lapCount, heats)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the simulator did not finish the heats")
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	decoder := omega.NewDecoder(bytes.NewReader(recorder.buf.Bytes()))
	var frames []omega.Frame
	for {
		frame, err := decoder.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		frames = append(frames, frame)
	}
	return frames, recorder.writtenAt
}

// mustLap returns the lap of a frame and fails the test if the field is empty or invalid.
func mustLap(t *testing.T, frame omega.Frame) int {
	t.Helper()
	lap, ok := frame.Lap()
	require.True(t, ok, "frame carries a lap")
	return lap
}

// mustRank returns the rank of a frame and fails the test if the field is empty or invalid.
func mustRank(t *testing.T, frame omega.Frame) int {
	t.Helper()
	rank, ok := frame.Rank()
	require.True(t, ok, "frame carries a rank")
	return rank
}

// mustLane returns the lane of a frame and fails the test if the field is empty or invalid.
func mustLane(t *testing.T, frame omega.Frame) int {
	t.Helper()
	lane, ok := frame.Lane()
	require.True(t, ok, "frame carries a lane")
	return lane
}

// mustTimeMS returns the time of a frame and fails the test if the field holds no valid time.
func mustTimeMS(t *testing.T, frame omega.Frame) int {
	t.Helper()
	ms, ok := frame.TimeMS()
	require.True(t, ok, "frame carries a valid time")
	return ms
}

// newFakeClock returns a fake clock at a fixed time.
func newFakeClock() *fakeClock {
	return &fakeClock{
		now: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
	}
}

// cancelAfter cancels the run after the given number of written frames.
type cancelAfter struct {
	mu      sync.Mutex
	written int
	frames  int
	cancel  context.CancelFunc
}

// Write counts the frame and cancels the run once the given number of frames is written.
func (w *cancelAfter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.written++
	if w.written == w.frames {
		w.cancel()
	}
	return len(p), nil
}

// count returns the number of written frames.
func (w *cancelAfter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.written
}

// failingWriter fails every write and cancels the run once the official frame was attempted.
type failingWriter struct {
	mu     sync.Mutex
	writes int
	cancel context.CancelFunc
}

// Write always fails and cancels the run after the fourth attempt.
func (w *failingWriter) Write([]byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	if w.writes == 4 {
		w.cancel()
	}
	return 0, errors.New("broken pipe")
}
