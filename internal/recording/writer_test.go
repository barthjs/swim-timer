package recording

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestFileName(t *testing.T) {
	t.Parallel()

	t.Run("names the file after the day", func(t *testing.T) {
		t.Parallel()

		day := time.Date(2026, 10, 18, 23, 59, 0, 0, time.Local)

		got := FileName(day)

		assert.Equal(t, "recording_2026-10-18.jsonl", got)
	})
}

func TestWriter_Start(t *testing.T) {
	t.Parallel()

	t.Run("starts a session and writes the entries in order", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		session := newTestSession(t)
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: session,
		})
		files := []MeetFile{
			NewMeetFile("LSTSTART", []byte(apitest.Faker(t).Sentence(4))),
		}
		at := time.Now()

		w.Start()
		w.Meet(at, files)
		w.Data(at.Add(time.Second), []byte{0x01, 0x02, 0x08})
		w.Close()

		entries := readRecording(t, dir)
		require.Len(t, entries, 3)
		assert.Equal(t, TypeSession, entries[0].Type)
		require.NotNil(t, entries[0].Session)
		session.Format = FormatVersion
		assert.Equal(t, session, *entries[0].Session)
		assert.Equal(t, TypeMeet, entries[1].Type)
		assert.Equal(t, files, entries[1].Files)
		assert.True(t, at.Equal(entries[1].At))
		assert.Equal(t, TypeData, entries[2].Type)
		assert.Equal(t, HexBytes{0x01, 0x02, 0x08}, entries[2].Data)
		assert.True(t, at.Add(time.Second).Equal(entries[2].At))
	})

	t.Run("appends a new session to the file of the day", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		first := runWriter(t, dir, []byte{0x01})

		runWriter(t, dir, []byte{0x02})

		entries := readRecording(t, dir)
		require.Len(t, entries, 4, "two sessions with their data in one file")
		assert.Equal(t, TypeSession, entries[2].Type)
		assert.Equal(t, HexBytes{0x02}, entries[3].Data)
		assert.Equal(t, first, filepath.Join(dir, FileName(time.Now())))
	})

	t.Run("starts the session on a new line after a broken last line", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := runWriter(t, dir, []byte{0x01})
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
		require.NoError(t, err)
		_, err = file.WriteString(`{"type":"data","at":"2026`)
		require.NoError(t, err)
		require.NoError(t, file.Close())

		runWriter(t, dir, []byte{0x02})

		entries := readRecording(t, dir)
		require.Len(t, entries, 4, "the broken line is skipped")
		assert.Equal(t, TypeSession, entries[2].Type)
		assert.Equal(t, HexBytes{0x02}, entries[3].Data)
	})

	t.Run("keeps accepting entries with an unwritable directory", func(t *testing.T) {
		t.Parallel()

		// A file in place of the directory makes opening the recording fail
		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, nil, 0o644))
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     blocker,
			Session: newTestSession(t),
		})

		w.Start()
		for range queueSize + 1 {
			w.Meet(time.Now(), nil)
		}
		w.Data(time.Now(), []byte{0x01})

		closed := make(chan struct{})
		go func() {
			w.Close()
			close(closed)
		}()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Close did not return")
		}
	})

	t.Run("starts only one session", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})

		w.Start()
		w.Start()
		w.Close()

		entries := readRecording(t, dir)
		assert.Len(t, entries, 1)
	})
}

func TestWriter_Close(t *testing.T) {
	t.Parallel()

	t.Run("writes the entries queued before Start", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		w.Data(time.Now(), []byte{0x01})
		w.Data(time.Now(), []byte{0x02})
		w.Start()

		w.Close()

		entries := readRecording(t, dir)
		require.Len(t, entries, 3, "the session and both data lines")
		assert.Equal(t, HexBytes{0x01}, entries[1].Data)
		assert.Equal(t, HexBytes{0x02}, entries[2].Data)
	})

	t.Run("returns without Start", func(t *testing.T) {
		t.Parallel()

		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     t.TempDir(),
			Session: newTestSession(t),
		})

		w.Close()
		w.Close()
	})
}

func TestWriter_FileName(t *testing.T) {
	t.Parallel()

	t.Run("returns the file being written", func(t *testing.T) {
		t.Parallel()

		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     t.TempDir(),
			Session: newTestSession(t),
		})
		w.Start()
		t.Cleanup(w.Close)
		require.Eventually(t, func() bool { return w.FileName() != "" }, 5*time.Second, time.Millisecond)

		got := w.FileName()

		assert.Equal(t, FileName(time.Now()), got)
	})

	t.Run("returns nothing before the file is open", func(t *testing.T) {
		t.Parallel()

		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     t.TempDir(),
			Session: newTestSession(t),
		})

		got := w.FileName()

		assert.Empty(t, got)
	})
}

func TestWriter_Reader(t *testing.T) {
	t.Parallel()

	t.Run("passes the bytes on and records every read", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		stream := apitest.Faker(t).Sentence(20)

		got, err := io.ReadAll(w.Reader(&chunkReader{
			chunks: [][]byte{
				[]byte(stream[:5]),
				[]byte(stream[5:]),
			},
		}))
		w.Start()
		w.Close()

		require.NoError(t, err)
		assert.Equal(t, stream, string(got))
		entries := readRecording(t, dir)
		require.Len(t, entries, 3, "one data line per read")
		assert.Equal(t, HexBytes(stream[:5]), entries[1].Data)
		assert.Equal(t, HexBytes(stream[5:]), entries[2].Data)
	})
}

func TestWriter_Data(t *testing.T) {
	t.Parallel()

	t.Run("copies the bytes", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		buffer := []byte{0x01}

		w.Data(time.Now(), buffer)
		buffer[0] = 0x02
		w.Start()
		w.Close()

		entries := readRecording(t, dir)
		require.Len(t, entries, 2)
		assert.Equal(t, HexBytes{0x01}, entries[1].Data, "the caller may reuse its buffer")
	})

	t.Run("drops data beyond the queue and reports it as a gap at the end", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		chunk := make([]byte, 1024)

		for range maxQueuedBytes/len(chunk) + 2 {
			w.Data(time.Now(), chunk)
		}
		w.Start()
		w.Close()

		entries := readRecording(t, dir)
		require.Len(t, entries, 1+maxQueuedBytes/len(chunk)+1, "the session, the queued data, and the gap")
		gap := entries[len(entries)-1]
		assert.Equal(t, TypeGap, gap.Type)
		assert.Equal(t, int64(2*len(chunk)), gap.DroppedBytes)
	})

	t.Run("reports dropped data as a gap before the next data", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		chunk := make([]byte, 1024)
		for range maxQueuedBytes/len(chunk) + 1 {
			w.Data(time.Now(), chunk)
		}
		w.Start()
		require.Eventually(t, func() bool { return w.queuedBytes.Load() == 0 }, 5*time.Second, time.Millisecond)

		w.Data(time.Now(), []byte{0x01})
		w.Close()

		entries := readRecording(t, dir)
		require.GreaterOrEqual(t, len(entries), 3)
		last := entries[len(entries)-2:]
		assert.Equal(t, TypeGap, last[0].Type)
		assert.Equal(t, int64(len(chunk)), last[0].DroppedBytes)
		assert.Equal(t, HexBytes{0x01}, last[1].Data)
	})

	t.Run("drops data when the queue holds too many lines", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})

		for range queueSize + 1 {
			w.Data(time.Now(), []byte{0x01})
		}
		w.Start()
		w.Close()

		entries := readRecording(t, dir)
		gap := entries[len(entries)-1]
		assert.Equal(t, TypeGap, gap.Type)
		assert.Equal(t, int64(1), gap.DroppedBytes)
	})
}

func TestWriter_Meet(t *testing.T) {
	t.Parallel()

	t.Run("waits for room instead of dropping the meet files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     dir,
			Session: newTestSession(t),
		})
		for range queueSize {
			w.Data(time.Now(), []byte{0x01})
		}
		queued := make(chan struct{})
		go func() {
			w.Meet(time.Now(), nil)
			close(queued)
		}()

		w.Start()
		<-queued
		w.Close()

		entries := readRecording(t, dir)
		require.Len(t, entries, 1+queueSize+1)
		assert.Equal(t, TypeMeet, entries[len(entries)-1].Type)
	})

	t.Run("returns when the writer stops", func(t *testing.T) {
		t.Parallel()

		w := NewWriter(zerolog.Nop(), WriterOptions{
			Dir:     t.TempDir(),
			Session: newTestSession(t),
		})
		for range queueSize {
			w.Data(time.Now(), []byte{0x01})
		}
		returned := make(chan struct{})
		go func() {
			w.Meet(time.Now(), nil)
			close(returned)
		}()

		w.Close()

		select {
		case <-returned:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Meet did not return")
		}
	})
}

// runWriter records one session with one data line into dir and returns the path of the file.
func runWriter(t *testing.T, dir string, data []byte) string {
	t.Helper()

	w := NewWriter(zerolog.Nop(), WriterOptions{
		Dir:     dir,
		Session: newTestSession(t),
	})
	w.Start()
	w.Data(time.Now(), data)
	w.Close()

	return filepath.Join(dir, FileName(time.Now()))
}

// newTestSession returns the description of an application writing a recording.
func newTestSession(t *testing.T) Session {
	t.Helper()

	faker := apitest.Faker(t)
	return Session{
		Version:  faker.AppVersion(),
		Source:   SourceDevice,
		Port:     "/dev/" + faker.LetterN(6),
		BaudRate: 9600,
	}
}

// readRecording reads every entry of the only recording file in dir.
func readRecording(t *testing.T, dir string) []Entry {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "recording_*.jsonl"))
	require.NoError(t, err)
	require.Len(t, paths, 1)
	content, err := os.ReadFile(paths[0])
	require.NoError(t, err)

	reader := NewReader(strings.NewReader(string(content)))
	var entries []Entry
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return entries
		}
		require.NoError(t, err)
		entries = append(entries, entry)
	}
}

// chunkReader returns one chunk per read.
type chunkReader struct {
	chunks [][]byte
}

// Read returns the next chunk, or io.EOF after the last one.
func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if len(r.chunks[0]) == 0 {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}
