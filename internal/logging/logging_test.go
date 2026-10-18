package logging_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/logging"
)

func TestNew(t *testing.T) {
	t.Parallel()

	stdoutTests := []struct {
		name   string
		format logging.Format
		want   string
	}{
		{
			name:   "writes json to stdout above the level",
			format: logging.FormatJSON,
			want:   `"message":"hello"`,
		},
		{
			name:   "writes console lines to stdout above the level",
			format: logging.FormatConsole,
			want:   "hello",
		},
	}

	for _, tt := range stdoutTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			logger, closer, err := logging.New(logging.Options{
				Level:   zerolog.InfoLevel,
				Channel: logging.ChannelStdout,
				Format:  tt.format,
				Stdout:  &stdout,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = closer.Close() })

			logger.Info().Msg("hello")
			logger.Debug().Msg("hidden")

			assert.Contains(t, stdout.String(), tt.want)
			assert.NotContains(t, stdout.String(), "hidden")
		})
	}

	t.Run("writes a sortable console time", func(t *testing.T) {
		t.Parallel()

		var stdout bytes.Buffer
		logger, closer, err := logging.New(logging.Options{
			Level:   zerolog.InfoLevel,
			Channel: logging.ChannelStdout,
			Format:  logging.FormatConsole,
			Stdout:  &stdout,
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = closer.Close() })

		logger.Info().Msg(apitest.Faker(t).Word())

		assert.Regexp(t, `^\x1b\[90m\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\x1b\[0m `, stdout.String())
	})

	fileTests := []struct {
		name       string
		channel    logging.Channel
		wantStdout bool
	}{
		{
			name:       "writes only to the file on the file channel",
			channel:    logging.ChannelFile,
			wantStdout: false,
		},
		{
			name:       "writes to the file and stdout on both channels",
			channel:    logging.ChannelBoth,
			wantStdout: true,
		},
	}

	for _, tt := range fileTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "app.log")
			var stdout bytes.Buffer
			logger, closer, err := logging.New(logging.Options{
				Level:   zerolog.InfoLevel,
				Channel: tt.channel,
				Format:  logging.FormatConsole,
				File: logging.FileOptions{
					Filename: path,
				},
				Stdout: &stdout,
			})
			require.NoError(t, err)

			logger.Info().Msg("written")
			require.NoError(t, closer.Close())

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Contains(t, string(content), "written")
			assert.NotContains(t, string(content), "\x1b[", "a log file gets no color codes")
			assert.Equal(t, tt.wantStdout, bytes.Contains(stdout.Bytes(), []byte("written")))
		})
	}

	fallbackTests := []struct {
		name    string
		channel logging.Channel
	}{
		{
			name:    "falls back to stdout when the file of the file channel fails",
			channel: logging.ChannelFile,
		},
		{
			name:    "falls back to stdout once when the file of both channels fails",
			channel: logging.ChannelBoth,
		},
	}

	for _, tt := range fallbackTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			blocker := filepath.Join(t.TempDir(), "blocker")
			require.NoError(t, os.WriteFile(blocker, nil, 0o600))
			message := apitest.Faker(t).UUID()
			var stdout bytes.Buffer
			logger, closer, err := logging.New(logging.Options{
				Level:   zerolog.InfoLevel,
				Channel: tt.channel,
				Format:  logging.FormatJSON,
				File: logging.FileOptions{
					Filename: filepath.Join(blocker, "app.log"),
					Errors:   io.Discard,
				},
				Stdout: &stdout,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = closer.Close() })

			logger.Info().Msg(message)

			assert.Equal(t, 1, strings.Count(stdout.String(), message), "the line reaches stdout exactly once")
		})
	}

	t.Run("rejects a file channel without a filename", func(t *testing.T) {
		t.Parallel()

		_, closer, err := logging.New(logging.Options{
			Channel: logging.ChannelFile,
		})

		require.Error(t, err)
		assert.NoError(t, closer.Close())
	})
}
