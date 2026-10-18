package omega

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDiscardHandler(t *testing.T) {
	t.Parallel()

	t.Run("ignores a nil handler", func(t *testing.T) {
		t.Parallel()

		stream := append([]byte("noise"), finishFrame...)

		got := decodeAllFrom(t, NewDecoder(bytes.NewReader(stream), WithDiscardHandler(nil)))

		assert.Len(t, got, 1)
	})
}

func TestDecoder_Next(t *testing.T) {
	// Not parallel, so no other test distorts the allocation count
	t.Run("returns consecutive frames", func(t *testing.T) {
		t.Parallel()

		var stream []byte
		for _, frame := range [][]byte{
			loadFrame,
			startFrame,
			intermediateFrame,
			finishFrame,
			officialFrame,
		} {
			stream = append(stream, frame...)
		}

		got := decodeAll(t, stream)

		require.Len(t, got, 5)
		assert.Equal(t, []Kind{
			KindLoad,
			KindStart,
			KindIntermediate,
			KindFinish,
			KindOfficial,
		}, kinds(got))
	})

	t.Run("returns independent frames", func(t *testing.T) {
		t.Parallel()

		second := withText(intermediateFrame, offsetLane, "7")
		decoder := NewDecoder(bytes.NewReader(append(bytes.Clone(intermediateFrame), second...)))

		a, err := decoder.Next()
		require.NoError(t, err)
		b, err := decoder.Next()
		require.NoError(t, err)

		// A later frame must not overwrite an earlier one
		laneA, _ := a.Lane()
		laneB, _ := b.Lane()
		assert.Equal(t, 2, laneA)
		assert.Equal(t, 7, laneB)
	})

	t.Run("resynchronizes after a disturbance", func(t *testing.T) {
		t.Parallel()

		overlong := append([]byte{0x01, 0x02, 0x08}, bytes.Repeat([]byte("1"), 30)...)
		overlong = append(overlong, 0x04)

		tests := []struct {
			name        string
			disturbance []byte
		}{
			{
				name:        "after a stray block",
				disturbance: []byte{0x01, 0x02, 0x04},
			},
			{
				name:        "after noise without control bytes",
				disturbance: []byte("noise"),
			},
			{
				name:        "after noise ending with a partial header",
				disturbance: []byte{'x', 0x01, 0x02},
			},
			{
				name:        "after part 1 cut off without EOT",
				disturbance: intermediateFrame[:12],
			},
			{
				name:        "after part 2 cut off without EOT",
				disturbance: intermediateFrame[:framePart1Size+10],
			},
			{
				name:        "after part 1 without part 2",
				disturbance: intermediateFrame[:framePart1Size],
			},
			{
				name:        "after part 2 without part 1",
				disturbance: intermediateFrame[framePart1Size:],
			},
			{
				name:        "after part 1 followed by a block of the wrong length",
				disturbance: append(bytes.Clone(intermediateFrame[:framePart1Size]), 0x01, 0x04),
			},
			{
				name:        "after a frame that deviates from the structure",
				disturbance: withByte(intermediateFrame, offsetPart2LF, ' '),
			},
			{
				name:        "after an overlong block with EOT",
				disturbance: overlong,
			},
			{
				name:        "after an overlong block without EOT",
				disturbance: overlong[:len(overlong)-1],
			},
			{
				name:        "after repeated headers",
				disturbance: bytes.Repeat([]byte{0x01, 0x02, 0x08}, 20),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				stream := append(bytes.Clone(tt.disturbance), finishFrame...)
				var discarded []byte
				decoder := NewDecoder(bytes.NewReader(stream), WithDiscardHandler(func(data []byte) {
					discarded = append(discarded, data...)
				}))

				got := decodeAllFrom(t, decoder)

				require.Len(t, got, 1, "only the valid frame after the disturbance")
				assert.Equal(t, finishFrame, got[0].raw)
				assert.Equal(t, tt.disturbance, discarded, "the disturbance is reported as dropped")
			})
		}
	})

	t.Run("reports the pending bytes at the end of the stream", func(t *testing.T) {
		t.Parallel()

		stream := append(bytes.Clone(finishFrame), intermediateFrame[:framePart1Size+5]...)
		var discarded []byte
		decoder := NewDecoder(bytes.NewReader(stream), WithDiscardHandler(func(data []byte) {
			discarded = append(discarded, data...)
		}))

		got := decodeAllFrom(t, decoder)

		require.Len(t, got, 1)
		assert.Equal(t, intermediateFrame[:framePart1Size+5], discarded, "a frame cut off by the end is dropped")
	})

	t.Run("skips a disturbed frame between valid ones", func(t *testing.T) {
		t.Parallel()

		var stream []byte
		stream = append(stream, startFrame...)
		stream = append(stream, intermediateFrame[:framePart1Size+5]...)
		stream = append(stream, 0x7F, 0x00)
		stream = append(stream, finishFrame...)

		got := decodeAll(t, stream)

		assert.Equal(t, []Kind{
			KindStart,
			KindFinish,
		}, kinds(got))
	})

	t.Run("bounds the memory without EOT", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			fill byte
		}{
			{
				name: "for data bytes",
				fill: '1',
			},
			{
				name: "for start of header bytes",
				fill: 0x01,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				const streamSize = 8 << 20
				stream := io.MultiReader(
					io.LimitReader(endlessReader{
						b: tt.fill,
					}, streamSize),
					bytes.NewReader(finishFrame),
				)
				decoder := NewDecoder(stream)

				frame, err := decoder.Next()

				require.NoError(t, err)
				assert.Equal(t, finishFrame, frame.raw, "the frame after the stream is kept")
				assert.LessOrEqual(t, cap(decoder.block), framePart2Size, "the block buffer never grows")
			})
		}
	})

	t.Run("allocates little for a long stream without EOT", func(t *testing.T) {
		data := bytes.Repeat([]byte("1"), 1<<20)

		allocated := testing.AllocsPerRun(5, func() {
			decoder := NewDecoder(bytes.NewReader(data))
			_, _ = decoder.Next()
		})

		assert.Less(t, allocated, float64(10), "allocations do not depend on the length of the stream")
	})
}

// decodeAll decodes a stream until its end and returns every frame.
func decodeAll(t *testing.T, stream []byte) []Frame {
	t.Helper()

	return decodeAllFrom(t, NewDecoder(bytes.NewReader(stream)))
}

// decodeAllFrom reads frames from a decoder until the end of its stream and returns every frame.
func decodeAllFrom(t *testing.T, decoder *Decoder) []Frame {
	t.Helper()

	var frames []Frame
	for {
		frame, err := decoder.Next()
		if errors.Is(err, io.EOF) {
			return frames
		}
		require.NoError(t, err)
		frames = append(frames, frame)
	}
}

// kinds returns the kind of every frame.
func kinds(frames []Frame) []Kind {
	result := make([]Kind, 0, len(frames))
	for _, frame := range frames {
		result = append(result, frame.Kind())
	}
	return result
}

// endlessReader returns the same byte forever.
type endlessReader struct {
	b byte
}

// Read fills p with the byte.
func (r endlessReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
	}
	return len(p), nil
}

// The fuzz tests run their seeds with every `go test`.
// To search for new failing inputs, run one of them explicitly,
// for example: go test -run='^$' -fuzz=FuzzDecoder -fuzztime=30s ./internal/omega
// A failing input is stored under testdata/fuzz and replayed by every later test run.

// FuzzDecoder feeds arbitrary bytes to the decoder, as a disturbed serial line could deliver them.
// The decoder must never panic, must only return frames with a valid structure,
// whose accessors work on any content, and must never hold more than a part 2 of pending bytes.
func FuzzDecoder(f *testing.F) {
	f.Add(finishFrame)
	f.Add(append(bytes.Clone(startFrame), finishFrame...))
	f.Add(finishFrame[:framePart1Size])
	f.Add(finishFrame[framePart1Size:])
	f.Add(finishFrame[:len(finishFrame)-3])
	f.Add([]byte{0x04, 0x04, 0x04})
	f.Add(append([]byte{0x01, 0x02, 0x04}, finishFrame...))
	f.Add(append(bytes.Clone(finishFrame[:framePart1Size]), finishFrame...))
	f.Add(append(bytes.Clone(finishFrame[:10]), finishFrame...))
	f.Add(bytes.Repeat([]byte{0x01}, 64))
	f.Add(bytes.Repeat([]byte{0x01, 0x02, 0x08}, 32))
	f.Add([]byte("random text without any control bytes"))

	f.Fuzz(func(t *testing.T, data []byte) {
		decoder := NewDecoder(bytes.NewReader(data))

		for count := 0; ; count++ {
			frame, err := decoder.Next()
			assert.LessOrEqual(t, cap(decoder.block), framePart2Size)

			if err != nil {
				require.True(t, errors.Is(err, io.EOF), "unexpected error %v", err)
				// Every frame consumes at least its own bytes
				assert.LessOrEqual(t, count*frameTotalSize, len(data))
				return
			}

			_, ok := parseFrame(frame.raw)
			require.True(t, ok, "a returned frame has a valid structure")
			exerciseAccessors(t, frame)
		}
	})
}

// exerciseAccessors calls every accessor of a frame, which must not panic on any content.
func exerciseAccessors(t *testing.T, frame Frame) {
	t.Helper()

	assert.GreaterOrEqual(t, frame.Kind(), KindUnknown)
	assert.LessOrEqual(t, frame.Kind(), KindButton)
	_, _ = frame.LapCount()
	_, _ = frame.Event()
	_, _ = frame.Heat()
	_, _ = frame.Lane()
	_, _ = frame.Lap()
	_, _ = frame.Rank()
	_, _ = frame.TimeMS()
	_ = frame.Fields()
	assert.Len(t, strings.Fields(frame.Hex()), frameTotalSize)
}
