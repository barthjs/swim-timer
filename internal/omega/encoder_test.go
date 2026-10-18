package omega

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	t.Parallel()

	allLanes := []int{1, 2, 3, 4, 5, 6, 7, 8}

	tests := []struct {
		name string
		data FrameData
		want []byte
	}{
		{
			name: "encodes a load frame",
			data: FrameData{
				Kind:      KindLoad,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
			},
			want: loadFrame,
		},
		{
			name: "encodes a start frame",
			data: FrameData{
				Kind:      KindStart,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
				Time:      14*time.Hour + 17*time.Minute + 55260*time.Millisecond,
			},
			want: startFrame,
		},
		{
			name: "encodes a start frame before ten o'clock",
			data: FrameData{
				Kind:      KindStart,
				UsedLanes: []int{3, 4, 5},
				LapCount:  2,
				Event:     12,
				Heat:      1,
				Time:      7*time.Hour + 52*time.Minute + 33600*time.Millisecond,
			},
			want: deviceFrame("2S \x27\x20 201201   0", "1 0\x02 7:52:33.60 "),
		},
		{
			name: "encodes an intermediate frame",
			data: FrameData{
				Kind:      KindIntermediate,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
				Rank:      3,
				Lane:      2,
				Lap:       1,
				Time:      21890 * time.Millisecond,
			},
			want: intermediateFrame,
		},
		{
			name: "encodes a button frame",
			data: FrameData{
				Kind:      KindButton,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
				Lane:      3,
				Lap:       2,
				Time:      35120 * time.Millisecond,
				Button:    1,
			},
			want: buttonFrame,
		},
		{
			name: "encodes a finish frame",
			data: FrameData{
				Kind:      KindFinish,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
				Rank:      2,
				Lane:      4,
				Lap:       4,
				Time:      82070 * time.Millisecond,
			},
			want: finishFrame,
		},
		{
			name: "encodes a finish frame in lane ten after an hour",
			data: FrameData{
				Kind:      KindFinish,
				UsedLanes: []int{9, 10},
				LapCount:  30,
				Event:     123,
				Heat:      12,
				Rank:      10,
				Lane:      10,
				Lap:       30,
				Time:      time.Hour + 2*time.Minute + 3450*time.Millisecond,
			},
			want: deviceFrame("2A \x20\x233012312  10", "030\x02 1:02:03.45 "),
		},
		{
			name: "encodes an intermediate frame without rank",
			data: FrameData{
				Kind:      KindIntermediate,
				UsedLanes: []int{5},
				LapCount:  2,
				Event:     1,
				Heat:      1,
				Lane:      5,
				Lap:       1,
				Time:      5070 * time.Millisecond,
			},
			want: deviceFrame("2I \x21\x20 200101    ", "5 1\x02       5.07 "),
		},
		{
			name: "encodes an official frame",
			data: FrameData{
				Kind:      KindOfficial,
				UsedLanes: allLanes,
				LapCount:  4,
				Event:     3,
				Heat:      2,
			},
			want: officialFrame,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Encode(tt.data)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKindBytes(t *testing.T) {
	t.Parallel()

	t.Run("returns spaces for an unknown kind", func(t *testing.T) {
		t.Parallel()

		msgType, msgKind := kindBytes(KindUnknown)

		assert.Equal(t, byte(byteSpace), msgType)
		assert.Equal(t, byte(byteSpace), msgKind)
	})
}

func TestUsedLanesMask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lanes []int
		want  [2]byte
	}{
		{
			name:  "sets no bit without lanes",
			lanes: nil,
			want: [2]byte{
				0x20,
				0x20,
			},
		},
		{
			name:  "sets bit four for the first lane",
			lanes: []int{1},
			want: [2]byte{
				0x30,
				0x20,
			},
		},
		{
			name:  "sets bit zero for the fifth lane",
			lanes: []int{5},
			want: [2]byte{
				0x21,
				0x20,
			},
		},
		{
			name:  "sets the bits of the lanes three to five",
			lanes: []int{3, 4, 5},
			want: [2]byte{
				0x27,
				0x20,
			},
		},
		{
			name:  "sets the bits of the lanes one to eight",
			lanes: []int{1, 2, 3, 4, 5, 6, 7, 8},
			want: [2]byte{
				0x3F,
				0x3C,
			},
		},
		{
			name:  "sets bit four of the second byte for the sixth lane",
			lanes: []int{6},
			want: [2]byte{
				0x20,
				0x30,
			},
		},
		{
			name:  "sets the same bit for lane ten and its single digit zero",
			lanes: []int{10, 0},
			want: [2]byte{
				0x20,
				0x21,
			},
		},
		{
			name:  "ignores lanes out of range",
			lanes: []int{-1, 11},
			want: [2]byte{
				0x20,
				0x20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := usedLanesMask(tt.lanes)

			assert.Equal(t, tt.want, got)
		})
	}
}

// FuzzEncodeDecode encodes frames with values in the ranges of their fields
// and checks that decoding returns the same values.
func FuzzEncodeDecode(f *testing.F) {
	f.Add(uint8(KindFinish), uint8(4), uint16(12), uint8(2), uint8(1), uint8(5), uint8(4), uint32(125340))
	f.Add(uint8(KindLoad), uint8(0), uint16(0), uint8(0), uint8(0), uint8(0), uint8(0), uint32(0))
	f.Add(uint8(KindStart), uint8(8), uint16(7), uint8(3), uint8(0), uint8(0), uint8(0), uint32(51_475_260))
	f.Add(uint8(KindIntermediate), uint8(99), uint16(999), uint8(99), uint8(99), uint8(9), uint8(99), uint32(86_399_990))

	f.Fuzz(func(t *testing.T, kind, lapCount uint8, event uint16, heat, rank, lane, lap uint8, ms uint32) {
		data := FrameData{
			Kind:      Kind(int(kind)%int(KindButton) + 1),
			UsedLanes: []int{int(lane) % 10},
			LapCount:  int(lapCount) % 100,
			Event:     int(event) % 1000,
			Heat:      int(heat) % 100,
			Rank:      int(rank) % 100,
			Lane:      int(lane) % 10,
			Lap:       int(lap) % 100,
			// Below 24 hours, the largest time the field holds
			Time: msDuration(int(ms % 86_400_000)),
		}

		frame, err := NewDecoder(bytes.NewReader(Encode(data))).Next()
		require.NoError(t, err)

		lapCountGot, err := frame.LapCount()
		require.NoError(t, err)
		eventGot, err := frame.Event()
		require.NoError(t, err)
		heatGot, err := frame.Heat()
		require.NoError(t, err)

		assert.Equal(t, data.Kind, frame.Kind())
		assert.Equal(t, data.LapCount, lapCountGot)
		assert.Equal(t, data.Event, eventGot)
		assert.Equal(t, data.Heat, heatGot)
		assert.Equal(t, usedLanesMask(data.UsedLanes), [2]byte{
			frame.raw[offsetUsedLanes],
			frame.raw[offsetUsedLanes+1],
		})

		lapGot, lapOK := frame.Lap()
		rankGot, rankOK := frame.Rank()
		laneGot, laneOK := frame.Lane()
		timeGot, timeOK := frame.TimeMS()

		switch data.Kind {
		case KindStart:
			assert.Equal(t, 0, lapGot)
			assert.True(t, lapOK)
			assert.Equal(t, 0, rankGot)
			assert.True(t, rankOK)
			assert.Equal(t, 1, laneGot)
			assert.True(t, laneOK)
			assert.Equal(t, int(data.Time.Milliseconds())/10*10, timeGot)
			assert.True(t, timeOK)
		case KindIntermediate, KindFinish, KindButton:
			assert.Equal(t, data.Lap, lapGot)
			assert.True(t, lapOK)
			assert.Equal(t, data.Rank, rankGot)
			assert.Equal(t, data.Rank > 0, rankOK, "a zero rank is sent as spaces")
			assert.Equal(t, data.Lane, laneGot)
			assert.True(t, laneOK)
			// The time field holds hundredths of a second
			assert.Equal(t, int(data.Time.Milliseconds())/10*10, timeGot)
			assert.True(t, timeOK)
		default:
			assert.False(t, lapOK)
			assert.False(t, rankOK)
			assert.False(t, laneOK)
			assert.False(t, timeOK)
		}
	})
}

// msDuration returns a number of milliseconds as a duration.
func msDuration(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
