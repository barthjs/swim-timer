package omega

import "bytes"

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

// withText returns a copy of frame with text written at offset.
func withText(frame []byte, offset int, text string) []byte {
	changed := bytes.Clone(frame)
	copy(changed[offset:], text)
	return changed
}

// withByte returns a copy of frame with the byte at offset replaced.
func withByte(frame []byte, offset int, b byte) []byte {
	changed := bytes.Clone(frame)
	changed[offset] = b
	return changed
}
