package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/logging"
)

func TestState_Setup(t *testing.T) {
	t.Parallel()

	t.Run("creates the log directory", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "logs")

		code, _, _ := run(t, "config", "show", "--log-channel", "both", "--log-path", dir)

		assert.Equal(t, 0, code, "the setup opens the log file of the file channels")
		assert.DirExists(t, dir, "the log directory is created")
	})

	t.Run("falls back to stdout for an unusable log directory", func(t *testing.T) {
		t.Parallel()

		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

		code, _, _ := run(t, "config", "show", "--log-channel", "file", "--log-path", filepath.Join(blocker, "logs"))

		assert.Equal(t, 0, code, "a log directory that is not writable does not stop a command")
	})

	t.Run("rejects an invalid configuration", func(t *testing.T) {
		t.Parallel()

		code, _, stderr := run(t, "config", "show", "--server-port", "0")

		require.Equal(t, 1, code)
		assert.Contains(t, stderr, "loading the configuration")
	})
}

func TestLogOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the log settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.LogConfig{
			Level:      zerolog.InfoLevel,
			Channel:    config.LogChannelBoth,
			Format:     config.LogFormatConsole,
			Path:       t.TempDir(),
			MaxSizeMB:  f.IntRange(1, 100),
			MaxAgeDays: f.IntRange(1, 100),
			Compress:   true,
		}

		got, err := logOptions(cfg)

		require.NoError(t, err)
		assert.Equal(t, logging.Options{
			Level:   zerolog.InfoLevel,
			Channel: logging.ChannelBoth,
			Format:  logging.FormatConsole,
			File: logging.FileOptions{
				Filename:   filepath.Join(cfg.Path, config.AppName+".log"),
				MaxSizeMB:  cfg.MaxSizeMB,
				MaxAgeDays: cfg.MaxAgeDays,
				Compress:   true,
			},
		}, got)
	})

	t.Run("keeps the log directory out for stdout", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "logs")

		got, err := logOptions(config.LogConfig{
			Channel: config.LogChannelStdout,
			Path:    dir,
		})

		require.NoError(t, err)
		assert.Empty(t, got.File.Filename)
		assert.NoDirExists(t, dir, "stdout leaves the log directory alone")
	})

	t.Run("rejects an unusable directory", func(t *testing.T) {
		t.Parallel()

		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

		_, err := logOptions(config.LogConfig{
			Channel: config.LogChannelFile,
			Path:    filepath.Join(blocker, "logs"),
		})

		assert.Error(t, err)
	})
}
