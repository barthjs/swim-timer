package recording

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Lines of a recording, as the writer writes them.
const (
	sessionLine = `{"type":"session","at":"2026-10-18T09:30:00+02:00","format":1,"version":"1.0.0","source":"device","port":"","baudRate":9600}`
	meetLine    = `{"type":"meet","at":"2026-10-18T09:30:00.1+02:00","files":[{"name":"LSTSTART","sha256":"00","content":"YQ=="}]}`
	dataLine    = `{"type":"data","at":"2026-10-18T09:31:00.5+02:00","data":"01 02 08"}`
	gapLine     = `{"type":"gap","at":"2026-10-18T09:31:01+02:00","droppedBytes":12}`
	brokenLine  = `{"type":"data","at":"2026-10-18T09:3`
)

func TestReader_Next(t *testing.T) {
	t.Parallel()

	t.Run("returns the entries in order", func(t *testing.T) {
		t.Parallel()

		got, err := readAll(sessionLine, meetLine, dataLine, gapLine)

		require.NoError(t, err)
		require.Len(t, got, 4)
		assert.Equal(t, []EntryType{
			TypeSession,
			TypeMeet,
			TypeData,
			TypeGap,
		}, types(got))
		require.NotNil(t, got[0].Session)
		assert.Equal(t, SourceDevice, got[0].Source)
		assert.Equal(t, 9600, got[0].BaudRate)
		assert.Equal(t, []byte("a"), got[1].Files[0].Content)
		assert.Equal(t, HexBytes{0x01, 0x02, 0x08}, got[2].Data)
		assert.Equal(t, int64(12), got[3].DroppedBytes)
	})

	t.Run("reads a last line without line break", func(t *testing.T) {
		t.Parallel()

		got, err := readEntries(sessionLine + "\n" + dataLine)

		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("skips empty lines", func(t *testing.T) {
		t.Parallel()

		got, err := readAll(sessionLine, "", dataLine, "  ")

		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("skips a broken line before a session", func(t *testing.T) {
		t.Parallel()

		got, err := readAll(sessionLine, dataLine, brokenLine, sessionLine, dataLine)

		require.NoError(t, err)
		assert.Equal(t, []EntryType{
			TypeSession,
			TypeData,
			TypeSession,
			TypeData,
		}, types(got))
	})

	t.Run("skips a broken last line", func(t *testing.T) {
		t.Parallel()

		got, err := readAll(sessionLine, dataLine, brokenLine)

		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	tests := []struct {
		name  string
		lines []string
	}{
		{
			name: "rejects a broken line within a session",
			lines: []string{
				sessionLine,
				brokenLine,
				dataLine,
			},
		},
		{
			name: "rejects two broken lines before a session",
			lines: []string{
				sessionLine,
				brokenLine,
				brokenLine,
				sessionLine,
			},
		},
		{
			name: "rejects a recording that does not start with a session",
			lines: []string{
				dataLine,
			},
		},
		{
			name: "rejects an unsupported format",
			lines: []string{
				strings.Replace(sessionLine, `"format":1`, `"format":2`, 1),
			},
		},
		{
			name: "rejects an unknown line type",
			lines: []string{
				sessionLine,
				strings.Replace(dataLine, `"data",`, `"other",`, 1),
				dataLine,
			},
		},
		{
			name: "rejects a data line without data",
			lines: []string{
				sessionLine,
				`{"type":"data","at":"2026-10-18T09:31:00+02:00"}`,
				dataLine,
			},
		},
		{
			name: "rejects a gap line without dropped bytes",
			lines: []string{
				sessionLine,
				`{"type":"gap","at":"2026-10-18T09:31:00+02:00"}`,
				dataLine,
			},
		},
		{
			name: "rejects a session line without session data",
			lines: []string{
				`{"type":"session","at":"2026-10-18T09:30:00+02:00"}`,
				dataLine,
			},
		},
		{
			name: "rejects a line without time",
			lines: []string{
				sessionLine,
				`{"type":"data","data":"01"}`,
				dataLine,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := readAll(tt.lines...)

			assert.Error(t, err)
		})
	}
}

// readAll reads a recording made of lines.
func readAll(lines ...string) ([]Entry, error) {
	return readEntries(strings.Join(lines, "\n") + "\n")
}

// readEntries reads every entry of a recording until its end or the first error.
func readEntries(content string) ([]Entry, error) {
	reader := NewReader(strings.NewReader(content))
	var entries []Entry
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return entries, err
		}
		entries = append(entries, entry)
	}
}

// types returns the type of every entry.
func types(entries []Entry) []EntryType {
	result := make([]EntryType, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.Type)
	}
	return result
}
