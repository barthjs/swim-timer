package omega

import (
	"fmt"
	"strings"
)

// Kind classifies a frame by its message type and message kind.
type Kind int

const (
	KindUnknown Kind = iota
	KindLoad
	KindOfficial
	KindStart
	KindIntermediate
	KindFinish
	KindButton
)

// String returns the name of the kind in lower case, for example "intermediate".
func (k Kind) String() string {
	switch k {
	case KindUnknown:
		return "unknown"
	case KindLoad:
		return "load"
	case KindOfficial:
		return "official"
	case KindStart:
		return "start"
	case KindIntermediate:
		return "intermediate"
	case KindFinish:
		return "finish"
	case KindButton:
		return "button"
	}
	return "unknown"
}

// Frame is a complete timing frame with a valid header, both parts combined.
type Frame struct {
	raw []byte
}

// FrameFields holds the raw, trimmed text of every frame field, for logs and exports.
type FrameFields struct {
	MsgType   string
	MsgKind   string
	TimeType  string
	UsedLanes string
	LapCount  string
	Event     string
	Heat      string
	Rank      string
	Lane      string
	Lap       string
	Time      string
}

// parseFrame wraps a combined frame, rejecting a wrong length or a byte that deviates from the fixed structure:
// the header bytes and EOT of both parts, the line feed of part 2, the STX before the time field,
// and the fixed spaces. The variable fields are checked by their accessors.
func parseFrame(line []byte) (Frame, bool) {
	if len(line) != frameTotalSize {
		return Frame{}, false
	}

	for offset, want := range frameStructure {
		if line[offset] != want {
			return Frame{}, false
		}
	}

	return Frame{
		raw: line,
	}, true
}

// frameStructure maps every byte offset with a fixed value to that value.
var frameStructure = map[int]byte{
	offsetSOH:         byteSOH,
	offsetSTX:         byteSTX,
	offsetHOME:        byteHOME,
	offsetPadding:     byteSpace,
	offsetPadding + 1: byteSpace,
	offsetPart1EOT:    byteEOT,
	offsetPart2SOH:    byteSOH,
	offsetPart2STX:    byteSTX,
	offsetPart2HOME:   byteHOME,
	offsetPart2LF:     byteLF,
	offsetTimeSTX:     byteSTX,
	offsetTimeSpace:   byteSpace,
	offsetPart2EOT:    byteEOT,
}

// Kind returns the frame's classification. Load and official frames carry a space as message kind,
// running frames distinguish start, intermediate, and finish times, and a button frame carries the time
// of a button pressed at the finish for a lane and lap. Every other combination is KindUnknown.
func (f Frame) Kind() Kind {
	msgType, msgKind := f.raw[offsetMsgType], f.raw[offsetMsgKind]

	switch {
	case msgType == msgTypeLoad && msgKind == msgKindLoad:
		return KindLoad
	case msgType == msgTypeOfficial && msgKind == msgKindLoad:
		return KindOfficial
	case msgType == msgTypeRunning && msgKind == msgKindStart:
		return KindStart
	case msgType == msgTypeRunning && msgKind == msgKindIntermediate:
		return KindIntermediate
	case msgType == msgTypeRunning && msgKind == msgKindFinish:
		return KindFinish
	case msgType == msgTypeButton && msgKind == msgKindButton:
		return KindButton
	default:
		return KindUnknown
	}
}

// UsedLanes returns the lanes the device reports in use (DD) as single-digit lane numbers like Lane,
// lane 10 as 0, and whether the field holds a valid mask. Each byte must carry the fixed bit 5 and no higher bit.
func (f Frame) UsedLanes() ([]int, bool) {
	lanes := make([]int, 0, usedLanesMax)
	for index := range 2 {
		mask := f.raw[offsetUsedLanes+index]
		if mask&^usedLanesBits != usedLanesBase {
			return nil, false
		}
		for position := range usedLanesPerByte {
			if mask&(1<<(usedLanesPerByte-1-position)) == 0 {
				continue
			}
			lane := index*usedLanesPerByte + position + 1
			lanes = append(lanes, lane%usedLanesMax)
		}
	}
	return lanes, true
}

// LapCount returns the total number of laps of the heat (EE), the same in every frame of a heat.
func (f Frame) LapCount() (int, error) {
	return f.requiredNumber("lap count", offsetLapCount, offsetLapCountEnd)
}

// Event returns the event number.
func (f Frame) Event() (int, error) {
	return f.requiredNumber("event", offsetEvent, offsetEventEnd)
}

// Heat returns the heat number, counted from 1.
func (f Frame) Heat() (int, error) {
	return f.requiredNumber("heat", offsetHeat, offsetHeatEnd)
}

// Lane returns the single-digit lane number (J) and whether the field holds one.
// It reports false for an empty or invalid field. The field has room for one digit,
// so a lane 10 arrives as 0, which is a valid lane number.
func (f Frame) Lane() (int, bool) {
	return f.number(offsetLane, offsetLaneEnd)
}

// Lap returns the number of the lap the time was taken at (KK), counted from 1,
// and whether the field holds one. It reports false for an empty or invalid field.
// A start frame carries 0.
func (f Frame) Lap() (int, bool) {
	return f.number(offsetLap, offsetLapEnd)
}

// Rank returns the rank of the swimmer at the lap of this time (HH) and whether the field holds one.
// It reports false for an empty or invalid field. A start frame carries 0.
func (f Frame) Rank() (int, bool) {
	return f.number(offsetRank, offsetRankEnd)
}

// TimeMS returns the transmitted time in milliseconds and whether the field holds a valid time.
// Running frames carry the elapsed time of the heat, a start frame carries the device's time of day at the start.
// An empty field, a reaction time with its sign, and a malformed or out-of-range time report false.
func (f Frame) TimeMS() (int, bool) {
	return parseTimeMS(string(f.raw[offsetTime:offsetTimeEnd]))
}

// Fields returns the raw text of every field, for logs and exports.
func (f Frame) Fields() FrameFields {
	return FrameFields{
		MsgType:   string(f.raw[offsetMsgType]),
		MsgKind:   string(f.raw[offsetMsgKind]),
		TimeType:  string(f.raw[offsetTimeType]),
		UsedLanes: fmt.Sprintf("%02X%02X", f.raw[offsetUsedLanes], f.raw[offsetUsedLanes+1]),
		LapCount:  f.text(offsetLapCount, offsetLapCountEnd),
		Event:     f.text(offsetEvent, offsetEventEnd),
		Heat:      f.text(offsetHeat, offsetHeatEnd),
		Rank:      f.text(offsetRank, offsetRankEnd),
		Lane:      f.text(offsetLane, offsetLaneEnd),
		Lap:       f.text(offsetLap, offsetLapEnd),
		Time:      string(f.raw[offsetTime:offsetTimeEnd]),
	}
}

// Hex returns the frame as space-separated hex bytes, for logs and exports.
func (f Frame) Hex() string {
	var hexes strings.Builder
	for _, b := range f.raw {
		_, _ = fmt.Fprintf(&hexes, "%02X ", b)
	}
	return hexes.String()
}

// requiredNumber parses the field between start and end as a number and returns an error naming the field
// if it is empty or invalid.
func (f Frame) requiredNumber(name string, start, end int) (int, error) {
	value, ok := f.number(start, end)
	if !ok {
		return 0, fmt.Errorf("invalid %s field %q", name, f.raw[start:end])
	}
	return value, nil
}

// number parses the field between start and end as a right-aligned decimal number
// and reports whether the field holds one.
func (f Frame) number(start, end int) (int, bool) {
	return parseNumber(string(f.raw[start:end]))
}

// text returns the field between start and end without surrounding spaces.
func (f Frame) text(start, end int) string {
	return strings.TrimSpace(string(f.raw[start:end]))
}

// parseTimeMS converts a time field in the format Hh:Mm:Ss.dc to milliseconds and reports whether it is valid.
// Leading fields without a value are spaces together with their separator,
// a field after a present one has both digits, and seconds, tenths, and hundredths are always present.
// The hours range from 0 to 23, minutes and seconds from 0 to 59.
func parseTimeMS(t string) (int, bool) {
	if len(t) != timeLength || t[timeOffsetPoint] != '.' {
		return 0, false
	}

	hours, hoursOK := timeComponent(t[timeOffsetHours:timeOffsetHoursE], t[timeOffsetColon1], false)
	if !hoursOK {
		return 0, false
	}
	minutes, minutesOK := timeComponent(t[timeOffsetMin:timeOffsetMinE], t[timeOffsetColon2], hours.present)
	if !minutesOK {
		return 0, false
	}
	seconds, secondsOK := parseNumber(t[timeOffsetSec:timeOffsetSecE])
	if !secondsOK || (minutes.present && t[timeOffsetSec] == ' ') {
		return 0, false
	}
	hundredths, hundredthsOK := parseNumber(t[timeOffsetDC:timeOffsetDCE])
	if !hundredthsOK || t[timeOffsetDC] == ' ' {
		return 0, false
	}

	if hours.value > 23 || minutes.value > 59 || seconds > 59 {
		return 0, false
	}

	return (hours.value*3600+minutes.value*60+seconds)*1000 + hundredths*10, true
}

// timeValue is a leading component of the time field, which may be absent.
type timeValue struct {
	value   int
	present bool
}

// timeComponent parses a leading component of the time field with the separator that follows it.
// An absent component is two spaces followed by a space, a present one is a number followed by a colon.
// After a present component, the next one must be present with both digits.
func timeComponent(field string, separator byte, previousPresent bool) (timeValue, bool) {
	if field == "  " && separator == ' ' && !previousPresent {
		return timeValue{}, true
	}

	value, ok := parseNumber(field)
	if !ok || separator != ':' || (previousPresent && field[0] == ' ') {
		return timeValue{}, false
	}
	return timeValue{
		value:   value,
		present: true,
	}, true
}

// parseNumber parses a right-aligned decimal number, digits with optional leading spaces.
// It reports false for an empty field, signs, inner or trailing spaces, and any other character.
func parseNumber(field string) (int, bool) {
	digits := strings.TrimLeft(field, " ")
	if digits == "" {
		return 0, false
	}

	value := 0
	for _, char := range digits {
		if char < '0' || char > '9' {
			return 0, false
		}
		value = value*10 + int(char-'0')
	}
	return value, true
}
