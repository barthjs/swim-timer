package service

import (
	"bytes"
	"context"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/store"
)

func TestWithSimulatorClock(t *testing.T) {
	t.Parallel()

	t.Run("ignores a nil clock", func(t *testing.T) {
		t.Parallel()

		svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{}, nil, nil, nil, WithSimulatorClock(nil))

		assert.NotNil(t, svc.clock)
		assert.Empty(t, svc.generatorOptions)
	})
}

func TestWithSimulatorRandSeed(t *testing.T) {
	t.Parallel()

	t.Run("generates the same times for the same seed", func(t *testing.T) {
		t.Parallel()

		first := runFirstHeat(t, newEventSimulator(t, tenLaneFiles))
		second := runFirstHeat(t, newEventSimulator(t, tenLaneFiles))

		require.Len(t, second, len(first))
		for i := range first {
			firstTime, _ := first[i].TimeMS()
			secondTime, _ := second[i].TimeMS()
			assert.Equal(t, firstTime, secondTime)
		}
	})
}

func TestSimulatorService_Run(t *testing.T) {
	t.Parallel()

	t.Run("replays the event", func(t *testing.T) {
		t.Parallel()

		svc := newEventSimulator(t, tenLaneFiles)

		frames := runFirstHeat(t, svc)

		// load, start, two swimmers of the event's phase on two laps of the 50 m pool, official
		require.Len(t, frames, 7)
		heat, err := frames[0].Heat()
		require.NoError(t, err)
		assert.Equal(t, 1, heat, "the start list counts heats from 0, the device from 1")
		lapCount, err := frames[0].LapCount()
		require.NoError(t, err)
		assert.Equal(t, 2, lapCount, "100 m in a 50 m pool")
		lanes := map[int]bool{}
		for _, frame := range frames[2:6] {
			lane, ok := frame.Lane()
			require.True(t, ok)
			lanes[lane] = true
		}
		// Lane 10 is transmitted as the single digit 0
		assert.Equal(t, map[int]bool{1: true, 0: true}, lanes, "lanes 1 and 10 of phase 2 only")
	})

	t.Run("skips withdrawn entries", func(t *testing.T) {
		t.Parallel()

		files := maps.Clone(tenLaneFiles)
		files["LSTSTART.TXT"] = "event;round;heat;lane;relais;idBib\n7;2;-1;3;0;10;\n7;2;0;1;0;1;\n"
		svc := newEventSimulator(t, files)

		frames := runFirstHeat(t, svc)

		// load, start, one swimmer on two laps of the 50 m pool, official
		require.Len(t, frames, 5)
		heat, err := frames[0].Heat()
		require.NoError(t, err)
		assert.Equal(t, 1, heat, "the withdrawn entry does not form a heat of its own")
		lane, ok := frames[2].Lane()
		require.True(t, ok)
		assert.Equal(t, 1, lane, "the withdrawn lane is not swum")
	})

	t.Run("replays the recording instead of generating frames", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Second), 0x01),
		)
		svc, clock := newRecordingSimulator(t, recordings, 0, nil)
		writer := newReplayWriter(clock, 1)

		runReplay(t, writer, func(ctx context.Context) { svc.Run(ctx, writer) })

		assert.Equal(t, [][]byte{
			{0x01},
		}, writer.data)
	})

	tests := []struct {
		name  string
		files map[string]string
		event int
	}{
		{
			name:  "returns without data for an empty start list",
			files: map[string]string{},
			event: 7,
		},
		{
			name:  "returns without data for an event not in the start list",
			files: tenLaneFiles,
			event: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
				Event:      tt.event,
				PoolLength: 50,
			}, newTestMeetStore(t, tt.files), nil, nil)

			out := runSimulator(t, svc)

			assert.Zero(t, out.Len())
		})
	}

	t.Run("stops when canceled", func(t *testing.T) {
		t.Parallel()

		svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
			Event:      7,
			PoolLength: 50,
		}, newTestMeetStore(t, tenLaneFiles), nil, nil)

		out := runSimulator(t, svc)

		assert.Zero(t, out.Len())
	})

	t.Run("runs only once", func(t *testing.T) {
		t.Parallel()

		svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
			Event:      7,
			PoolLength: 50,
		}, newTestMeetStore(t, tenLaneFiles), nil, nil)
		svc.started.Store(true)

		// runSimulator fails the test if Run does not return in time
		runSimulator(t, svc)
	})
}

// newEventSimulator returns a simulator for event 7 in a 50 m pool that generates the heats without waiting.
func newEventSimulator(t *testing.T, files map[string]string) *SimulatorService {
	t.Helper()

	return NewSimulatorService(
		zerolog.Nop(),
		SimulatorOptions{
			Event:      7,
			PoolLength: 50,
		},
		newTestMeetStore(t, files),
		nil,
		nil,
		WithSimulatorClock(&instantClock{
			now: time.Unix(0, 0),
		}),
		WithSimulatorRandSeed(7),
	)
}

// runFirstHeat runs svc until the first official frame and returns the frames it wrote.
func runFirstHeat(t *testing.T, svc *SimulatorService) []omega.Frame {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	writer := &firstHeat{
		cancel: cancel,
	}

	done := make(chan struct{})
	go func() {
		svc.Run(ctx, writer)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Run did not finish the heat")
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.frames
}

// runSimulator runs a simulator for event with a canceled context and reports whether it returned in time.
func runSimulator(t *testing.T, svc *SimulatorService) *bytes.Buffer {
	t.Helper()

	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})
	go func() {
		svc.Run(ctx, &out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Run did not return")
	}
	return &out
}

// instantClock lets a simulated heat or a replay run without waiting, see WithSimulatorClock.
type instantClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the current fake time.
func (c *instantClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// After advances the fake time by d and returns a channel that already holds the new time.
func (c *instantClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	c.now = c.now.Add(d)
	ch := make(chan time.Time, 1)
	ch <- c.now
	c.mu.Unlock()
	return ch
}

// firstHeat records frames until the first official frame and then cancels the run.
type firstHeat struct {
	mu     sync.Mutex
	frames []omega.Frame
	done   bool
	cancel context.CancelFunc
}

// Write records the frame and cancels the run after the first official frame.
func (w *firstHeat) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done {
		return len(p), nil
	}

	frame, err := omega.NewDecoder(bytes.NewReader(p)).Next()
	if err == nil {
		w.frames = append(w.frames, frame)
		if frame.Kind() == omega.KindOfficial {
			w.done = true
			w.cancel()
		}
	}
	return len(p), nil
}

func TestSimulatorService_lapCount(t *testing.T) {
	t.Parallel()

	event := func(length int) *domain.Event {
		return &domain.Event{
			Distance: &domain.Distance{
				Length: length,
			},
		}
	}

	tests := []struct {
		name       string
		poolLength int
		event      *domain.Event
		found      bool
		want       int
	}{
		{
			name:       "returns the laps of a long course",
			poolLength: 50,
			event:      event(200),
			found:      true,
			want:       4,
		},
		{
			name:       "returns the laps of a short course",
			poolLength: 25,
			event:      event(200),
			found:      true,
			want:       8,
		},
		{
			name:       "assumes 50 m for an automatic pool length",
			poolLength: 0,
			event:      event(200),
			found:      true,
			want:       4,
		},
		{
			name:       "returns one lap for a distance shorter than the pool",
			poolLength: 50,
			event:      event(25),
			found:      true,
			want:       1,
		},
		{
			name:       "returns two laps for an unknown event",
			poolLength: 50,
			event:      nil,
			found:      false,
			want:       2,
		},
		{
			name:       "returns two laps for an event without distance",
			poolLength: 50,
			event:      &domain.Event{},
			found:      true,
			want:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
				Event:      7,
				PoolLength: tt.poolLength,
			}, nil, nil, nil)

			laps := svc.lapCount(tt.event, tt.found)

			assert.Equal(t, tt.want, laps)
		})
	}
}

func TestSimulatorService_replay(t *testing.T) {
	t.Parallel()

	t.Run("writes the data with the recorded pauses", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Second), 0x01),
			gapEntry(at.Add(2*time.Second)),
			dataEntry(at.Add(3*time.Second), 0x02),
		)
		svc, clock := newRecordingSimulator(t, recordings, 0, nil)
		writer := newReplayWriter(clock, 2)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, [][]byte{
			{0x01},
			{0x02},
		}, writer.data)
		assert.Equal(t, []time.Duration{
			replayPassPause,
			replayPassPause + 2*time.Second,
		}, writer.at, "the pause before the pass, then the recorded pauses from the first data on")
	})

	t.Run("starts again after a pause at the end", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Second), 0x01),
		)
		svc, clock := newRecordingSimulator(t, recordings, 0, nil)
		writer := newReplayWriter(clock, 2)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, []time.Duration{
			replayPassPause,
			2 * replayPassPause,
		}, writer.at)
	})

	t.Run("counts a clock set back between two lines as no pause", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(5*time.Second), 0x01),
			sessionEntry(at.Add(time.Second)),
			dataEntry(at.Add(2*time.Second), 0x02),
		)
		svc, clock := newRecordingSimulator(t, recordings, 0, nil)
		writer := newReplayWriter(clock, 2)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, []time.Duration{
			replayPassPause,
			replayPassPause + time.Second,
		}, writer.at)
	})

	t.Run("writes only the heats of the event with the recorded pauses", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Minute), loadFrame(102, 1)...),
			dataEntry(at.Add(2*time.Minute), loadFrame(101, 1)...),
			dataEntry(at.Add(3*time.Minute), loadFrame(101, 2)...),
			dataEntry(at.Add(4*time.Minute), loadFrame(103, 1)...),
		)
		svc, clock := newRecordingSimulator(t, recordings, 101, nil)
		writer := newReplayWriter(clock, 2)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, [][]byte{
			loadFrame(101, 1),
			loadFrame(101, 2),
		}, writer.data)
		assert.Equal(t, []time.Duration{
			replayPassPause,
			replayPassPause + time.Minute,
		}, writer.at, "the pass starts with the event and keeps the pause between its heats")
	})

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "stops for a recording without data",
			content: mustMarshalLines(t, sessionEntry(time.Now())),
		},
		{
			name:    "stops for a broken recording",
			content: mustMarshalLines(t, dataEntry(time.Now(), 0x01)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recordings := writeRawRecording(t, tt.content)
			svc, clock := newRecordingSimulator(t, recordings, 0, nil)
			writer := newReplayWriter(clock, 1)

			runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

			assert.Empty(t, writer.data)
		})
	}

	t.Run("stops for a missing file", func(t *testing.T) {
		t.Parallel()

		svc, clock := newRecordingSimulator(t, store.NewRecordingStore(newTestStorage(t)), 0, nil)
		writer := newReplayWriter(clock, 1)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Empty(t, writer.data)
	})

	t.Run("stops for an event without heats in the recording", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Minute), loadFrame(102, 1)...),
		)
		svc, clock := newRecordingSimulator(t, recordings, 101, nil)
		writer := newReplayWriter(clock, 1)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Empty(t, writer.data)
	})

	t.Run("measures the clock anew at every session of every pass", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		recordings := writeRecording(t,
			sessionEntry(at),
			dataEntry(at.Add(time.Second), 0x01),
			sessionEntry(at.Add(time.Minute)),
			dataEntry(at.Add(time.Minute+time.Second), 0x02),
		)
		var jumps atomic.Int32
		svc, clock := newRecordingSimulator(t, recordings, 0, func() { jumps.Add(1) })
		writer := newReplayWriter(clock, 4)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, int32(4), jumps.Load(), "two sessions in each of two passes")
	})

	t.Run("warns once per pass about other meet files", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		other := recording.Entry{
			Type: recording.TypeMeet,
			At:   at,
			Files: []recording.MeetFile{
				recording.NewMeetFile("LSTSTART", []byte("other")),
			},
		}
		recordings := writeRecording(t,
			sessionEntry(at),
			other,
			dataEntry(at.Add(time.Second), 0x01),
			other,
			dataEntry(at.Add(2*time.Second), 0x02),
		)
		var logs bytes.Buffer
		svc, clock := newRecordingSimulator(t, recordings, 0, nil)
		svc.logger = zerolog.New(&syncWriter{
			writer: &logs,
		})
		writer := newReplayWriter(clock, 2)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.Equal(t, 1, strings.Count(logs.String(), "other meet files"))
	})

	t.Run("checks only the meet files in effect while data is sent", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		var logs bytes.Buffer
		meetStore := newTestMeetStore(t, tenLaneFiles)
		var files []recording.MeetFile
		for _, content := range meetStore.ListFileContents() {
			files = append(files, recording.NewMeetFile(string(content.File), content.Content))
		}
		recordings := writeRecording(t,
			sessionEntry(at),
			recording.Entry{
				Type: recording.TypeMeet,
				At:   at,
				Files: []recording.MeetFile{
					recording.NewMeetFile("LSTSTART", []byte("other")),
				},
			},
			dataEntry(at.Add(time.Minute), loadFrame(102, 1)...),
			recording.Entry{
				Type:  recording.TypeMeet,
				At:    at.Add(2 * time.Minute),
				Files: files,
			},
			dataEntry(at.Add(3*time.Minute), loadFrame(101, 1)...),
		)
		clock := &instantClock{
			now: time.Unix(0, 0),
		}
		svc := NewSimulatorService(zerolog.New(&syncWriter{
			writer: &logs,
		}), SimulatorOptions{
			Event:      101,
			ReplayPath: testReplayPath,
		}, meetStore, recordings, nil, WithSimulatorClock(clock))
		writer := newReplayWriter(clock, 1)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.NotContains(t, logs.String(), "other meet files", "the other files are no longer in effect for the event")
	})

	t.Run("does not warn for the loaded meet files", func(t *testing.T) {
		t.Parallel()

		at := time.Now()
		var logs bytes.Buffer
		meetStore := newTestMeetStore(t, tenLaneFiles)
		var files []recording.MeetFile
		for _, content := range meetStore.ListFileContents() {
			files = append(files, recording.NewMeetFile(string(content.File), content.Content))
		}
		recordings := writeRecording(t,
			sessionEntry(at),
			recording.Entry{
				Type:  recording.TypeMeet,
				At:    at,
				Files: files,
			},
			dataEntry(at.Add(time.Second), 0x01),
		)
		clock := &instantClock{
			now: time.Unix(0, 0),
		}
		svc := NewSimulatorService(zerolog.New(&syncWriter{
			writer: &logs,
		}), SimulatorOptions{
			ReplayPath: testReplayPath,
		}, meetStore, recordings, nil, WithSimulatorClock(clock))
		writer := newReplayWriter(clock, 1)

		runReplay(t, writer, func(ctx context.Context) { svc.replay(ctx, writer) })

		assert.NotContains(t, logs.String(), "other meet files")
	})
}

// newRecordingSimulator returns a simulator that replays the recording at testReplayPath without waiting,
// all of it or only the heats of event, and its clock.
func newRecordingSimulator(
	t *testing.T,
	recordings *store.RecordingStore,
	event int,
	onClockJump func(),
) (*SimulatorService, *instantClock) {
	t.Helper()

	clock := &instantClock{
		now: time.Unix(0, 0),
	}
	svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
		Event:      event,
		ReplayPath: testReplayPath,
	}, newTestMeetStore(t, tenLaneFiles), recordings, onClockJump, WithSimulatorClock(clock))
	return svc, clock
}

// runReplay runs replay until writer has received its writes or replay returns.
func runReplay(t *testing.T, writer *replayWriter, replay func(ctx context.Context)) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	writer.mu.Lock()
	writer.cancel = cancel
	writer.mu.Unlock()

	done := make(chan struct{})
	go func() {
		replay(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the replay did not return")
	}
}

// replayWriter records every write with the time of the clock since its start
// and cancels the replay after a number of writes.
type replayWriter struct {
	clock *instantClock
	start time.Time
	limit int

	mu     sync.Mutex
	cancel context.CancelFunc
	data   [][]byte
	at     []time.Duration
}

// newReplayWriter creates a replayWriter that cancels the replay after limit writes.
func newReplayWriter(clock *instantClock, limit int) *replayWriter {
	return &replayWriter{
		clock: clock,
		start: clock.Now(),
		limit: limit,
	}
}

// Write records the data and the time of the write.
func (w *replayWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.data = append(w.data, bytes.Clone(p))
	w.at = append(w.at, w.clock.Now().Sub(w.start))
	if len(w.data) >= w.limit && w.cancel != nil {
		w.cancel()
	}
	return len(p), nil
}

// syncWriter serializes the writes of a logger used by several goroutines.
type syncWriter struct {
	mu     sync.Mutex
	writer *bytes.Buffer
}

// Write writes p under the lock.
func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

func TestReplayWindows(t *testing.T) {
	t.Parallel()

	at := time.Now()

	t.Run("selects the reads from the first to the last frame of every heat of the event", func(t *testing.T) {
		t.Parallel()

		entries := []recording.Entry{
			sessionEntry(at),
			dataEntry(at, loadFrame(101, 1)...),
			dataEntry(at, officialFrame(102, 1)...),
			dataEntry(at, 'x'),
			dataEntry(at, officialFrame(101, 1)...),
			dataEntry(at, loadFrame(103, 1)...),
		}

		got := replayWindows(entries, 101)

		assert.Equal(t, []bool{
			false,
			true,
			true,
			true,
			true,
			false,
		}, got, "the frames of other events and the noise within the heat stay")
	})

	t.Run("ends a heat with the next load frame", func(t *testing.T) {
		t.Parallel()

		entries := []recording.Entry{
			sessionEntry(at),
			dataEntry(at, loadFrame(101, 1)...),
			dataEntry(at, loadFrame(102, 1)...),
			dataEntry(at, officialFrame(102, 1)...),
			dataEntry(at, officialFrame(101, 1)...),
		}

		got := replayWindows(entries, 101)

		assert.Equal(t, []bool{
			false,
			true,
			false,
			false,
			true,
		}, got, "a late frame of the heat starts a short window of its own")
	})

	t.Run("starts a heat with the read that brought the first byte of its frame", func(t *testing.T) {
		t.Parallel()

		frame := loadFrame(101, 1)
		entries := []recording.Entry{
			sessionEntry(at),
			dataEntry(at, loadFrame(102, 1)...),
			dataEntry(at, frame[:10]...),
			dataEntry(at, frame[10:]...),
		}

		got := replayWindows(entries, 101)

		assert.Equal(t, []bool{
			false,
			false,
			true,
			true,
		}, got)
	})

	t.Run("decodes every session on its own", func(t *testing.T) {
		t.Parallel()

		frame := loadFrame(101, 1)
		entries := []recording.Entry{
			sessionEntry(at),
			dataEntry(at, frame[:20]...),
			sessionEntry(at),
			dataEntry(at, frame[20:]...),
		}

		got := replayWindows(entries, 101)

		assert.NotContains(t, got, true, "parts of different sessions are never combined")
	})
}

// loadFrame encodes the load frame of a heat.
func loadFrame(event, heat int) []byte {
	return omega.Encode(omega.FrameData{
		Kind:      omega.KindLoad,
		UsedLanes: []int{4},
		LapCount:  2,
		Event:     event,
		Heat:      heat,
	})
}

// officialFrame encodes the official frame of a heat.
func officialFrame(event, heat int) []byte {
	return omega.Encode(omega.FrameData{
		Kind:      omega.KindOfficial,
		UsedLanes: []int{4},
		LapCount:  2,
		Event:     event,
		Heat:      heat,
	})
}

func TestSimulatorService_matchesMeet(t *testing.T) {
	t.Parallel()

	changed := func(name, content string) map[string]string {
		files := maps.Clone(tenLaneFiles)
		files[name] = content
		return files
	}

	tests := []struct {
		name     string
		event    int
		recorded map[string]string
		want     bool
	}{
		{
			name:     "matches the loaded files",
			event:    0,
			recorded: tenLaneFiles,
			want:     true,
		},
		{
			name:     "rejects other files without an event",
			event:    0,
			recorded: changed("LSTSTART.TXT", "event;round;heat;lane;relais;idBib\n7;2;0;1;0;1;\n"),
			want:     false,
		},
		{
			name:     "matches other files that show the event the same way",
			event:    7,
			recorded: changed("LSTSTART.TXT", tenLaneFiles["LSTSTART.TXT"]+"8;1;0;2;0;1;\n"),
			want:     true,
		},
		{
			name:     "rejects a changed start entry of the event",
			event:    7,
			recorded: changed("LSTSTART.TXT", "event;round;heat;lane;relais;idBib\n7;2;0;2;0;1;\n7;2;0;10;0;10;\n7;1;0;5;0;1;\n"),
			want:     false,
		},
		{
			name:     "rejects a changed competitor of the event",
			event:    7,
			recorded: changed("LSTCONC.TXT", strings.Replace(tenLaneFiles["LSTCONC.TXT"], "Erika", "Erik", 1)),
			want:     false,
		},
		{
			name:     "rejects a changed name of the event",
			event:    7,
			recorded: changed("LSTSTYLE.TXT", "idStyle;Style;StyleAbrév\n3;\"Freestyle\";\"FR\"\n"),
			want:     false,
		},
		{
			name:     "rejects files that cannot be read",
			event:    7,
			recorded: changed("LSTRACE.TXT", "event;round;nbHeat\nx;y;z\n"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewSimulatorService(zerolog.Nop(), SimulatorOptions{
				Event: tt.event,
			}, newTestMeetStore(t, tenLaneFiles), nil, nil)

			got := svc.matchesMeet(recordedMeetFiles(t, tt.recorded))

			assert.Equal(t, tt.want, got)
		})
	}
}

// recordedMeetFiles returns meet files as a recording keeps them, encoded as Windows-1252.
func recordedMeetFiles(t *testing.T, files map[string]string) []recording.MeetFile {
	t.Helper()

	recorded := make([]recording.MeetFile, 0, len(files))
	for name, content := range files {
		encoded, err := charmap.Windows1252.NewEncoder().String(content)
		require.NoError(t, err)
		recorded = append(recorded, recording.NewMeetFile(strings.TrimSuffix(name, ".TXT"), []byte(encoded)))
	}
	return recorded
}
