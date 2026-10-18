package cli

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
)

func TestNewRecordingCSVCommand(t *testing.T) {
	t.Parallel()

	t.Run("prints the table", func(t *testing.T) {
		t.Parallel()

		path := writeTestRecording(t)

		code, stdout, _ := run(t, "recording", "csv", path)

		require.Equal(t, 0, code)
		rows, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
		require.NoError(t, err)
		require.Len(t, rows, 3, "the header, the session, and the frame")
		assert.Equal(t, "Time", rows[0][0])
		assert.Equal(t, "frame", rows[2][2])
	})

	t.Run("writes the table to a file", func(t *testing.T) {
		t.Parallel()

		path := writeTestRecording(t)
		target := filepath.Join(t.TempDir(), "recording.csv")

		code, stdout, _ := run(t, "recording", "csv", path, target)

		require.Equal(t, 0, code)
		assert.Empty(t, stdout)
		content, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, 3, strings.Count(string(content), "\n"))
	})

	t.Run("removes the file of a failed conversion", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "recording.jsonl")
		require.NoError(t, os.WriteFile(path, []byte("{\n"), 0o644))
		target := filepath.Join(t.TempDir(), "recording.csv")

		code, _, stderr := run(t, "recording", "csv", path, target)

		require.Equal(t, 1, code)
		assert.Contains(t, stderr, path)
		assert.NoFileExists(t, target)
	})

	t.Run("rejects a missing recording", func(t *testing.T) {
		t.Parallel()

		code, _, _ := run(t, "recording", "csv", filepath.Join(t.TempDir(), "missing.jsonl"))

		assert.Equal(t, 1, code)
	})
}

// writeTestRecording writes a recording with one session and one load frame and returns its path.
func writeTestRecording(t *testing.T) string {
	t.Helper()

	at := time.Now()
	path := filepath.Join(t.TempDir(), "recording.jsonl")
	file, err := os.Create(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()

	encoder := json.NewEncoder(file)
	require.NoError(t, encoder.Encode(recording.Entry{
		Type: recording.TypeSession,
		At:   at,
		Session: &recording.Session{
			Format: recording.FormatVersion,
			Source: recording.SourceDevice,
		},
	}))
	require.NoError(t, encoder.Encode(recording.Entry{
		Type: recording.TypeData,
		At:   at.Add(time.Second),
		Data: omega.Encode(omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: 2,
			Event:    1,
			Heat:     1,
		}),
	}))
	return path
}
