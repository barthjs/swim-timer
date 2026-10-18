package omega

import (
	"fmt"
	"time"
)

// FrameData holds the values Encode writes into a frame.
type FrameData struct {
	Kind Kind
	// UsedLanes are the lanes in use, from 1 to 10, with 0 for lane 10. Other numbers are ignored.
	UsedLanes []int
	LapCount  int
	Event     int
	Heat      int
	// Rank, Lane, and Lap are only written into intermediate, finish, and button frames.
	// A zero rank is sent as spaces, a lane as its last digit.
	Rank int
	Lane int
	Lap  int
	// Time is the elapsed time of the heat in intermediate, finish, and button frames,
	// and the device's time of day in a start frame.
	Time time.Duration
	// Button is the number of the button pressed in a button frame, sent as time type.
	Button int
}

// Encode builds both parts of a timing frame in the layout a device sends,
// as the simulator and tests send it in place of a device.
// Numbers are right-aligned with spaces, the event and heat with leading zeros.
// Load and official frames leave the lane, lap, and time fields of part 2 empty,
// a start frame carries lane 1, lap 0, and rank 0 with the time of day.
func Encode(d FrameData) []byte {
	msgType, msgKind := kindBytes(d.Kind)
	usedLanes := usedLanesMask(d.UsedLanes)

	timeType := byte(byteSpace)
	rank := "  "
	lane := " "
	lap := "  "
	clock := "           "
	switch d.Kind {
	case KindStart:
		rank = " 0"
		lane = "1"
		lap = " 0"
		clock = formatTimeOfDay(d.Time)
	case KindIntermediate, KindFinish, KindButton:
		if d.Rank > 0 {
			rank = fmt.Sprintf("%2d", d.Rank)
		}
		lane = fmt.Sprintf("%d", d.Lane%10)
		lap = fmt.Sprintf("%2d", d.Lap)
		clock = formatElapsed(d.Time)
		if d.Kind == KindButton {
			timeType = byte('0' + d.Button%10)
		}
	case KindLoad, KindOfficial, KindUnknown:
		// These frames leave lane, lap, and time empty
	}

	frame := make([]byte, 0, frameTotalSize)

	// Part 1: [SOH][STX][HOME]ABCDDEEFFFGG¬¬HH[EOT]
	frame = append(frame, frameHeader...)
	frame = append(frame, msgType, msgKind, timeType)
	frame = append(frame, usedLanes[0], usedLanes[1])
	frame = append(frame, fmt.Sprintf("%2d%03d%02d  %s", d.LapCount, d.Event, d.Heat, rank)...)
	frame = append(frame, byteEOT)

	// Part 2: [SOH][STX][HOME][LF]JKK[STX]Hh:Mm:Ss.dc¬[EOT]
	frame = append(frame, frameHeader...)
	frame = append(frame, byteLF)
	frame = append(frame, lane+lap...)
	frame = append(frame, byteSTX)
	frame = append(frame, clock...)
	frame = append(frame, byteSpace, byteEOT)

	return frame
}

// kindBytes returns the message type and message kind bytes for a Kind.
func kindBytes(kind Kind) (byte, byte) {
	switch kind {
	case KindLoad:
		return msgTypeLoad, msgKindLoad
	case KindOfficial:
		return msgTypeOfficial, msgKindLoad
	case KindStart:
		return msgTypeRunning, msgKindStart
	case KindIntermediate:
		return msgTypeRunning, msgKindIntermediate
	case KindFinish:
		return msgTypeRunning, msgKindFinish
	case KindButton:
		return msgTypeButton, msgKindButton
	default:
		return byteSpace, byteSpace
	}
}

// usedLanesMask returns the two bytes of the used lanes field (DD) for the given lanes.
// The first byte holds the lanes 1 to 5, the second the lanes 6 to 10, each lane at bit 4 down to bit 0.
func usedLanesMask(lanes []int) [2]byte {
	mask := [2]byte{
		usedLanesBase,
		usedLanesBase,
	}
	for _, lane := range lanes {
		if lane == 0 {
			lane = usedLanesMax
		}
		if lane < 1 || lane > usedLanesMax {
			continue
		}

		index := (lane - 1) / usedLanesPerByte
		position := (lane - 1) % usedLanesPerByte
		mask[index] |= 1 << (usedLanesPerByte - 1 - position)
	}
	return mask
}

// formatTimeOfDay formats a time of day as Hh:Mm:Ss.dc with the hours right-aligned with a space.
func formatTimeOfDay(t time.Duration) string {
	hours, minutes, seconds, hundredths := timeParts(t)
	return fmt.Sprintf("%2d:%02d:%02d.%02d", hours, minutes, seconds, hundredths)
}

// formatElapsed formats an elapsed time as Hh:Mm:Ss.dc, with leading fields without a value left as spaces
// and the first field with a value right-aligned with a space.
func formatElapsed(t time.Duration) string {
	hours, minutes, seconds, hundredths := timeParts(t)
	switch {
	case hours > 0:
		return fmt.Sprintf("%2d:%02d:%02d.%02d", hours, minutes, seconds, hundredths)
	case minutes > 0:
		return fmt.Sprintf("   %2d:%02d.%02d", minutes, seconds, hundredths)
	default:
		return fmt.Sprintf("      %2d.%02d", seconds, hundredths)
	}
}

// timeParts splits a duration into hours, minutes, seconds, and hundredths of a second.
func timeParts(t time.Duration) (int, int, int, int) {
	hundredthsTotal := int(t.Milliseconds()) / 10
	hundredths := hundredthsTotal % 100
	secondsTotal := hundredthsTotal / 100
	return secondsTotal / 3600, secondsTotal / 60 % 60, secondsTotal % 60, hundredths
}
