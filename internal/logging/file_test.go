package logging_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/logging"
)

// fakeClock is a settable clock, so tests can cross midnight without waiting.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Set changes the current fake time.
func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

func TestOpenFile(t *testing.T) {
	t.Parallel()

	t.Run("rotates a file left from an earlier day", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "app.log")
		require.NoError(t, os.WriteFile(path, []byte("before the restart\n"), 0o600))
		yesterday := localTime(1, 12, 0)
		require.NoError(t, os.Chtimes(path, yesterday, yesterday))
		clock := &fakeClock{
			now: localTime(2, 9, 0),
		}
		file := open(t, logging.FileOptions{
			Filename: path,
			Clock:    clock.Now,
		})

		_, err := file.Write([]byte("after the restart\n"))
		require.NoError(t, err)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "after the restart\n", string(content))
		assert.Len(t, backups(t, dir, "app.log"), 1)
	})

	t.Run("stores rotated files as gzip", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		clock := &fakeClock{
			now: localTime(1, 23, 0),
		}
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(dir, "app.log"),
			Compress: true,
			Clock:    clock.Now,
		})

		_, err := file.Write([]byte("yesterday\n"))
		require.NoError(t, err)
		clock.Set(localTime(2, 0, 0))
		_, err = file.Write([]byte("today\n"))
		require.NoError(t, err)

		// lumberjack compresses in the background after the rotation
		assert.Eventually(t, func() bool {
			for _, name := range backups(t, dir, "app.log") {
				if strings.HasSuffix(name, ".gz") {
					return true
				}
			}
			return false
		}, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("keeps rotated files with a negative maximum age", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		clock := &fakeClock{
			now: localTime(1, 23, 0),
		}
		file := open(t, logging.FileOptions{
			Filename:   filepath.Join(dir, "app.log"),
			MaxAgeDays: -1,
			Clock:      clock.Now,
		})

		_, err := file.Write([]byte("kept\n"))
		require.NoError(t, err)
		clock.Set(localTime(2, 0, 1))
		_, err = file.Write([]byte("next day\n"))
		require.NoError(t, err)

		assert.Len(t, backups(t, dir, "app.log"), 1)
	})

	t.Run("rejects a missing filename", func(t *testing.T) {
		t.Parallel()

		_, err := logging.OpenFile(logging.FileOptions{})

		require.ErrorIs(t, err, logging.ErrNoFilename)
	})
}

func TestFile_Write(t *testing.T) {
	t.Parallel()

	t.Run("keeps the file within a day", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		clock := &fakeClock{
			now: localTime(1, 8, 0),
		}
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(dir, "app.log"),
			Clock:    clock.Now,
		})

		_, err := file.Write([]byte("morning\n"))
		require.NoError(t, err)
		clock.Set(localTime(1, 23, 59))
		_, err = file.Write([]byte("evening\n"))
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dir, "app.log"))
		require.NoError(t, err)
		assert.Equal(t, "morning\nevening\n", string(content))
		assert.Empty(t, backups(t, dir, "app.log"))
	})

	t.Run("rotates after midnight", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		clock := &fakeClock{
			now: localTime(1, 23, 0),
		}
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(dir, "app.log"),
			Clock:    clock.Now,
		})

		_, err := file.Write([]byte("yesterday\n"))
		require.NoError(t, err)
		clock.Set(localTime(2, 0, 1))
		_, err = file.Write([]byte("today\n"))
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dir, "app.log"))
		require.NoError(t, err)
		assert.Equal(t, "today\n", string(content))

		rotated := backups(t, dir, "app.log")
		require.Len(t, rotated, 1)
		old, err := os.ReadFile(filepath.Join(dir, rotated[0]))
		require.NoError(t, err)
		assert.Equal(t, "yesterday\n", string(old))
	})

	t.Run("falls back when the file cannot be written", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))
		var fallback, errorOutput bytes.Buffer
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(blocker, "app.log"),
			Fallback: &fallback,
			Errors:   &errorOutput,
		})

		_, err := file.Write([]byte("first\n"))
		require.NoError(t, err)
		_, err = file.Write([]byte("second\n"))
		require.NoError(t, err)

		assert.Equal(t, "first\nsecond\n", fallback.String(), "no line is lost")
		assert.Equal(t, 1, strings.Count(errorOutput.String(), "\n"), "the failure is reported once")
	})

	t.Run("falls back when the daily rotation fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))
		clock := &fakeClock{
			now: localTime(1, 23, 0),
		}
		var fallback, errorOutput bytes.Buffer
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(blocker, "app.log"),
			Clock:    clock.Now,
			Fallback: &fallback,
			Errors:   &errorOutput,
		})
		clock.Set(localTime(2, 0, 0))

		_, err := file.Write([]byte("kept\n"))

		require.NoError(t, err)
		assert.Equal(t, "kept\n", fallback.String())
		assert.Contains(t, errorOutput.String(), "rotating")
	})

	t.Run("keeps the file and retries a failed rotation", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("a read-only directory only blocks the rotation for an unprivileged user on Unix")
		}

		dir := filepath.Join(t.TempDir(), "logs")
		require.NoError(t, os.Mkdir(dir, 0o700))
		clock := &fakeClock{
			now: localTime(1, 23, 0),
		}
		var fallback, errorOutput bytes.Buffer
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(dir, "app.log"),
			Clock:    clock.Now,
			Fallback: &fallback,
			Errors:   &errorOutput,
		})
		_, err := file.Write([]byte("yesterday\n"))
		require.NoError(t, err)

		// A read-only directory keeps the active file writable, but the rotation cannot rename it
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		clock.Set(localTime(2, 0, 0))
		_, err = file.Write([]byte("today\n"))
		require.NoError(t, err)
		_, err = file.Write([]byte("still today\n"))
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dir, "app.log"))
		require.NoError(t, err)
		assert.Equal(t, "yesterday\ntoday\nstill today\n", string(content), "the current file takes the lines")
		assert.Empty(t, fallback.String())
		assert.Equal(t, 1, strings.Count(errorOutput.String(), "\n"), "the failed rotation is reported once")

		require.NoError(t, os.Chmod(dir, 0o700))
		clock.Set(localTime(2, 0, 0).Add(30 * time.Second))
		_, err = file.Write([]byte("before the retry\n"))
		require.NoError(t, err)
		assert.Empty(t, backups(t, dir, "app.log"), "the rotation waits for the retry interval")

		clock.Set(localTime(2, 0, 1))
		_, err = file.Write([]byte("after the retry\n"))
		require.NoError(t, err)
		assert.Len(t, backups(t, dir, "app.log"), 1)

		content, err = os.ReadFile(filepath.Join(dir, "app.log"))
		require.NoError(t, err)
		assert.Equal(t, "after the retry\n", string(content))
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		file := open(t, logging.FileOptions{
			Filename: filepath.Join(dir, "app.log"),
		})

		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				_, _ = file.Write([]byte("line\n"))
			})
		}
		wg.Wait()

		content, err := os.ReadFile(filepath.Join(dir, "app.log"))
		require.NoError(t, err)
		assert.Equal(t, 20, strings.Count(string(content), "line\n"))
	})
}

// localTime returns a local time on the given day of a fixed month.
func localTime(day, hour, minute int) time.Time {
	return time.Date(2026, time.May, day, hour, minute, 0, 0, time.Local)
}

// open opens a log file that is closed when the test ends.
func open(t *testing.T, options logging.FileOptions) *logging.File {
	t.Helper()

	file, err := logging.OpenFile(options)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	return file
}

// backups lists the rotated files next to the active log file.
func backups(t *testing.T, dir, active string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var names []string
	for _, entry := range entries {
		if entry.Name() != active {
			names = append(names, entry.Name())
		}
	}
	return names
}
