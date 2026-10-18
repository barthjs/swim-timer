package omega

import (
	"bufio"
	"bytes"
	"io"
)

// decoderBufferSize is the read buffer of the decoder, a few frames long.
const decoderBufferSize = 4 * frameTotalSize

// Decoder reads timing frames from a byte stream.
type Decoder struct {
	reader *bufio.Reader
	// block collects the current transmission block, never more than a part 2 long.
	block []byte
	// part1 is the last complete part 1, waiting for its part 2.
	part1 []byte
	// onDiscard receives the bytes the decoder drops, nil if nobody needs them.
	onDiscard func(data []byte)
}

// DecoderOption configures a Decoder.
type DecoderOption func(*Decoder)

// WithDiscardHandler passes every run of bytes the decoder drops to handle, in the order of the stream,
// for example to analyze a disturbed line. The slice is only valid during the call. A nil function is ignored.
func WithDiscardHandler(handle func(data []byte)) DecoderOption {
	return func(d *Decoder) {
		if handle != nil {
			d.onDiscard = handle
		}
	}
}

// NewDecoder creates a Decoder that reads from r.
func NewDecoder(r io.Reader, opts ...DecoderOption) *Decoder {
	d := &Decoder{
		reader: bufio.NewReaderSize(r, decoderBufferSize),
		block:  make([]byte, 0, framePart2Size),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Next blocks until the next complete frame arrives and returns it. The device splits every frame
// into two transmission blocks, each starting with SOH, STX, HOME and ending with EOT.
// A header always starts a new block, so bytes before it, such as noise or a block cut off without its EOT,
// are dropped and the frame that follows is kept. A block never grows beyond the length of a part 2.
// A block of the part 1 length is kept until a block of the part 2 length follows it.
// Any other block and any dropped byte discard the pending part 1, so parts of different frames are never combined.
// Combined blocks that deviate from the frame structure are skipped. It returns an error only if reading fails,
// and drops the pending bytes then.
func (d *Decoder) Next() (Frame, error) {
	for {
		b, err := d.reader.ReadByte()
		if err != nil {
			// The stream ends, so the pending bytes can no longer become a frame
			d.discardPart1()
			d.discard(d.block)
			d.block = d.block[:0]
			return Frame{}, err
		}

		d.appendByte(b)
		if b != byteEOT {
			continue
		}

		frame, ok := d.completeBlock()
		if ok {
			return frame, nil
		}
	}
}

// appendByte adds a byte to the current block. A header that completes after other bytes starts a new block
// and drops what came before it. A full block without EOT is dropped as well, except for trailing bytes
// that may begin the next header.
func (d *Decoder) appendByte(b byte) {
	if len(d.block) == framePart2Size {
		kept := headerPrefixLength(d.block)
		d.discardPart1()
		d.discard(d.block[:len(d.block)-kept])
		d.block = append(d.block[:0], frameHeader[:kept]...)
	}

	d.block = append(d.block, b)

	headerStart := len(d.block) - len(frameHeader)
	if headerStart > 0 && bytes.Equal(d.block[headerStart:], frameHeader) {
		d.discardPart1()
		d.discard(d.block[:headerStart])
		d.block = append(d.block[:0], frameHeader...)
	}
}

// headerPrefixLength returns the number of trailing bytes of the block that match the start of a header,
// which may begin the next block.
func headerPrefixLength(block []byte) int {
	for size := len(frameHeader) - 1; size > 0; size-- {
		if bytes.HasSuffix(block, frameHeader[:size]) {
			return size
		}
	}
	return 0
}

// completeBlock handles a block that has just received its EOT and returns a frame once both parts are complete.
// The block is reset in every case.
func (d *Decoder) completeBlock() (Frame, bool) {
	block := d.block
	d.block = d.block[:0]

	if !bytes.HasPrefix(block, frameHeader) {
		d.discardPart1()
		d.discard(block)
		return Frame{}, false
	}

	if len(block) == framePart1Size {
		d.discardPart1()
		d.part1 = bytes.Clone(block)
		return Frame{}, false
	}

	if len(block) != framePart2Size || d.part1 == nil {
		d.discardPart1()
		d.discard(block)
		return Frame{}, false
	}

	combined := make([]byte, 0, frameTotalSize)
	combined = append(combined, d.part1...)
	combined = append(combined, block...)
	d.part1 = nil

	frame, ok := parseFrame(combined)
	if !ok {
		d.discard(combined)
	}
	return frame, ok
}

// discardPart1 drops a pending part 1.
func (d *Decoder) discardPart1() {
	d.discard(d.part1)
	d.part1 = nil
}

// discard passes dropped bytes to the discard handler, if there are any and a handler is set.
func (d *Decoder) discard(data []byte) {
	if len(data) == 0 || d.onDiscard == nil {
		return
	}
	d.onDiscard(data)
}
