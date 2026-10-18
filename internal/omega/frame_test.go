package omega

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deviceFrame builds a frame byte by byte from the text of both parts between their header and EOT,
// independent of the offset constants the package uses. Part 1 holds ABCDDEEFFFGG¬¬HH (16 bytes),
// part 2 holds JKK[STX]Hh:Mm:Ss.dc¬ (16 bytes).
func deviceFrame(part1, part2 string) []byte {
	frame := []byte{0x01, 0x02, 0x08}
	frame = append(frame, part1...)
	frame = append(frame, 0x04, 0x01, 0x02, 0x08, 0x0A)
	frame = append(frame, part2...)
	return append(frame, 0x04)
}

// Frames as a device sends them for a heat with the lanes 1 to 8 in use, 4 laps, event 3, heat 2.
var (
	loadFrame         = deviceFrame("0  \x3F\x3C 400302    ", "   \x02            ")
	startFrame        = deviceFrame("2S \x3F\x3C 400302   0", "1 0\x0214:17:55.26 ")
	intermediateFrame = deviceFrame("2I \x3F\x3C 400302   3", "2 1\x02      21.89 ")
	finishFrame       = deviceFrame("2A \x3F\x3C 400302   2", "4 4\x02    1:22.07 ")
	officialFrame     = deviceFrame("1  \x3F\x3C 400302    ", "   \x02            ")
	reactionFrame     = deviceFrame("2R+\x3F\x3C 400302  D ", "5  \x02     + 0.71 ")
	buttonFrame       = deviceFrame("7B1\x3F\x3C 400302    ", "3 2\x02      35.12 ")
)

func TestKind_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{
			name: "names unknown frames",
			kind: KindUnknown,
			want: "unknown",
		},
		{
			name: "names load frames",
			kind: KindLoad,
			want: "load",
		},
		{
			name: "names official frames",
			kind: KindOfficial,
			want: "official",
		},
		{
			name: "names start frames",
			kind: KindStart,
			want: "start",
		},
		{
			name: "names intermediate frames",
			kind: KindIntermediate,
			want: "intermediate",
		},
		{
			name: "names finish frames",
			kind: KindFinish,
			want: "finish",
		},
		{
			name: "names button frames",
			kind: KindButton,
			want: "button",
		},
		{
			name: "names a value outside the kinds as unknown",
			kind: Kind(99),
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.kind.String()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFrame(t *testing.T) {
	t.Parallel()

	t.Run("accepts the frames of a device", func(t *testing.T) {
		t.Parallel()

		for _, raw := range [][]byte{
			loadFrame,
			startFrame,
			intermediateFrame,
			finishFrame,
			officialFrame,
			reactionFrame,
			buttonFrame,
		} {
			require.Len(t, raw, frameTotalSize)

			_, ok := parseFrame(raw)

			assert.True(t, ok)
		}
	})

	tests := []struct {
		name string
		raw  []byte
	}{
		{
			name: "rejects a frame that is too short",
			raw:  finishFrame[:len(finishFrame)-1],
		},
		{
			name: "rejects a frame that is too long",
			raw:  append(bytes.Clone(finishFrame), 0x04),
		},
		{
			name: "rejects part 1 without SOH",
			raw:  withByte(finishFrame, 0, 0x00),
		},
		{
			name: "rejects part 1 without STX",
			raw:  withByte(finishFrame, 1, 0x00),
		},
		{
			name: "rejects part 1 without HOME",
			raw:  withByte(finishFrame, 2, 0x00),
		},
		{
			name: "rejects a first padding byte that is no space",
			raw:  withByte(finishFrame, 15, '0'),
		},
		{
			name: "rejects a second padding byte that is no space",
			raw:  withByte(finishFrame, 16, '0'),
		},
		{
			name: "rejects part 1 without EOT",
			raw:  withByte(finishFrame, 19, ' '),
		},
		{
			name: "rejects part 2 without SOH",
			raw:  withByte(finishFrame, 20, 0x00),
		},
		{
			name: "rejects part 2 without STX",
			raw:  withByte(finishFrame, 21, 0x00),
		},
		{
			name: "rejects part 2 without HOME",
			raw:  withByte(finishFrame, 22, 0x00),
		},
		{
			name: "rejects part 2 without LF",
			raw:  withByte(finishFrame, 23, ' '),
		},
		{
			name: "rejects a time field without STX",
			raw:  withByte(finishFrame, 27, ' '),
		},
		{
			name: "rejects a time field followed by no space",
			raw:  withByte(finishFrame, 39, '0'),
		},
		{
			name: "rejects part 2 without EOT",
			raw:  withByte(finishFrame, 40, ' '),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, ok := parseFrame(tt.raw)

			assert.False(t, ok)
		})
	}
}

func TestFrame_Kind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  []byte
		want Kind
	}{
		{
			name: "recognizes a load frame",
			raw:  loadFrame,
			want: KindLoad,
		},
		{
			name: "recognizes an official frame",
			raw:  officialFrame,
			want: KindOfficial,
		},
		{
			name: "recognizes a start frame",
			raw:  startFrame,
			want: KindStart,
		},
		{
			name: "recognizes an intermediate frame",
			raw:  intermediateFrame,
			want: KindIntermediate,
		},
		{
			name: "recognizes a finish frame",
			raw:  finishFrame,
			want: KindFinish,
		},
		{
			name: "recognizes an edited intermediate frame",
			raw:  withByte(intermediateFrame, offsetTimeType, 'E'),
			want: KindIntermediate,
		},
		{
			name: "recognizes a button frame",
			raw:  buttonFrame,
			want: KindButton,
		},
		{
			name: "returns unknown for a reaction time frame",
			raw:  reactionFrame,
			want: KindUnknown,
		},
		{
			name: "returns unknown for an unknown message type",
			raw:  withText(finishFrame, offsetMsgType, "5B1"),
			want: KindUnknown,
		},
		{
			name: "returns unknown for a running type with button kind",
			raw:  withByte(buttonFrame, offsetMsgType, '2'),
			want: KindUnknown,
		},
		{
			name: "returns unknown for a load type with running kind",
			raw:  withByte(loadFrame, offsetMsgKind, 'S'),
			want: KindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frame, ok := parseFrame(tt.raw)
			require.True(t, ok, "a valid frame is accepted")

			got := frame.Kind()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFrame_UsedLanes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		want   []int
		wantOK bool
	}{
		{
			name:   "returns the lanes of both bytes",
			field:  "\x3F\x3C",
			want:   []int{1, 2, 3, 4, 5, 6, 7, 8},
			wantOK: true,
		},
		{
			name:   "returns the first lane of a byte in bit 4",
			field:  "\x27\x20",
			want:   []int{3, 4, 5},
			wantOK: true,
		},
		{
			name:   "returns lane ten as its last digit",
			field:  "\x20\x23",
			want:   []int{9, 0},
			wantOK: true,
		},
		{
			name:   "returns no lane",
			field:  "\x20\x20",
			want:   []int{},
			wantOK: true,
		},
		{
			name:   "rejects a field without the fixed bit",
			field:  "\x1F\x20",
			wantOK: false,
		},
		{
			name:   "rejects a field with a higher bit",
			field:  "\x20\x60",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frame, ok := parseFrame(withText(finishFrame, offsetUsedLanes, tt.field))
			require.True(t, ok)

			got, ok := frame.UsedLanes()

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFrame_LapCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the lap count", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(finishFrame, offsetLapCount, "16123 4"))
		require.True(t, ok)

		got, err := frame.LapCount()

		require.NoError(t, err)
		assert.Equal(t, 16, got)
	})

	t.Run("rejects an invalid lap count", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(loadFrame, offsetLapCount, "xx   -1"))
		require.True(t, ok, "the structure is valid")

		_, err := frame.LapCount()

		assert.ErrorContains(t, err, "lap count")
	})
}

func TestFrame_Event(t *testing.T) {
	t.Parallel()

	t.Run("returns the event", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(finishFrame, offsetLapCount, "16123 4"))
		require.True(t, ok)

		got, err := frame.Event()

		require.NoError(t, err)
		assert.Equal(t, 123, got)
	})

	t.Run("rejects an invalid event", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(loadFrame, offsetLapCount, "xx   -1"))
		require.True(t, ok, "the structure is valid")

		_, err := frame.Event()

		assert.ErrorContains(t, err, "event")
	})
}

func TestFrame_Heat(t *testing.T) {
	t.Parallel()

	t.Run("returns the heat", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(finishFrame, offsetLapCount, "16123 4"))
		require.True(t, ok)

		got, err := frame.Heat()

		require.NoError(t, err)
		assert.Equal(t, 4, got)
	})

	t.Run("rejects an invalid heat", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(withText(loadFrame, offsetLapCount, "xx   -1"))
		require.True(t, ok, "the structure is valid")

		_, err := frame.Heat()

		assert.ErrorContains(t, err, "heat")
	})
}

func TestFrame_Lane(t *testing.T) {
	t.Parallel()

	t.Run("returns the lane of a button frame", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(buttonFrame)
		require.True(t, ok)

		lane, laneOK := frame.Lane()

		require.True(t, laneOK)
		assert.Equal(t, 3, lane)
	})

	tests := []struct {
		name   string
		field  string
		want   int
		wantOK bool
	}{
		{
			name:   "returns the digit",
			field:  "5",
			want:   5,
			wantOK: true,
		},
		{
			name:   "returns zero as the last digit of lane ten",
			field:  "0",
			want:   0,
			wantOK: true,
		},
		{
			name:   "rejects an empty field",
			field:  " ",
			wantOK: false,
		},
		{
			name:   "rejects an invalid field",
			field:  "x",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frame, ok := parseFrame(withText(finishFrame, offsetLane, tt.field))
			require.True(t, ok)

			got, ok := frame.Lane()

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFrame_Lap(t *testing.T) {
	t.Parallel()

	t.Run("returns the lap of a button frame", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(buttonFrame)
		require.True(t, ok)

		lap, lapOK := frame.Lap()

		require.True(t, lapOK)
		assert.Equal(t, 2, lap)
	})

	tests := []struct {
		name   string
		raw    []byte
		want   int
		wantOK bool
	}{
		{
			name:   "returns the lap of a result frame",
			raw:    intermediateFrame,
			want:   1,
			wantOK: true,
		},
		{
			name:   "returns a two-digit lap",
			raw:    withText(intermediateFrame, offsetLap, "12"),
			want:   12,
			wantOK: true,
		},
		{
			name:   "returns zero for a start frame",
			raw:    startFrame,
			wantOK: true,
		},
		{
			name:   "rejects the empty field of a load frame",
			raw:    loadFrame,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frame, ok := parseFrame(tt.raw)
			require.True(t, ok)

			got, ok := frame.Lap()

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects the empty field of a reaction frame", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(reactionFrame)
		require.True(t, ok)

		_, lapOK := frame.Lap()

		assert.False(t, lapOK, "the lap field is empty")
	})
}

func TestFrame_Rank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    []byte
		want   int
		wantOK bool
	}{
		{
			name:   "returns the rank of a result frame",
			raw:    intermediateFrame,
			want:   3,
			wantOK: true,
		},
		{
			name:   "returns a two-digit rank",
			raw:    withText(intermediateFrame, offsetRank, "10"),
			want:   10,
			wantOK: true,
		},
		{
			name:   "returns zero for a start frame",
			raw:    startFrame,
			wantOK: true,
		},
		{
			name:   "rejects the empty field of a load frame",
			raw:    loadFrame,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frame, ok := parseFrame(tt.raw)
			require.True(t, ok)

			got, ok := frame.Rank()

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("rejects the letter of a reaction frame", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(reactionFrame)
		require.True(t, ok)

		_, rankOK := frame.Rank()

		assert.False(t, rankOK, "the rank field holds a letter")
	})
}

// withText returns a copy of frame with text written at offset.
func withText(frame []byte, offset int, text string) []byte {
	changed := bytes.Clone(frame)
	copy(changed[offset:], text)
	return changed
}

func TestFrame_TimeMS(t *testing.T) {
	t.Parallel()

	t.Run("returns the time", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(finishFrame)
		require.True(t, ok)

		got, ok := frame.TimeMS()

		require.True(t, ok)
		assert.Equal(t, 82070, got)
	})

	t.Run("returns the time of a button frame", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(buttonFrame)
		require.True(t, ok)

		ms, timeOK := frame.TimeMS()

		require.True(t, timeOK)
		assert.Equal(t, 35120, ms)
	})

	t.Run("rejects the reaction time", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(reactionFrame)
		require.True(t, ok)

		_, timeOK := frame.TimeMS()

		assert.False(t, timeOK, "the reaction time carries a sign")
	})
}

func TestFrame_Fields(t *testing.T) {
	t.Parallel()

	t.Run("returns the fields", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(finishFrame)
		require.True(t, ok)

		got := frame.Fields()

		assert.Equal(t, FrameFields{
			MsgType:   "2",
			MsgKind:   "A",
			TimeType:  " ",
			UsedLanes: "3F3C",
			LapCount:  "4",
			Event:     "003",
			Heat:      "02",
			Rank:      "2",
			Lane:      "4",
			Lap:       "4",
			Time:      "    1:22.07",
		}, got)
	})

	t.Run("holds the button number in the time type", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(buttonFrame)
		require.True(t, ok)

		got := frame.Fields()

		assert.Equal(t, "1", got.TimeType, "the time type holds the button number")
	})
}

func TestFrame_Hex(t *testing.T) {
	t.Parallel()

	t.Run("returns every byte from SOH to EOT", func(t *testing.T) {
		t.Parallel()

		frame, ok := parseFrame(finishFrame)
		require.True(t, ok)

		hex := frame.Hex()

		assert.True(t, strings.HasPrefix(hex, "01 02 08 32 41 "), "Hex() starts at SOH: %q", hex)
		assert.True(t, strings.HasSuffix(hex, "20 04 "), "Hex() ends at EOT: %q", hex)
		assert.Len(t, strings.Fields(hex), frameTotalSize)
	})
}

func TestParseTimeMS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     string
		want   int
		wantOK bool
	}{
		{
			name:   "parses seconds only",
			in:     "      21.89",
			want:   21890,
			wantOK: true,
		},
		{
			name:   "parses single-digit seconds",
			in:     "       5.07",
			want:   5070,
			wantOK: true,
		},
		{
			name:   "parses minutes",
			in:     "    1:22.07",
			want:   82070,
			wantOK: true,
		},
		{
			name:   "parses two-digit minutes",
			in:     "   12:05.00",
			want:   725000,
			wantOK: true,
		},
		{
			name:   "parses a time of day with a single-digit hour",
			in:     " 7:52:33.60",
			want:   (7*3600+52*60+33)*1000 + 600,
			wantOK: true,
		},
		{
			name:   "parses a time of day",
			in:     "14:17:55.26",
			want:   (14*3600+17*60+55)*1000 + 260,
			wantOK: true,
		},
		{
			name:   "parses the upper bounds",
			in:     "23:59:59.99",
			want:   (23*3600+59*60+59)*1000 + 990,
			wantOK: true,
		},
		{
			name:   "parses midnight",
			in:     " 0:00:00.00",
			want:   0,
			wantOK: true,
		},
		{
			name:   "rejects an empty field",
			in:     "           ",
			wantOK: false,
		},
		{
			name:   "rejects a field of the wrong length",
			in:     "21.89",
			wantOK: false,
		},
		{
			name:   "rejects a reaction time with plus sign",
			in:     "     + 0.24",
			wantOK: false,
		},
		{
			name:   "rejects a reaction time with minus sign",
			in:     "     - 0.11",
			wantOK: false,
		},
		{
			name:   "rejects hours out of range",
			in:     "24:00:00.00",
			wantOK: false,
		},
		{
			name:   "rejects minutes out of range",
			in:     "   60:00.00",
			wantOK: false,
		},
		{
			name:   "rejects seconds out of range",
			in:     "      60.00",
			wantOK: false,
		},
		{
			name:   "rejects a comma instead of a point",
			in:     "      21,89",
			wantOK: false,
		},
		{
			name:   "rejects a point instead of a colon",
			in:     "    1.22.07",
			wantOK: false,
		},
		{
			name:   "rejects minutes without colon",
			in:     "    1 22.07",
			wantOK: false,
		},
		{
			name:   "rejects a colon without minutes",
			in:     "     :22.07",
			wantOK: false,
		},
		{
			name:   "rejects seconds after minutes with a single digit",
			in:     "    1: 2.07",
			wantOK: false,
		},
		{
			name:   "rejects minutes after hours with a single digit",
			in:     " 7: 2:33.60",
			wantOK: false,
		},
		{
			name:   "rejects hours without minutes",
			in:     " 7:  :33.60",
			wantOK: false,
		},
		{
			name:   "rejects missing seconds",
			in:     "    1:  .07",
			wantOK: false,
		},
		{
			name:   "rejects single-digit hundredths",
			in:     "      21. 8",
			wantOK: false,
		},
		{
			name:   "rejects letters",
			in:     "      2a.89",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseTimeMS(tt.in)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// FuzzParseTimeMS checks the time field parser on arbitrary text. A valid time has the length of the field,
// lies within a day, and is a multiple of ten milliseconds.
func FuzzParseTimeMS(f *testing.F) {
	f.Add("      21.89")
	f.Add("    1:22.07")
	f.Add("14:17:55.26")
	f.Add(" 7:52:33.60")
	f.Add("     + 0.65")
	f.Add("           ")
	f.Add("21.89")
	f.Add("99:99:99.99")

	f.Fuzz(func(t *testing.T, in string) {
		got, ok := parseTimeMS(in)
		if !ok {
			assert.Zero(t, got)
			return
		}

		assert.Len(t, in, timeLength)
		assert.GreaterOrEqual(t, got, 0)
		assert.Less(t, got, 24*60*60*1000)
		assert.Zero(t, got%10)
	})
}

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		want   int
		wantOK bool
	}{
		{
			name:   "parses a number right-aligned with a space",
			field:  " 4",
			want:   4,
			wantOK: true,
		},
		{
			name:   "parses leading zeros",
			field:  "003",
			want:   3,
			wantOK: true,
		},
		{
			name:   "parses zero",
			field:  " 0",
			want:   0,
			wantOK: true,
		},
		{
			name:   "parses all digits",
			field:  "123",
			want:   123,
			wantOK: true,
		},
		{
			name:   "rejects an empty field",
			field:  "  ",
			wantOK: false,
		},
		{
			name:   "rejects a plus sign",
			field:  "+4",
			wantOK: false,
		},
		{
			name:   "rejects a minus sign",
			field:  "-4",
			wantOK: false,
		},
		{
			name:   "rejects a trailing space",
			field:  "4 ",
			wantOK: false,
		},
		{
			name:   "rejects an inner space",
			field:  "1 2",
			wantOK: false,
		},
		{
			name:   "rejects a letter",
			field:  "D ",
			wantOK: false,
		},
		{
			name:   "rejects a control byte",
			field:  "\x041",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseNumber(tt.field)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// withByte returns a copy of frame with the byte at offset replaced.
func withByte(frame []byte, offset int, b byte) []byte {
	changed := bytes.Clone(frame)
	changed[offset] = b
	return changed
}
