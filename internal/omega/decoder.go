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

// WithDiscardHandler passes every run of bytes the decoder drops to handle in the order of the stream,
// with a slice that is only valid during the call.
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

// Next blocks until the next complete frame arrives and returns it, or the error of the reader.
func (d *Decoder) Next() (Frame, error) {
	// The device splits every frame into two blocks, each from a header to EOT, see the page omega-transmission-protocol.
	// A header always starts a new block and drops the bytes before it. A block of the part 1 length waits
	// for one of the part 2 length, and any other block or dropped byte discards it,
	// so parts of different frames are never combined.
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

// appendByte adds a byte to the current block, which a completed header starts anew.
func (d *Decoder) appendByte(b byte) {
	// A full block without EOT is dropped, except for trailing bytes that may begin the next header
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

// headerPrefixLength returns the number of trailing bytes of the block
// that match the start of a header, which may begin the next block.
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
