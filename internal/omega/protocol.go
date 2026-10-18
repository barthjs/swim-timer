// Package omega implements the Omega (Swiss Timing) transmission protocol:
// decoding the timing frames a device sends over a serial line, and encoding them for simulation.
package omega

import "github.com/barthjs/swim-timer/internal/serial"

// SerialMode returns the line setting of the device's serial output at the given baud rate.
// The device transmits 8 data bits without parity, one stop bit, and no handshake,
// at a baud rate between 9600 and 115200 set on the device.
func SerialMode(baudRate int) serial.Mode {
	return serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.ParityNone,
		StopBits: serial.StopBitsOne,
	}
}

// Frame structure constants for the Omega transmission protocol (see docs/omega-transmission-protocol.md).
// A complete frame consists of two parts (41 bytes total), each terminated by EOT:
//
//	Part 1 (20 bytes): [SOH][STX][HOME] A B C DD EE FFF GG ¬¬ HH [EOT]
//	Part 2 (21 bytes): [SOH][STX][HOME][LF] J KK [STX] Hh:Mm:Ss.dc ¬ [EOT]
//
// The protocol description reuses words like "lap" and "time" for different fields,
// so every offset below names exactly one field and what it holds.

// Frame sizes.
const (
	framePart1Size = 20
	framePart2Size = 21
	frameTotalSize = framePart1Size + framePart2Size
)

// FrameSize is the number of bytes of a complete frame, both parts together.
const FrameSize = frameTotalSize

// Control bytes.
const (
	byteSOH  = 0x01 // Start of Header
	byteSTX  = 0x02 // Start of Text
	byteEOT  = 0x04 // End of Transmission
	byteHOME = 0x08 // Cursor Home
	byteLF   = 0x0A // Line Feed, the fourth header byte of part 2
)

// frameHeader starts both parts of a frame, so a decoder resynchronizes at it.
var frameHeader = []byte{byteSOH, byteSTX, byteHOME}

// Part 1 byte offsets.
const (
	offsetSOH          = 0
	offsetSTX          = 1
	offsetHOME         = 2
	offsetMsgType      = 3 // A: message type (load, official, running)
	offsetMsgKind      = 4 // B: kind of running message (start, split, finish, reaction)
	offsetTimeType     = 5 // C: how the time was taken (touchpad, edited, platform, button)
	offsetUsedLanes    = 6 // DD: bitmask of the lanes in use, two bytes
	offsetUsedLanesEnd = 8
	offsetLapCount     = 8 // EE: total number of laps of the heat, not the lap just swum
	offsetLapCountEnd  = 10
	offsetEvent        = 10 // FFF: event number, as in the start list
	offsetEventEnd     = 13
	offsetHeat         = 13 // GG: heat number, counted from 1
	offsetHeatEnd      = 15
	offsetPadding      = 15 // two fixed spaces
	offsetPaddingEnd   = 17
	offsetRank         = 17 // HH: rank of the swimmer at the lap of this time
	offsetRankEnd      = 19
	offsetPart1EOT     = 19
)

// Part 2 byte offsets (relative to combined frame, i.e., offset + framePart1Size).
const (
	offsetPart2SOH  = 20
	offsetPart2STX  = 21
	offsetPart2HOME = 22
	offsetPart2LF   = 23
	offsetLane      = 24 // J: lane of the swimmer, a single digit
	offsetLaneEnd   = 25
	offsetLap       = 25 // KK: number of the lap the time was taken at, counted from 1
	offsetLapEnd    = 27
	offsetTimeSTX   = 27
	offsetTime      = 28 // Hh:Mm:Ss.dc: elapsed time of the heat, or the device's time of day in a start frame
	offsetTimeEnd   = 39
	offsetTimeSpace = 39 // one fixed space
	offsetPart2EOT  = 40
)

// Message type values (byte at offsetMsgType).
const (
	msgTypeLoad     = 0x30 // '0' - Load / Ready
	msgTypeOfficial = 0x31 // '1' - Official result
	msgTypeRunning  = 0x32 // '2' - Running (start, intermediate, finish)
	msgTypeButton   = 0x37 // '7' - Button press at the finish
)

// Message kind values (byte at offsetMsgKind). Reaction times ('R'), relay takeovers ('D'), and further message types
// are transmitted as well, but carry nothing the live heat uses, so they are not decoded.
const (
	msgKindLoad         = 0x20 // ` ` - used with msgTypeLoad and msgTypeOfficial
	msgKindStart        = 0x53 // 'S' - Start of the heat
	msgKindIntermediate = 0x49 // 'I' - Intermediate time
	msgKindFinish       = 0x41 // 'A' - Finish time
	msgKindButton       = 0x42 // 'B' - Button press, used with msgTypeButton and the button number as time type
)

// Time string layout offsets (relative to the time substring).
const (
	timeLength       = 11
	timeOffsetHours  = 0
	timeOffsetHoursE = 2
	timeOffsetColon1 = 2
	timeOffsetMin    = 3
	timeOffsetMinE   = 5
	timeOffsetColon2 = 5
	timeOffsetSec    = 6
	timeOffsetSecE   = 8
	timeOffsetPoint  = 8
	timeOffsetDC     = 9 // dc: tenths (d) and hundredths (c) of a second
	timeOffsetDCE    = 11
)

// Used lanes bitmask (DD). Each byte carries the fixed bit 5 and one bit per lane in its five low bits,
// the first lane of the byte in bit 4 and the fifth in bit 0.
const (
	usedLanesBase    = 0x20
	usedLanesBits    = 0x1F
	usedLanesPerByte = 5
	usedLanesMax     = 10
)

// byteSpace fills empty fields and separators.
const byteSpace = 0x20
