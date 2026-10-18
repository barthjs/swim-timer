package service

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/serial"
	"github.com/barthjs/swim-timer/internal/store"
)

// fakeDevice reports a fixed port and never delivers frames.
type fakeDevice struct {
	port string
}

// Run returns at once, since the fake device delivers no frames.
func (f fakeDevice) Run(context.Context, serial.Handler) {}

// Port returns the fixed port of the fake device.
func (f fakeDevice) Port() string {
	return f.port
}

// Ping reports the fake device as reachable.
func (f fakeDevice) Ping(context.Context) error {
	return nil
}

// streamDevice delivers a fixed byte stream once.
type streamDevice struct {
	fakeDevice
	stream []byte
}

// Run hands the stream to the handler and returns once the handler has read it.
func (d streamDevice) Run(ctx context.Context, handle serial.Handler) {
	_ = handle(ctx, bytes.NewReader(d.stream))
}

func TestNewTimingService(t *testing.T) {
	t.Parallel()

	t.Run("keeps the recorder", func(t *testing.T) {
		t.Parallel()

		recorder := newTestRecorder(t.TempDir())

		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, &fakeBroadcaster{}, fakeDevice{}, recorder, newTestTimingService(t, 50).store)

		assert.Same(t, recorder, svc.recorder)
	})
}

func TestTimingService_GetLiveHeat(t *testing.T) {
	t.Parallel()

	t.Run("returns a copy of the live heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"))

		copied := svc.GetLiveHeat()

		copied.Lanes[4].Time = 99
		*copied.StartTime = time.Time{}
		copied.Lanes[8] = &domain.Lane{
			Number: 8,
		}
		state := svc.GetLiveHeat()
		assert.Zero(t, state.Lanes[4].Time, "changing a returned lane changes the live heat")
		assert.False(t, state.StartTime.IsZero(), "changing the returned start changes the live heat")
		assert.NotContains(t, state.Lanes, 8, "adding a lane to the returned state changes the live heat")
	})
}

func TestTimingService_GetStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		port string
		want TimingStatus
	}{
		{
			name: "returns the status of a connected device",
			port: "/dev/ttyUSB0",
			want: TimingStatus{
				Connected:      true,
				Port:           "/dev/ttyUSB0",
				ConfiguredPort: "COM3",
				SimulateEvent:  7,
				ReplayFile:     "recording_2026-10-18.jsonl",
			},
		},
		{
			name: "returns the status of a disconnected device",
			port: "",
			want: TimingStatus{
				Connected:      false,
				Port:           "",
				ConfiguredPort: "COM3",
				SimulateEvent:  7,
				ReplayFile:     "recording_2026-10-18.jsonl",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options := TimingOptions{
				ConfiguredPort: "COM3",
				SimulateEvent:  7,
				ReplayFile:     "recording_2026-10-18.jsonl",
			}
			svc := NewTimingService(zerolog.Nop(), options, &fakeBroadcaster{}, fakeDevice{
				port: tt.port,
			}, nil, nil)

			status := svc.GetStatus()

			assert.Equal(t, tt.want, status)
		})
	}
}

func TestTimingService_Run(t *testing.T) {
	t.Parallel()

	t.Run("records the meet files and the received data", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		stream := omega.Encode(omega.FrameData{
			Kind:      omega.KindLoad,
			UsedLanes: []int{4},
			LapCount:  4,
			Event:     101,
			Heat:      2,
		})
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, &fakeBroadcaster{}, streamDevice{
			stream: stream,
		}, newTestRecorder(dir), newTestTimingService(t, 50).store)

		svc.Run(t.Context())

		entries := readRecording(t, dir)
		require.Len(t, entries, 3, "the session, the meet files, and the data")
		assert.Equal(t, recording.TypeSession, entries[0].Type)
		assert.Equal(t, recording.TypeMeet, entries[1].Type)
		assert.Equal(t, []string{
			"LSTCONC",
			"LSTLONG",
			"LSTRACE",
			"LSTSTART",
		}, meetFileNames(entries[1].Files))
		assert.Equal(t, recording.HexBytes(stream), entries[2].Data)
		assert.Equal(t, 2, svc.GetLiveHeat().Heat, "the recorded data reaches the live heat")
	})

	t.Run("returns at once when it already runs", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.started.Store(true)

		done := make(chan struct{})
		go func() {
			svc.Run(t.Context())
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "second Run did not return immediately")
		}
	})
}

func TestTimingService_Snapshot(t *testing.T) {
	t.Parallel()

	t.Run("returns the live heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '0', ' ', 4, 101, 1, 0, 0, 0, ""))

		raw, err := svc.Snapshot()

		require.NoError(t, err)
		var message map[string]jsontext.Value
		require.NoError(t, json.Unmarshal(raw, &message))
		assert.JSONEq(t, `"init"`, string(message["type"]))
		assert.Contains(t, message, "serverTime")
		require.Contains(t, message, "data")
		var data map[string]jsontext.Value
		require.NoError(t, json.Unmarshal(message["data"], &data))
		assert.NotContains(t, data, "startTime", "no start time before any start")
		assert.NotEqual(t, "null", string(data["lanes"]), "lanes are an object")
	})

	t.Run("returns an empty heat before the first frame", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)

		raw, err := svc.Snapshot()

		require.NoError(t, err)
		var message struct {
			Data map[string]jsontext.Value `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &message))
		assert.JSONEq(t, "{}", string(message.Data["lanes"]))
		assert.JSONEq(t, `""`, string(message.Data["status"]))
	})
}

func TestTimingService_RecordMeet(t *testing.T) {
	t.Parallel()

	t.Run("records the loaded meet files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		meetStore := newTestTimingService(t, 50).store
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, &fakeBroadcaster{}, fakeDevice{}, newTestRecorder(dir), meetStore)
		svc.recorder.Start()

		svc.RecordMeet()

		svc.recorder.Close()
		entries := readRecording(t, dir)
		require.Len(t, entries, 2)
		files := entries[1].Files
		contents := meetStore.ListFileContents()
		require.Len(t, files, len(contents))
		for i, content := range contents {
			assert.Equal(t, string(content.File), files[i].Name)
			assert.Equal(t, content.Content, files[i].Content)
			assert.Equal(t, recording.Hash(content.Content), files[i].SHA256)
		}
	})

	t.Run("does nothing without recording", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)

		svc.RecordMeet()

		assert.Nil(t, svc.recorder)
	})
}

func TestTimingService_ResetClock(t *testing.T) {
	t.Parallel()

	t.Run("measures the next start anew", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		svc.correctStart(frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, deviceClock(first, time.Minute)), first)

		svc.ResetClock()

		now := time.Now()
		correction := svc.correctStart(frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, deviceClock(now, -time.Hour)), now)
		assert.WithinDuration(t, now, correction.at, 50*time.Millisecond, "the jumped clock does not count as a delay")
		assert.Equal(t, time.Hour, svc.clock.baseline.Round(time.Second))
	})
}

func TestTimingService_handleFrame(t *testing.T) {
	t.Parallel()

	t.Run("shows a loaded heat with its competitors", func(t *testing.T) {
		t.Parallel()

		competitor := newTestCompetitor(t)
		svc := newTestTimingServiceFor(t, 50, competitor)

		svc.handleFrame(frame(t, '0', ' ', 4, 101, 2, 0, 0, 0, ""))

		state := svc.GetLiveHeat()
		assert.Equal(t, 101, state.Event)
		assert.Equal(t, 2, state.Heat)
		assert.Equal(t, 2, state.HeatCount)
		assert.Equal(t, 4, state.LapCount)
		assert.Equal(t, domain.HeatStatusLoad, state.Status)
		assert.Nil(t, state.StartTime)
		// The device counts heats from 1, the start list from 0
		lane := state.Lanes[4]
		require.NotNil(t, lane, "the competitor of the second heat")
		assert.Equal(t, competitor.FirstName, lane.FirstName)
		assert.Equal(t, competitor.LastName, lane.LastName)
		assert.Equal(t, competitor.BirthYear, lane.BirthYear)
		assert.Equal(t, competitor.Club(), lane.Club)
	})

	t.Run("shows the name and phase of the event", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)

		svc.handleFrame(frame(t, '0', ' ', 2, 7, 1, 0, 0, 0, ""))

		state := svc.GetLiveHeat()
		assert.Equal(t, "100 m Backstroke Open", state.Name, "distance, style, and category")
		assert.Equal(t, 2, state.Phase)
		assert.Equal(t, "Semifinal", state.PhaseName)
		assert.Equal(t, 1, state.FirstLane, "the pool is numbered from 1")
	})

	t.Run("shows a split", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"))

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 1, lane.Lap)
		assert.Equal(t, 50, lane.Meter)
		assert.Equal(t, 30100, lane.Time)
		assert.Equal(t, 1, lane.Rank)
		assert.False(t, lane.Finished)
	})

	t.Run("replaces the time of a corrected split", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"), frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"))

		// A time corrected at the device replaces the earlier time of the same lap
		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 2, 4, 1, "      30.30"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 30300, lane.Time)
		assert.Equal(t, 1, lane.Rank, "the rank comes from the times, not from the device")
	})

	t.Run("shows a finish", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"), frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"))

		svc.handleFrame(frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 4, lane.Lap)
		assert.Equal(t, 200, lane.Meter)
		assert.Equal(t, 125000, lane.Time)
		assert.True(t, lane.Finished)
	})

	t.Run("keeps the finish on display after a late correction of an earlier lap", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"),
			frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"),
		)

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 4, 2, "    1:01.00"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 4, lane.Lap)
		assert.Equal(t, 125000, lane.Time)
		assert.True(t, lane.Finished)
	})

	t.Run("computes the meters of a lap", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			poolLength int
			lapCount   int
			wantMeter  int
		}{
			{
				name:       "from the configured pool length",
				poolLength: 50,
				lapCount:   8,
				wantMeter:  100,
			},
			{
				name:       "from the distance in eight laps",
				poolLength: 0,
				lapCount:   8,
				wantMeter:  50,
			},
			{
				name:       "from the distance in four laps",
				poolLength: 0,
				lapCount:   4,
				wantMeter:  100,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				svc := newTestTimingService(t, tt.poolLength)
				svc.handleFrame(frame(t, '2', 'S', tt.lapCount, 101, 1, 0, 1, 0, " 9:00:00.00"))

				svc.handleFrame(frame(t, '2', 'I', tt.lapCount, 101, 1, 1, 4, 2, "      58.10"))

				require.NotNil(t, svc.GetLiveHeat().Lanes[4])
				assert.Equal(t, tt.wantMeter, svc.GetLiveHeat().Lanes[4].Meter)
			})
		}
	})

	t.Run("ranks the lanes by their times", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		handleFrames(svc, frame(t, '2', 'S', 2, 7, 1, 0, 1, 0, "10:00:00.00"), frame(t, '2', 'I', 2, 7, 1, 1, 1, 1, "      30.00"))

		svc.handleFrame(frame(t, '2', 'I', 2, 7, 1, 2, 10, 1, "      31.00"))

		lanes := svc.GetLiveHeat().Lanes
		assert.Equal(t, 1, lanes[1].Rank)
		assert.Equal(t, 2, lanes[10].Rank)
	})

	t.Run("ranks the lanes again after a faster time entered late", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		handleFrames(
			svc,
			frame(t, '2', 'S', 2, 7, 1, 0, 1, 0, "10:00:00.00"),
			frame(t, '2', 'I', 2, 7, 1, 1, 1, 1, "      30.00"),
			frame(t, '2', 'I', 2, 7, 1, 2, 10, 1, "      31.00"),
		)

		// The device sends rank 1 only for lane 10, yet both lanes are ranked again
		svc.handleFrame(frame(t, '2', 'I', 2, 7, 1, 1, 10, 1, "      29.50"))

		lanes := svc.GetLiveHeat().Lanes
		assert.Equal(t, 2, lanes[1].Rank)
		assert.Equal(t, 1, lanes[10].Rank)
	})

	t.Run("shows lane 10 for the lane digit 0", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		svc.handleFrame(frame(t, '2', 'S', 2, 7, 1, 0, 0, 0, "10:00:00.00"))

		svc.handleFrame(frame(t, '2', 'A', 2, 7, 1, 1, 10, 2, "    1:01.50"))

		lane := svc.GetLiveHeat().Lanes[10]
		require.NotNil(t, lane)
		assert.Equal(t, 61500, lane.Time, "the finish transmitted for lane digit 0")
		assert.Equal(t, 1, lane.Rank)
		assert.True(t, lane.Finished)
	})

	t.Run("shows lane 0 of a pool numbered from 0", func(t *testing.T) {
		t.Parallel()

		files := maps.Clone(tenLaneFiles)
		files["LSTSTART.TXT"] = "event;round;heat;lane;relais;idBib\n7;2;0;0;0;1;\n7;2;0;9;0;10;\n"
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, &fakeBroadcaster{}, fakeDevice{}, nil, newTestMeetStore(t, files))
		svc.handleFrame(frame(t, '2', 'S', 2, 7, 1, 0, 1, 0, "10:00:00.00"))

		svc.handleFrame(frame(t, '2', 'A', 2, 7, 1, 1, 10, 2, "    1:01.50"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 0, state.FirstLane)
		require.Contains(t, state.Lanes, 0)
		assert.Equal(t, 61500, state.Lanes[0].Time, "the lane digit 0 is lane 0")
		assert.NotContains(t, state.Lanes, 10)
	})

	t.Run("shows only the start entries of the phase of the event", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)

		// Every lane from 1 to 10 is in use
		svc.handleFrame(specFrame(t, frameSpec{
			msgType:   '0',
			msgKind:   ' ',
			timeType:  ' ',
			usedLanes: "??",
			lapCount:  2,
			event:     7,
			heat:      1,
		}))

		lanes := svc.GetLiveHeat().Lanes
		assert.Len(t, lanes, 2, "lanes 1 and 10 of phase 2 only")
		assert.Contains(t, lanes, 1)
		assert.Contains(t, lanes, 10)
	})

	t.Run("shows a heat of an unknown event without meet data", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		handleFrames(svc, frame(t, '0', ' ', 2, 999, 1, 0, 0, 0, ""), frame(t, '2', 'S', 2, 999, 1, 0, 1, 0, "10:00:00.00"))

		svc.handleFrame(frame(t, '2', 'I', 2, 999, 1, 1, 3, 1, "      30.10"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 999, state.Event, "the heat is shown without meet data")
		assert.Empty(t, state.Name)
		require.Contains(t, state.Lanes, 3)
		assert.Equal(t, 30100, state.Lanes[3].Time)
		assert.Empty(t, state.Lanes[3].LastName)
	})

	t.Run("shows the competitors of a reloaded start list", func(t *testing.T) {
		t.Parallel()

		files := maps.Clone(tenLaneFiles)
		delete(files, "LSTSTART.TXT")
		meetStore, root := newTestMeetStoreWithRoot(t, files)
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, &fakeBroadcaster{}, fakeDevice{}, nil, meetStore)
		svc.handleFrame(frame(t, '0', ' ', 2, 7, 1, 0, 0, 0, ""))
		require.Empty(t, svc.GetLiveHeat().Lanes, "precondition: the start list is not loaded yet")
		writeMeetFiles(t, root, map[string]string{
			"LSTSTART.TXT": tenLaneFiles["LSTSTART.TXT"],
		})
		require.NoError(t, meetStore.LoadAll(t.Context()))

		svc.handleFrame(frame(t, '2', 'I', 2, 7, 1, 1, 1, 1, "      30.10"))

		lane := svc.GetLiveHeat().Lanes[1]
		require.NotNil(t, lane)
		assert.Equal(t, "One", lane.LastName, "the next frame takes the competitors from the reloaded start list")
		assert.Equal(t, 30100, lane.Time)
	})

	t.Run("shows a lane without a start entry", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 7, 1, "      30.10"))

		lane := svc.GetLiveHeat().Lanes[7]
		require.NotNil(t, lane, "someone swims in the lane")
		assert.Equal(t, 30100, lane.Time)
		assert.Empty(t, lane.LastName)
	})

	t.Run("shows the heat of a time for another heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"))

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 2, 1, 4, 1, "      31.20"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 2, state.Heat, "a time shows its heat")
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		assert.Equal(t, 31200, state.Lanes[4].Time)
	})

	t.Run("keeps the times of a heat shown before", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"),
			frame(t, '2', 'I', 4, 101, 2, 1, 4, 1, "      31.20"),
		)

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 4, 2, "    1:02.00"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 1, state.Heat)
		assert.Equal(t, 2, state.Lanes[4].Lap, "the heat shown before keeps its times")
		assert.NotNil(t, state.StartTime)
	})

	t.Run("shows the official result of the heat on display", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"), frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"))

		svc.handleFrame(frame(t, '1', ' ', 4, 101, 1, 0, 0, 0, ""))

		state := svc.GetLiveHeat()
		assert.Equal(t, domain.HeatStatusOfficial, state.Status)
		assert.NotNil(t, state.StartTime, "the start time stays after the official frame")
		require.NotNil(t, state.Lanes[4])
		assert.Equal(t, 125000, state.Lanes[4].Time, "the result stays")
		assert.True(t, state.Lanes[4].Finished)
	})

	t.Run("publishes the live heat", func(t *testing.T) {
		t.Parallel()

		broadcaster := &fakeBroadcaster{}
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, broadcaster, fakeDevice{}, nil, newTestTimingService(t, 50).store)
		before := time.Now()

		svc.handleFrame(frame(t, '0', ' ', 4, 101, 1, 0, 0, 0, ""))

		require.Equal(t, 1, broadcaster.count())
		var message struct {
			Type       string          `json:"type"`
			ServerTime time.Time       `json:"serverTime"`
			Data       domain.LiveHeat `json:"data"`
		}
		require.NoError(t, json.Unmarshal(broadcaster.messages[0], &message))
		assert.Equal(t, "update", message.Type)
		assert.Equal(t, 101, message.Data.Event)
		assert.Contains(t, message.Data.Lanes, 4)
		assert.WithinRange(t, message.ServerTime, before.Add(-time.Second), time.Now().Add(time.Second), "the time the message was built")
	})

	t.Run("ignores reaction times and unknown frame types", func(t *testing.T) {
		t.Parallel()

		broadcaster := &fakeBroadcaster{}
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, broadcaster, fakeDevice{}, nil, newTestTimingService(t, 50).store)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, " 9:00:00.00"))

		handleFrames(svc, frame(t, '2', 'R', 4, 101, 1, 0, 4, 0, "     + 0.65"), frame(t, '5', 'B', 4, 101, 1, 0, 4, 1, "      35.12"))

		assert.Equal(t, 1, broadcaster.count(), "reaction times and unknown types are ignored")
		assert.Zero(t, svc.GetLiveHeat().Lanes[4].Time)
	})

	t.Run("ignores frames without a valid lap count, event, or heat", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			lapCount string
			event    string
			heat     string
		}{
			{
				name:     "an empty lap count",
				lapCount: "  ",
				event:    "101",
				heat:     " 1",
			},
			{
				name:     "an invalid lap count",
				lapCount: " x",
				event:    "101",
				heat:     " 1",
			},
			{
				name:     "an empty event",
				lapCount: " 4",
				event:    "   ",
				heat:     " 1",
			},
			{
				name:     "an invalid event",
				lapCount: " 4",
				event:    "1x1",
				heat:     " 1",
			},
			{
				name:     "an empty heat",
				lapCount: " 4",
				event:    "101",
				heat:     "  ",
			},
			{
				name:     "an invalid heat",
				lapCount: " 4",
				event:    "101",
				heat:     "-1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				broadcaster := &fakeBroadcaster{}
				svc := NewTimingService(zerolog.Nop(), TimingOptions{
					PoolLength: 50,
				}, broadcaster, fakeDevice{}, nil, newTestTimingService(t, 50).store)

				handleFrames(
					svc,
					textFrame(t, '0', ' ', tt.lapCount, tt.event, tt.heat, " ", "  ", ""),
					textFrame(t, '2', 'S', tt.lapCount, tt.event, tt.heat, "1", " 0", " 9:00:00.00"),
					textFrame(t, '2', 'I', tt.lapCount, tt.event, tt.heat, "4", " 1", "      30.10"),
					textFrame(t, '2', 'A', tt.lapCount, tt.event, tt.heat, "4", " 4", "    2:05.00"),
				)

				assert.Zero(t, broadcaster.count(), "frames without a heat are ignored")
				assert.Zero(t, svc.GetLiveHeat().Event)
			})
		}
	})

	t.Run("ignores times without a valid lane, lap, or time", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			lane  string
			lap   string
			clock string
		}{
			{
				name:  "an empty lane",
				lane:  " ",
				lap:   " 1",
				clock: "      30.10",
			},
			{
				name:  "an invalid lane",
				lane:  "x",
				lap:   " 1",
				clock: "      30.10",
			},
			{
				name:  "an empty lap",
				lane:  "4",
				lap:   "  ",
				clock: "      30.10",
			},
			{
				name:  "an invalid lap",
				lane:  "4",
				lap:   " x",
				clock: "      30.10",
			},
			{
				name:  "a lap below one",
				lane:  "4",
				lap:   " 0",
				clock: "      30.10",
			},
			{
				name:  "an empty time",
				lane:  "4",
				lap:   " 1",
				clock: "",
			},
			{
				name:  "an invalid time",
				lane:  "4",
				lap:   " 1",
				clock: "      3x.10",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				broadcaster := &fakeBroadcaster{}
				svc := NewTimingService(zerolog.Nop(), TimingOptions{
					PoolLength: 50,
				}, broadcaster, fakeDevice{}, nil, newTestTimingService(t, 50).store)
				svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))

				handleFrames(
					svc,
					textFrame(t, '2', 'I', " 4", "101", " 1", tt.lane, tt.lap, tt.clock),
					textFrame(t, '2', 'A', " 4", "101", " 1", tt.lane, tt.lap, tt.clock),
					textFrame(t, '7', 'B', " 4", "101", " 1", tt.lane, tt.lap, tt.clock),
				)

				assert.Equal(t, 1, broadcaster.count(), "only the start is published")
				for number, lane := range svc.GetLiveHeat().Lanes {
					assert.Zero(t, lane.Time, "lane %d", number)
				}
			})
		}
	})

	t.Run("ignores heats below one", func(t *testing.T) {
		t.Parallel()

		files := maps.Clone(tenLaneFiles)
		files["LSTSTART.TXT"] = "event;round;heat;lane;relais;idBib\n7;2;-1;3;0;10;\n7;2;0;1;0;1;\n"
		broadcaster := &fakeBroadcaster{}
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, broadcaster, fakeDevice{}, nil, newTestMeetStore(t, files))

		handleFrames(
			svc,
			frame(t, '0', ' ', 2, 7, 0, 0, 0, 0, ""),
			frame(t, '2', 'S', 2, 7, 0, 0, 1, 0, "10:00:00.00"),
			frame(t, '2', 'I', 2, 7, 0, 1, 3, 1, "      30.10"),
		)

		assert.Zero(t, broadcaster.count(), "frames for heat 0 are ignored")
		state := svc.GetLiveHeat()
		assert.Zero(t, state.Event)
		assert.Empty(t, state.Lanes, "withdrawn entries are never shown")
	})

	// Lanes 4 and 5 are in use
	stopTimeFrames := func(t *testing.T) []omega.Frame {
		t.Helper()

		timeFrame := func(msgKind byte, lane int, clock string) omega.Frame {
			return specFrame(t, frameSpec{
				msgType:   '2',
				msgKind:   msgKind,
				timeType:  ' ',
				usedLanes: "# ",
				lapCount:  4,
				event:     101,
				heat:      1,
				lane:      lane,
				lap:       4,
				clock:     clock,
			})
		}
		return []omega.Frame{
			specFrame(t, frameSpec{
				msgType:   '2',
				msgKind:   'S',
				timeType:  ' ',
				usedLanes: "# ",
				lapCount:  4,
				event:     101,
				heat:      1,
				lane:      1,
				clock:     " 9:00:00.00",
			}),
			timeFrame('A', 4, "    2:05.00"),
			timeFrame('A', 5, "    2:07.00"),
			timeFrame('I', 5, "    2:07.00"),
			timeFrame('A', 5, "    2:08.00"),
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:05:00.00"),
		}
	}

	t.Run("keeps the timer running while a lane in use swims", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := stopTimeFrames(t)
		handleFrames(svc, frames[:1]...)

		svc.handleFrame(frames[1])

		assert.Zero(t, svc.GetLiveHeat().StopTime, "a lane in use is still swimming")
	})

	t.Run("stops the timer once every lane in use finished", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := stopTimeFrames(t)
		handleFrames(svc, frames[:2]...)

		svc.handleFrame(frames[2])

		assert.Equal(t, 127000, svc.GetLiveHeat().StopTime, "the highest finish once every lane in use finished")
	})

	t.Run("lets the timer run again after a withdrawn finish", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := stopTimeFrames(t)
		handleFrames(svc, frames[:3]...)

		svc.handleFrame(frames[3])

		assert.Zero(t, svc.GetLiveHeat().StopTime, "a withdrawn finish lets the timer run again")
	})

	t.Run("stops the timer again at the next finish", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := stopTimeFrames(t)
		handleFrames(svc, frames[:4]...)

		svc.handleFrame(frames[4])

		assert.Equal(t, 128000, svc.GetLiveHeat().StopTime)
	})

	t.Run("clears the stop time on a restart", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := stopTimeFrames(t)
		handleFrames(svc, frames[:5]...)

		svc.handleFrame(frames[5])

		assert.Zero(t, svc.GetLiveHeat().StopTime, "a restart clears the stop time")
	})

	t.Run("fills a lap without a time with a button time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))

		svc.handleFrame(buttonFrame(t, 4, 1, "      30.10"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 1, lane.Lap)
		assert.Equal(t, 30100, lane.Time, "a button time fills a lap without a time")
		assert.Equal(t, 1, lane.Rank)
	})

	t.Run("keeps a time against a button time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), buttonFrame(t, 4, 1, "      30.10"))

		svc.handleFrame(buttonFrame(t, 4, 1, "      30.40"))

		assert.Equal(t, 30100, svc.GetLiveHeat().Lanes[4].Time, "a button time never replaces a time")
	})

	t.Run("replaces a button time with a touch time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), buttonFrame(t, 4, 1, "      30.10"))

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.20"))

		assert.Equal(t, 30200, svc.GetLiveHeat().Lanes[4].Time, "a touch time replaces a button time")
	})

	t.Run("keeps a touch time against a button time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), frame(t, '2', 'I', 4, 101, 1, 1, 4, 2, "    1:02.00"))

		svc.handleFrame(buttonFrame(t, 4, 2, "    1:02.30"))

		assert.Equal(t, 62000, svc.GetLiveHeat().Lanes[4].Time, "a button time never replaces a touch time")
	})

	t.Run("takes no finish from a button time at the last lap", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))

		svc.handleFrame(buttonFrame(t, 4, 4, "    2:05.00"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 4, lane.Lap)
		assert.Equal(t, 125000, lane.Time)
		assert.False(t, lane.Finished, "a button time at the last lap is no finish")
		assert.Zero(t, lane.Rank)
	})

	t.Run("finishes the lane with the touch time after a button time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), buttonFrame(t, 4, 4, "    2:05.00"))

		svc.handleFrame(frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.10"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 125100, lane.Time)
		assert.True(t, lane.Finished, "the touch time finishes the lane")
	})

	// The operator demotes a false touch to a split, either with or without the correction mark
	withdrawnFinishFrames := func(t *testing.T, timeType byte) []omega.Frame {
		t.Helper()

		return []omega.Frame{
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:00.00"),
			specFrame(t, frameSpec{
				msgType:   '2',
				msgKind:   'I',
				timeType:  timeType,
				usedLanes: "?<",
				lapCount:  4,
				event:     101,
				heat:      1,
				rank:      1,
				lane:      4,
				lap:       4,
				clock:     "    2:00.00",
			}),
			frame(t, '2', 'A', 4, 101, 1, 1, 5, 4, "    2:03.00"),
			frame(t, '2', 'A', 4, 101, 1, 2, 4, 4, "    2:05.00"),
		}
	}
	withdrawnFinishMarks := []struct {
		name     string
		timeType byte
	}{
		{
			name:     "with the correction mark",
			timeType: 'E',
		},
		{
			name:     "without the correction mark",
			timeType: ' ',
		},
	}

	for _, tt := range withdrawnFinishMarks {
		t.Run("withdraws a finish demoted to a split "+tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestTimingService(t, 50)
			frames := withdrawnFinishFrames(t, tt.timeType)
			handleFrames(svc, frames[:2]...)
			lane := svc.GetLiveHeat().Lanes[4]
			require.NotNil(t, lane)
			require.True(t, lane.Finished, "precondition: the false touch arrives as a finish")
			svc.handleFrame(frames[2])

			svc.handleFrame(frames[3])

			lanes := svc.GetLiveHeat().Lanes
			assert.Equal(t, 4, lanes[4].Lap)
			assert.Equal(t, 120000, lanes[4].Time, "the lap keeps its time as a split")
			assert.False(t, lanes[4].Finished)
			assert.Zero(t, lanes[4].Rank, "a withdrawn finish has no rank among the finishes")
			assert.Equal(t, 1, lanes[5].Rank)
		})

		t.Run("finishes with the real touch after a withdrawn finish "+tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestTimingService(t, 50)
			frames := withdrawnFinishFrames(t, tt.timeType)
			handleFrames(svc, frames[:4]...)

			svc.handleFrame(frames[4])

			lanes := svc.GetLiveHeat().Lanes
			assert.Equal(t, 125000, lanes[4].Time)
			assert.True(t, lanes[4].Finished)
			assert.Equal(t, 2, lanes[4].Rank)
			assert.Equal(t, 1, lanes[5].Rank)
		})
	}

	t.Run("keeps a withdrawn finish without a real touch unfinished", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:00.00"),
			frame(t, '2', 'I', 4, 101, 1, 1, 4, 4, "    2:00.00"),
		)

		svc.handleFrame(frame(t, '1', ' ', 4, 101, 1, 0, 0, 0, ""))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 120000, lane.Time)
		assert.False(t, lane.Finished, "the lane stays unfinished without a finish")
		assert.Zero(t, lane.Rank)
	})

	t.Run("replaces the time of a repeated finish", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"))

		svc.handleFrame(frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:04.50"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, 124500, lane.Time, "the repeated finish replaces the time")
		assert.True(t, lane.Finished)
		assert.Equal(t, 1, lane.Rank)
	})

	// The masks hold lane 1 ("0 "), lane 10 (" !"), both ("0!"), none ("  "), or are invalid ("AA")
	laneMaskFrames := func(t *testing.T) []omega.Frame {
		t.Helper()

		maskFrame := func(msgType, msgKind byte, usedLanes string, lane, lap int, clock string) omega.Frame {
			return specFrame(t, frameSpec{
				msgType:   msgType,
				msgKind:   msgKind,
				timeType:  ' ',
				usedLanes: usedLanes,
				lapCount:  2,
				event:     7,
				heat:      1,
				lane:      lane,
				lap:       lap,
				clock:     clock,
			})
		}
		return []omega.Frame{
			maskFrame('0', ' ', "0!", 0, 0, ""),
			maskFrame('2', 'S', "0 ", 1, 0, "10:00:00.00"),
			maskFrame('2', 'I', "AA", 3, 1, "      30.10"),
			maskFrame('2', 'I', " !", 1, 1, "      31.20"),
			maskFrame('2', 'I', "0 ", 1, 2, "    1:02.00"),
			maskFrame('1', ' ', "  ", 0, 0, ""),
		}
	}

	t.Run("shows the lanes of the lane mask", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)

		svc.handleFrame(frames[0])

		lanes := svc.GetLiveHeat().Lanes
		assert.Contains(t, lanes, 1)
		assert.Contains(t, lanes, 10)
	})

	t.Run("hides a lane without a time outside the lane mask", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)
		handleFrames(svc, frames[:1]...)

		svc.handleFrame(frames[1])

		lanes := svc.GetLiveHeat().Lanes
		require.Contains(t, lanes, 1)
		assert.NotContains(t, lanes, 10, "a lane without a time outside the mask is hidden")
		assert.NotEmpty(t, lanes[1].LastName, "the remaining lanes keep their names")
	})

	t.Run("keeps the earlier lane mask for an invalid one", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)
		handleFrames(svc, frames[:2]...)

		svc.handleFrame(frames[2])

		lanes := svc.GetLiveHeat().Lanes
		assert.NotContains(t, lanes, 10, "an invalid mask keeps the earlier one")
		assert.Contains(t, lanes, 3, "a lane with a time stays outside the mask")
	})

	t.Run("shows a lane again once it is in the lane mask", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)
		handleFrames(svc, frames[:3]...)

		svc.handleFrame(frames[3])

		lanes := svc.GetLiveHeat().Lanes
		assert.Contains(t, lanes, 1, "a lane with a time stays when it leaves the mask")
		require.Contains(t, lanes, 10, "a lane in the mask appears again")
		assert.NotEmpty(t, lanes[10].LastName)
	})

	t.Run("hides a lane again once it leaves the lane mask", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)
		handleFrames(svc, frames[:4]...)

		svc.handleFrame(frames[4])

		assert.NotContains(t, svc.GetLiveHeat().Lanes, 10)
	})

	t.Run("hides no lane with an empty lane mask", func(t *testing.T) {
		t.Parallel()

		svc := newTenLaneTimingService(t)
		frames := laneMaskFrames(t)
		handleFrames(svc, frames[:5]...)

		svc.handleFrame(frames[5])

		assert.Contains(t, svc.GetLiveHeat().Lanes, 10, "an empty mask hides nothing")
	})

	t.Run("removes a lane without a start entry on a restart", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), frame(t, '2', 'I', 4, 101, 1, 1, 7, 1, "      30.10"))

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:05:00.00"))

		assert.NotContains(t, svc.GetLiveHeat().Lanes, 7, "a restart removes the lane again")
	})

	t.Run("clears the times on a restart", func(t *testing.T) {
		t.Parallel()

		competitor := newTestCompetitor(t)
		svc := newTestTimingServiceFor(t, 50, competitor)
		handleFrames(svc, frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"), frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:10.40"))

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:05:00.00"))

		lane := svc.GetLiveHeat().Lanes[4]
		require.NotNil(t, lane)
		assert.Equal(t, competitor.LastName, lane.LastName, "the competitor stays")
		assert.Zero(t, lane.Time)
		assert.Zero(t, lane.Rank)
		assert.False(t, lane.Finished)
	})

	t.Run("keeps the start time and the times for a start repeated for the same heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))
		first := svc.GetLiveHeat().StartTime
		require.NotNil(t, first)
		svc.handleFrame(frame(t, '2', 'I', 4, 101, 1, 1, 4, 1, "      30.10"))

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"))

		state := svc.GetLiveHeat()
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		require.NotNil(t, state.StartTime)
		assert.Equal(t, *first, *state.StartTime, "the repeated start keeps the start time")
		require.NotNil(t, state.Lanes[4])
		assert.Equal(t, 30100, state.Lanes[4].Time, "the repeated start keeps the times")
		assert.Nil(t, svc.heldStart)
	})

	repeatedStartFrames := func(t *testing.T) []omega.Frame {
		t.Helper()

		return []omega.Frame{
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"),
			frame(t, '0', ' ', 4, 101, 2, 0, 0, 0, ""),
			frame(t, '2', 'S', 4, 101, 2, 0, 1, 0, " 9:00:00.00"),
		}
	}

	t.Run("holds back a start repeated under the next heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := repeatedStartFrames(t)
		handleFrames(svc, frames[:3]...)

		svc.handleFrame(frames[3])

		state := svc.GetLiveHeat()
		assert.Equal(t, 2, state.Heat)
		assert.Equal(t, domain.HeatStatusLoad, state.Status, "the repeated start is held back")
		assert.Nil(t, state.StartTime, "the timer does not start")
		assert.NotNil(t, svc.heldStart)
	})

	t.Run("replaces a held back start with a fresh start", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, repeatedStartFrames(t)...)
		require.NotNil(t, svc.heldStart, "precondition: the repeated start is held back")
		heldAt := svc.heldStart.start.at

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 2, 0, 1, 0, " 9:03:10.00"))

		state := svc.GetLiveHeat()
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		require.NotNil(t, state.StartTime)
		assert.NotEqual(t, heldAt, *state.StartTime, "the fresh start sets the start time")
		assert.Nil(t, svc.heldStart)
	})

	t.Run("applies a held back start with the next time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, repeatedStartFrames(t)...)
		require.NotNil(t, svc.heldStart, "precondition: the repeated start is held back")
		heldAt := svc.heldStart.start.at

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 2, 1, 4, 1, "      30.10"))

		state := svc.GetLiveHeat()
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		require.NotNil(t, state.StartTime)
		assert.Equal(t, heldAt, *state.StartTime, "the held back start sets the start time")
		assert.Equal(t, 30100, state.Lanes[4].Time)
		assert.Nil(t, svc.heldStart)
	})

	t.Run("drops a held back start once the heat is official", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '0', ' ', 4, 101, 2, 0, 0, 0, ""),
			frame(t, '2', 'S', 4, 101, 2, 0, 1, 0, " 9:00:00.00"),
		)
		require.NotNil(t, svc.heldStart, "precondition: the repeated start is held back")
		svc.handleFrame(frame(t, '1', ' ', 4, 101, 2, 0, 0, 0, ""))

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 2, 1, 4, 1, "      30.10"))

		state := svc.GetLiveHeat()
		assert.Nil(t, svc.heldStart)
		assert.Equal(t, domain.HeatStatusOfficial, state.Status)
		assert.Nil(t, state.StartTime, "the held back start of an official heat is never applied")
	})

	t.Run("keeps the running heat for the official frame of another heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '0', ' ', 4, 101, 2, 0, 0, 0, ""),
			frame(t, '2', 'S', 4, 101, 2, 0, 1, 0, " 9:00:00.00"),
			frame(t, '1', ' ', 4, 101, 1, 0, 0, 0, ""),
		)

		svc.handleFrame(frame(t, '2', 'I', 4, 101, 2, 1, 4, 1, "      31.25"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 2, state.Heat, "heat 2 is still on display")
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		require.NotNil(t, state.Lanes[4])
		assert.Equal(t, 31250, state.Lanes[4].Time, "the split of heat 2 is kept")
	})

	t.Run("ignores a start for an official heat", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(
			svc,
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"),
			frame(t, '1', ' ', 4, 101, 1, 0, 0, 0, ""),
		)

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:05:00.00"))

		state := svc.GetLiveHeat()
		assert.Equal(t, domain.HeatStatusOfficial, state.Status)
		require.NotNil(t, state.Lanes[4])
		assert.Equal(t, 125000, state.Lanes[4].Time, "the official result stays")
		assert.True(t, state.Lanes[4].Finished)
	})

	// The official heat 1 leaves the recent heats while later heats are loaded
	beyondRecentFrames := func(t *testing.T) []omega.Frame {
		t.Helper()

		frames := []omega.Frame{
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '1', ' ', 4, 101, 1, 0, 0, 0, ""),
		}
		for heat := 2; heat <= recentHeatCount+2; heat++ {
			frames = append(frames, frame(t, '0', ' ', 4, 101, heat, 0, 0, 0, ""))
		}
		return frames
	}

	t.Run("shows a start for an official heat that left the recent heats", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, beyondRecentFrames(t)...)
		for _, state := range svc.recent {
			require.NotEqual(t, 1, state.key.heat, "precondition: the official heat left the recent heats")
		}

		svc.handleFrame(frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:30:00.00"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 1, state.Heat, "a heat that left the recent heats is unknown, so its start shows it")
		assert.Equal(t, domain.HeatStatusStart, state.Status)
	})

	t.Run("runs an official heat that left the recent heats with its next time", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		handleFrames(svc, beyondRecentFrames(t)...)
		for _, state := range svc.recent {
			require.NotEqual(t, 1, state.key.heat, "precondition: the official heat left the recent heats")
		}

		svc.handleFrame(frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.10"))

		state := svc.GetLiveHeat()
		assert.Equal(t, 1, state.Heat, "the time shows its heat")
		assert.Equal(t, domain.HeatStatusStart, state.Status, "a heat that left the recent heats is unknown, so it runs")
		require.NotNil(t, state.Lanes[4])
		assert.Equal(t, 125100, state.Lanes[4].Time)
	})

	// Heat 1 becomes official while heat 2 runs, with lane 4 as the only lane in use
	lateOfficialFrames := func(t *testing.T) []omega.Frame {
		t.Helper()

		return []omega.Frame{
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:00:00.00"),
			frame(t, '2', 'A', 4, 101, 1, 1, 4, 4, "    2:05.00"),
			frame(t, '0', ' ', 4, 101, 2, 0, 0, 0, ""),
			frame(t, '2', 'S', 4, 101, 2, 0, 1, 0, " 9:05:00.00"),
			specFrame(t, frameSpec{
				msgType:   '1',
				msgKind:   ' ',
				timeType:  ' ',
				usedLanes: "\" ",
				lapCount:  4,
				event:     101,
				heat:      1,
			}),
			frame(t, '2', 'S', 4, 101, 1, 0, 1, 0, " 9:10:00.00"),
			specFrame(t, frameSpec{
				msgType:   '2',
				msgKind:   'A',
				timeType:  'E',
				usedLanes: "\" ",
				lapCount:  4,
				event:     101,
				heat:      1,
				rank:      1,
				lane:      4,
				lap:       4,
				clock:     "    2:04.90",
			}),
		}
	}

	t.Run("keeps the running heat for a late official frame", func(t *testing.T) {
		t.Parallel()

		broadcaster := &fakeBroadcaster{}
		svc := NewTimingService(zerolog.Nop(), TimingOptions{
			PoolLength: 50,
		}, broadcaster, fakeDevice{}, nil, newTestTimingService(t, 50).store)
		frames := lateOfficialFrames(t)
		handleFrames(svc, frames[:4]...)
		published := broadcaster.count()

		svc.handleFrame(frames[4])

		state := svc.GetLiveHeat()
		assert.Equal(t, 2, state.Heat, "the running heat stays on display")
		assert.Equal(t, domain.HeatStatusStart, state.Status)
		assert.Equal(t, published, broadcaster.count(), "nothing visible changes")
	})

	t.Run("ignores a late start for a heat that became official late", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := lateOfficialFrames(t)
		handleFrames(svc, frames[:5]...)

		svc.handleFrame(frames[5])

		assert.Equal(t, 2, svc.GetLiveHeat().Heat)
	})

	t.Run("shows a late correction of a heat that became official late", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		frames := lateOfficialFrames(t)
		handleFrames(svc, frames[:6]...)

		svc.handleFrame(frames[6])

		state := svc.GetLiveHeat()
		assert.Equal(t, 1, state.Heat)
		assert.Equal(t, domain.HeatStatusOfficial, state.Status)
		assert.Equal(t, 124900, state.StopTime, "the timer stops with every lane in use finished")
		require.NotNil(t, state.Lanes[4])
		assert.True(t, state.Lanes[4].Finished)
	})
}

func TestTimingService_correctStart(t *testing.T) {
	t.Parallel()

	// The device clock runs three minutes ahead.
	// The hundredths of the frame bound the precision.
	const (
		ahead     = 3 * time.Minute
		tolerance = 50 * time.Millisecond
	)

	within := func(t *testing.T, got, want time.Time) {
		t.Helper()

		assert.WithinDuration(t, want, got, tolerance)
	}
	start := func(t *testing.T, now time.Time, deviceAhead time.Duration) omega.Frame {
		t.Helper()

		return frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, deviceClock(now, deviceAhead))
	}

	t.Run("starts the first start at its arrival", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		now := time.Now()

		correction := svc.correctStart(start(t, now, ahead), now)

		within(t, correction.at, now)
	})

	t.Run("shifts a late start back by its delay", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		svc.correctStart(start(t, first, ahead), first)
		now := time.Now()

		// The start arrives six seconds after it happened
		correction := svc.correctStart(start(t, now.Add(-6*time.Second), ahead), now)

		within(t, correction.at, now.Add(-6*time.Second))
	})

	t.Run("shifts a late start back across midnight", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		// The device clock is already past midnight, the local clock is not
		before := time.Date(2026, 5, 1, 23, 58, 0, 0, time.Local)
		require.Equal(t, before, svc.correctStart(start(t, before, ahead), before).at)
		after := time.Date(2026, 5, 2, 0, 5, 0, 0, time.Local)

		// Both clocks are past midnight, and the start arrives six seconds late
		correction := svc.correctStart(start(t, after.Add(-6*time.Second), ahead), after)

		assert.Equal(t, after.Add(-6*time.Second), correction.at)
		assert.Equal(t, 6*time.Second, correction.delay)
	})

	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	daylightSavingChanges := []struct {
		name string
		// before is a start shortly before the change, two hours later the zone offset differs
		before time.Time
	}{
		{
			name:   "shifts a late start back after the clocks are put forward",
			before: time.Date(2026, 3, 29, 1, 30, 0, 0, berlin),
		},
		{
			name:   "shifts a late start back after the clocks are put back",
			before: time.Date(2026, 10, 25, 1, 30, 0, 0, berlin),
		},
	}

	for _, tt := range daylightSavingChanges {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestTimingService(t, 50)
			svc.clock.zone = fixedZone(tt.before)
			// The device clock does not follow the change and keeps the offset it had before
			zoned := func(at time.Time) omega.Frame {
				return start(t, at.In(svc.clock.zone), ahead)
			}
			for range baselineConfirmations + 1 {
				svc.correctStart(zoned(tt.before), tt.before)
			}
			after := tt.before.Add(2 * time.Hour)
			_, offsetBefore := tt.before.Zone()
			_, offsetAfter := after.Zone()
			require.NotEqual(t, offsetBefore, offsetAfter, "precondition: the zone offset changed")

			correction := svc.correctStart(zoned(after.Add(-6*time.Second)), after)

			assert.Equal(t, 6*time.Second, correction.delay, "the delay is measured against the same baseline")
			assert.Equal(t, after.Add(-6*time.Second), correction.at)
		})
	}

	t.Run("starts at the arrival without a time of day", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		now := time.Now()
		empty := frame(t, '2', 'S', 4, 101, 1, 0, 0, 0, "")

		correction := svc.correctStart(empty, now)

		assert.Equal(t, now, correction.at, "without a time of day, the start is the arrival")
		assert.False(t, correction.valid)
		assert.Nil(t, svc.clock.baseline, "a start without a time of day does not count for the baseline")
	})

	t.Run("replaces the baseline with a smaller offset", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		svc.correctStart(start(t, first, ahead), first)
		now := time.Now()

		// A start with a smaller offset than the baseline starts at its arrival
		correction := svc.correctStart(start(t, now, ahead+2*time.Second), now)

		within(t, correction.at, now)
	})

	t.Run("measures the next start against the replaced baseline", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		svc.correctStart(start(t, first, ahead), first)
		earlier := time.Now()
		svc.correctStart(start(t, earlier, ahead+2*time.Second), earlier)
		now := time.Now()

		correction := svc.correctStart(start(t, now, ahead), now)

		within(t, correction.at, now.Add(-2*time.Second))
	})

	t.Run("keeps the baseline for the whole run of the application", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		svc.correctStart(start(t, first, ahead), first)
		earlier := time.Now()
		svc.correctStart(start(t, earlier, ahead+2*time.Second), earlier)
		now := time.Now()
		svc.correctStart(start(t, now, ahead), now)
		later := now.Add(3 * time.Hour)

		correction := svc.correctStart(start(t, later, ahead), later)

		within(t, correction.at, later.Add(-2*time.Second))
	})

	t.Run("replaces an unconfirmed baseline with a start far below it", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		// The first start arrives a minute late and sets the baseline
		first := time.Now()
		svc.correctStart(start(t, first.Add(-time.Minute), ahead), first)
		now := time.Now()

		// A start without delay replaces it
		correction := svc.correctStart(start(t, now, ahead), now)

		within(t, correction.at, now)
	})

	t.Run("follows the drift of the clocks with a confirmed baseline", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		for range baselineConfirmations + 1 {
			svc.correctStart(start(t, first, ahead), first)
		}
		now := time.Now()
		drifted := ahead + time.Second

		// A drop within the drift replaces the baseline
		correction := svc.correctStart(start(t, now, drifted), now)

		within(t, correction.at, now)
		assert.Equal(t, -drifted, svc.clock.baseline.Round(time.Second))
	})

	t.Run("keeps a confirmed baseline against a start far below it", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		for range baselineConfirmations + 1 {
			svc.correctStart(start(t, first, ahead), first)
		}
		now := time.Now()

		// A start with a time of day far ahead starts at its arrival and keeps the baseline
		correction := svc.correctStart(start(t, now, ahead+time.Minute), now)

		within(t, correction.at, now)
		assert.Equal(t, -ahead, svc.clock.baseline.Round(time.Second))
	})

	t.Run("measures the next start against the kept confirmed baseline", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)
		first := time.Now()
		for range baselineConfirmations + 1 {
			svc.correctStart(start(t, first, ahead), first)
		}
		far := time.Now()
		svc.correctStart(start(t, far, ahead+time.Minute), far)
		now := time.Now()

		correction := svc.correctStart(start(t, now.Add(-6*time.Second), ahead), now)

		within(t, correction.at, now.Add(-6*time.Second))
	})
}

func TestTimingService_recall(t *testing.T) {
	t.Parallel()

	t.Run("keeps the recent heats", func(t *testing.T) {
		t.Parallel()

		svc := newTestTimingService(t, 50)

		for heat := 1; heat <= recentHeatCount+1; heat++ {
			svc.recall(heatKey{
				event: 101,
				heat:  heat,
			})
		}

		require.Len(t, svc.recent, recentHeatCount)
		assert.Equal(t, 2, svc.recent[0].key.heat, "the oldest heat is dropped")
	})
}

func TestTimingService_getPoolLength(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"LSTRACE.TXT":  "event;round;nbHeat;idLen;idStyle;abCat;date;time\n3;0;1;9;0;\"O\";\"26/05/01\";\"10:00\"\n",
		"LSTSTART.TXT": "event;round;heat;lane;relais;idBib\n3;0;0;2;0;1;\n",
	}
	svc := NewTimingService(zerolog.Nop(), TimingOptions{
		PoolLength: 0,
	}, &fakeBroadcaster{}, fakeDevice{}, nil, newTestMeetStore(t, files))

	tests := []struct {
		name     string
		event    int
		lapCount int
	}{
		{
			name:     "falls back for an event without a distance",
			event:    3,
			lapCount: 4,
		},
		{
			name:     "falls back for an unknown event",
			event:    4,
			lapCount: 4,
		},
		{
			name:     "falls back without a lap count",
			event:    3,
			lapCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			poolLength := svc.getPoolLength(tt.event, tt.lapCount)

			assert.Equal(t, 50, poolLength)
		})
	}
}

func TestLocalTimeOfDayMS(t *testing.T) {
	t.Parallel()

	t.Run("returns the milliseconds since midnight", func(t *testing.T) {
		t.Parallel()

		at := time.Date(2026, 5, 1, 10, 15, 42, 300_000_000, time.Local)

		ms := localTimeOfDayMS(at)

		assert.Equal(t, (10*3600+15*60+42)*1000+300, ms)
	})
}

func TestClockOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		localDayMS  int
		omegaTimeMS int
		want        time.Duration
	}{
		{
			name:        "returns a negative offset for a device ahead",
			localDayMS:  36_000_000,
			omegaTimeMS: 36_180_000,
			want:        -180 * time.Second,
		},
		{
			name:        "returns a positive offset for a device behind",
			localDayMS:  43_200_150,
			omegaTimeMS: 43_200_000,
			want:        150 * time.Millisecond,
		},
		{
			name:        "returns the offset with the local clock past midnight",
			localDayMS:  1000,
			omegaTimeMS: 86_399_000,
			want:        2 * time.Second,
		},
		{
			name:        "returns the offset with the device clock past midnight",
			localDayMS:  86_399_000,
			omegaTimeMS: 1000,
			want:        -2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			offset := clockOffset(tt.localDayMS, tt.omegaTimeMS)

			assert.Equal(t, tt.want, offset)
		})
	}
}

func TestHeatState_clearResults(t *testing.T) {
	t.Parallel()

	t.Run("removes the results and keeps the competitors", func(t *testing.T) {
		t.Parallel()

		lastName := apitest.Faker(t).LastName()
		state := &heatState{
			heat: &domain.LiveHeat{
				StopTime: 61230,
				Lanes: map[int]*domain.Lane{
					4: {
						Number:   4,
						LastName: lastName,
						Lap:      2,
						Meter:    100,
						Time:     61230,
						Rank:     1,
						Finished: true,
					},
					7: {
						Number: 7,
						Lap:    1,
						Time:   30120,
					},
				},
			},
			results: map[int]map[int]lapResult{
				4: {
					2: {
						time: 61230,
					},
				},
			},
			listed: map[int]bool{
				4: true,
			},
		}

		state.clearResults()

		assert.Equal(t, map[int]*domain.Lane{
			4: {
				Number:   4,
				LastName: lastName,
			},
		}, state.heat.Lanes, "the competitor stays, a lane without a start entry is removed")
		assert.Empty(t, state.results)
		assert.Zero(t, state.heat.StopTime)
	})
}

func TestHeatState_updateStopTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		usedLanes []int
		lanes     map[int]*domain.Lane
		want      int
	}{
		{
			name:      "stops at the highest finish once every lane in use finished",
			usedLanes: []int{4, 0},
			lanes: map[int]*domain.Lane{
				4: {
					Time:     61000,
					Finished: true,
				},
				10: {
					Time:     62000,
					Finished: true,
				},
			},
			want: 62000,
		},
		{
			name:      "keeps running while a lane in use has not finished",
			usedLanes: []int{4, 5},
			lanes: map[int]*domain.Lane{
				4: {
					Time:     61000,
					Finished: true,
				},
				5: {
					Time: 60000,
				},
			},
			want: 0,
		},
		{
			name:      "keeps running while a lane in use has no time",
			usedLanes: []int{4, 5},
			lanes: map[int]*domain.Lane{
				4: {
					Time:     61000,
					Finished: true,
				},
			},
			want: 0,
		},
		{
			name:      "keeps running without a lane in use",
			usedLanes: []int{},
			lanes: map[int]*domain.Lane{
				4: {
					Time:     61000,
					Finished: true,
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := &heatState{
				heat: &domain.LiveHeat{
					FirstLane: 1,
					StopTime:  1,
					Lanes:     tt.lanes,
				},
				usedLanes: tt.usedLanes,
			}

			state.updateStopTime()

			assert.Equal(t, tt.want, state.heat.StopTime)
		})
	}
}

func TestLaneNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		firstLane int
		frameLane int
		want      int
	}{
		{
			name:      "returns a regular lane",
			firstLane: 1,
			frameLane: 3,
			want:      3,
		},
		{
			name:      "returns lane 10 for the digit 0 of a pool numbered from 1",
			firstLane: 1,
			frameLane: 0,
			want:      10,
		},
		{
			name:      "returns lane 0 for the digit 0 of a pool numbered from 0",
			firstLane: 0,
			frameLane: 0,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			number := laneNumber(tt.firstLane, tt.frameLane)

			assert.Equal(t, tt.want, number)
		})
	}
}

func TestLapRanks(t *testing.T) {
	t.Parallel()

	t.Run("returns the ranks of each lap", func(t *testing.T) {
		t.Parallel()

		results := map[int]map[int]lapResult{
			1: {
				1: {
					time: 30000,
				},
				2: {
					time: 61000,
				},
			},
			2: {
				1: {
					time: 29000,
				},
			},
			3: {
				1: {
					time: 30000,
				},
			},
			4: {
				1: {
					time: 31000,
				},
			},
		}

		ranks := lapRanks(results, 4)

		assert.Equal(t, map[int]map[int]int{
			1: {
				2: 1,
				1: 2,
				3: 2,
				4: 4,
			},
			2: {
				1: 1,
			},
		}, ranks, "equal times share a rank, and the next rank is skipped")
	})

	t.Run("leaves out a withdrawn finish at the last lap", func(t *testing.T) {
		t.Parallel()

		results := map[int]map[int]lapResult{
			1: {
				2: {
					time: 60000,
				},
			},
			2: {
				2: {
					time:   61000,
					finish: true,
				},
			},
		}

		ranks := lapRanks(results, 2)

		assert.Equal(t, map[int]map[int]int{
			2: {
				2: 1,
			},
		}, ranks, "an intermediate time at the last lap takes no part in the ranks")
	})
}

// handleFrames passes the frames to the timing service in their order.
func handleFrames(svc *TimingService, frames ...omega.Frame) {
	for _, fr := range frames {
		svc.handleFrame(fr)
	}
}

// newTenLaneTimingService creates a timing service for the meet files of a pool with ten lanes.
func newTenLaneTimingService(t *testing.T) *TimingService {
	t.Helper()

	return NewTimingService(zerolog.Nop(), TimingOptions{
		PoolLength: 50,
	}, &fakeBroadcaster{}, fakeDevice{}, nil, newTestMeetStore(t, tenLaneFiles))
}

// buttonFrame builds a button frame of button 1 for a lane and lap of heat 1 of event 101 and decodes it.
func buttonFrame(t *testing.T, lane, lap int, clock string) omega.Frame {
	t.Helper()

	return specFrame(t, frameSpec{
		msgType:   '7',
		msgKind:   'B',
		timeType:  '1',
		usedLanes: "?<",
		lapCount:  4,
		event:     101,
		heat:      1,
		lane:      lane,
		lap:       lap,
		clock:     clock,
	})
}

// deviceClock formats the device's time of day for a start frame, ahead of the local clock by ahead.
func deviceClock(now time.Time, ahead time.Duration) string {
	device := now.Add(ahead)
	h, m, sec := device.Clock()
	return fmt.Sprintf("%02d:%02d:%02d.%02d", h, m, sec, device.Nanosecond()/10_000_000)
}

// newTestTimingService creates a timing service with event 101, a 200 m event with two heats,
// and one competitor in lane 4 of both heats.
func newTestTimingService(t *testing.T, poolLength int) *TimingService {
	t.Helper()

	return newTestTimingServiceFor(t, poolLength, newTestCompetitor(t))
}

// newTestTimingServiceFor creates the timing service of newTestTimingService with the given competitor.
func newTestTimingServiceFor(t *testing.T, poolLength int, competitor domain.Competitor) *TimingService {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, "meet")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	files := map[string]string{
		"LSTLONG.TXT": "idLength;Longueur;Mlongueur;Relais\n2;\"  200 m\"; 200; 1\n",
		"LSTRACE.TXT": "event;round;nbHeat;idLen;idStyle;abCat;date;time\n101;0; 2; 2; 0; \"W\"; \"26/05/01\";\"16:00\";\n",
		"LSTCONC.TXT": fmt.Sprintf(
			"id;bib;lastname;firstname;birthyear;abNat;abCat\n46;\"46\";\"%s\";\"%s\";\"%s\";\"%s\";\"W\";\n",
			competitor.LastName,
			competitor.FirstName,
			competitor.BirthYear,
			competitor.AbNat,
		),
		"LSTSTART.TXT": "event;round;heat;lane;relais;idBib\n101; 0; 0; 4; 0; 46;\n101; 0; 1; 4; 0; 46;\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}

	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: root,
	})
	require.NoError(t, err)
	meetStore := store.NewMeetStore(storage)
	require.NoError(t, meetStore.LoadAll(t.Context()))

	return NewTimingService(zerolog.Nop(), TimingOptions{
		PoolLength: poolLength,
	}, &fakeBroadcaster{}, fakeDevice{}, nil, meetStore)
}

// newTestCompetitor returns a competitor with fake personal data.
func newTestCompetitor(t *testing.T) domain.Competitor {
	t.Helper()

	faker := apitest.Faker(t)
	return domain.Competitor{
		LastName:  faker.LastName(),
		FirstName: faker.FirstName(),
		BirthYear: strconv.Itoa(faker.Year()),
		AbNat:     faker.Company(),
	}
}

// frame builds a timing frame as the device transmits it with the lanes 1 to 8 in use and decodes it.
func frame(t *testing.T, msgType, msgKind byte, lapCount, event, heat, rank, lane, lap int, clock string) omega.Frame {
	t.Helper()

	return specFrame(t, frameSpec{
		msgType:   msgType,
		msgKind:   msgKind,
		timeType:  ' ',
		usedLanes: "?<",
		lapCount:  lapCount,
		event:     event,
		heat:      heat,
		rank:      rank,
		lane:      lane,
		lap:       lap,
		clock:     clock,
	})
}

// frameSpec holds the fields of a timing frame built by specFrame, usedLanes as the two bytes of the lane mask.
type frameSpec struct {
	msgType   byte
	msgKind   byte
	timeType  byte
	usedLanes string
	lapCount  int
	event     int
	heat      int
	rank      int
	lane      int
	lap       int
	clock     string
}

// specFrame builds a timing frame from its fields as the device transmits it and decodes it.
func specFrame(t *testing.T, spec frameSpec) omega.Frame {
	t.Helper()

	rankField := "  "
	if spec.rank > 0 {
		rankField = fmt.Sprintf("%2d", spec.rank)
	}
	laneField := " "
	if spec.lane > 0 {
		laneField = fmt.Sprintf("%d", spec.lane%10)
	}

	part1 := fmt.Sprintf(
		"\x01\x02\x08%c%c%c%s%2d%03d%02d  %s\x04",
		spec.msgType,
		spec.msgKind,
		spec.timeType,
		spec.usedLanes,
		spec.lapCount,
		spec.event,
		spec.heat,
		rankField,
	)
	part2 := fmt.Sprintf("\x01\x02\x08\x0a%s%2d\x02%11s \x04", laneField, spec.lap, spec.clock)

	decoded, err := omega.NewDecoder(bytes.NewReader([]byte(part1 + part2))).Next()
	require.NoError(t, err)
	return decoded
}

// textFrame builds a timing frame with the lanes 1 to 8 in use from the text of its fields and decodes it,
// so a field can hold text that is no valid number. Every field has to fill its width in the frame.
func textFrame(t *testing.T, msgType, msgKind byte, lapCount, event, heat, lane, lap, clock string) omega.Frame {
	t.Helper()

	part1 := fmt.Sprintf("\x01\x02\x08%c%c ?<%s%s%s    \x04", msgType, msgKind, lapCount, event, heat)
	part2 := fmt.Sprintf("\x01\x02\x08\x0a%s%s\x02%11s \x04", lane, lap, clock)

	decoded, err := omega.NewDecoder(bytes.NewReader([]byte(part1 + part2))).Next()
	require.NoError(t, err)
	return decoded
}
