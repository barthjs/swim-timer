package omega

// Frame structure constants for the Omega transmission protocol.
// A complete frame consists of two parts (41 bytes total):
//	 Part 1 (20 bytes): [SOH][STX][HOME] AB CCC DD EE FF GG ¬¬ HH [EOT]
//   Part 2 (21 bytes): [SOH][STX][HOME][LF] J KK [STX] Hh:Mm:Ss.dc ¬ [EOT]

// Frame sizes
const (
	framePart1Size = 20
	framePart2Size = 21
	frameTotalSize = framePart1Size + framePart2Size
)

// Control bytes
const (
	byteSOH  = 0x01 // Start of Header
	byteSTX  = 0x02 // Start of Text
	byteEOT  = 0x04 // End of Transmission
	byteHOME = 0x08 // Cursor Home
)

// Part 1 byte offsets
const (
	offsetSOH          = 0
	offsetSTX          = 1
	offsetHOME         = 2
	offsetMsgType      = 3
	offsetMsgKind      = 4
	offsetTimeType     = 5
	offsetUsedLanes    = 6
	offsetUsedLanesEnd = 8
	offsetLapCount     = 8
	offsetLapCountEnd  = 10
	offsetEvent        = 10
	offsetEventEnd     = 13
	offsetHeat         = 13
	offsetHeatEnd      = 15
	offsetRank         = 17
	offsetRankEnd      = 19
)

// Part 2 byte offsets (relative to combined frame, i.e., offset + framePart1Size)
const (
	offsetLane    = 24
	offsetLaneEnd = 25
	offsetLap     = 25
	offsetLapEnd  = 27
	offsetTime    = 28
	offsetTimeEnd = 39
)

// Message type values (byte at offsetMsgType)
const (
	msgTypeLoad     = 0x30 // '0' - Load / Ready
	msgTypeOfficial = 0x31 // '1' - Official result
	msgTypeRunning  = 0x32 // '2' - Running (start, intermediate, finish)
)

// Message kind values (byte at offsetMsgKind, used with msgTypeRunning)
const (
	msgKindLoad         = 0x20 // ` ` - used with msgTypeLoad and msgTypeOfficial
	msgKindStart        = 0x53 // 'S' - Race start
	msgKindIntermediate = 0x49 // 'I' - Intermediate time
	msgKindFinish       = 0x41 // 'A' - Finish time
)

// Time string layout offsets (relative to the time substring)
const (
	timeLength       = 11
	timeOffsetHours  = 0
	timeOffsetHoursE = 2
	timeOffsetMin    = 3
	timeOffsetMinE   = 5
	timeOffsetSec    = 6
	timeOffsetSecE   = 8
	timeOffsetCenti  = 9
	timeOffsetCentiE = 11
)
