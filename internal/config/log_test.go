package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

func TestValidateLog(t *testing.T) {
	t.Parallel()

	t.Run("defaults the rotation", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Log.MaxSizeMB = 0
		cfg.Log.MaxAgeDays = 0

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, 100, cfg.Log.MaxSizeMB)
		assert.Equal(t, 30, cfg.Log.MaxAgeDays)
	})

	t.Run("normalizes the path", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path string
			want string
		}{
			{
				name: "keeps an empty path for the default location",
				path: " ",
				want: "",
			},
			{
				name: "cleans a set path",
				path: " logs/../app/ ",
				want: "app",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				cfg := testConfig(t)
				cfg.Log.Channel = LogChannelFile
				cfg.Log.Path = tt.path

				err := Validate(&cfg)

				require.NoError(t, err, "the file channels need no path")
				assert.Equal(t, tt.want, cfg.Log.Path)
			})
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "unknown channel",
				change:  func(c *Config) { c.Log.Channel = "syslog" },
				wantErr: "log.channel",
			},
			{
				name:    "path too long",
				change:  func(c *Config) { c.Log.Path = strings.Repeat("a", maxPath+1) },
				wantErr: "log.path",
			},
			{
				name:    "unknown format",
				change:  func(c *Config) { c.Log.Format = "xml" },
				wantErr: "log.format",
			},
			{
				name:    "size below one",
				change:  func(c *Config) { c.Log.MaxSizeMB = -1 },
				wantErr: "log.max_size_mb",
			},
			{
				name:    "age below minus one",
				change:  func(c *Config) { c.Log.MaxAgeDays = -2 },
				wantErr: "log.max_age_days",
			},
		})
	})

	t.Run("keeps an unknown level lenient", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Log.RawLevel = "trace"

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, zerolog.WarnLevel, cfg.Log.Level)
		assert.Equal(t, "warn", cfg.Log.RawLevel)
	})
}

func TestResolveLogDir(t *testing.T) {
	t.Run("creates the configured directory", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "nested", "logs")

		got, err := ResolveLogDir(LogConfig{
			Path: dir,
		})

		require.NoError(t, err)
		assert.Equal(t, dir, got)
		assert.DirExists(t, got)
	})

	t.Run("returns the log directory of the system or the user for an empty path", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		t.Setenv("LocalAppData", t.TempDir())

		got, err := ResolveLogDir(LogConfig{})

		require.NoError(t, err)
		assert.Contains(t, appdirs.LogDirs(AppName), got)
	})

	t.Run("rejects a directory that cannot be created", func(t *testing.T) {
		t.Parallel()

		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

		_, err := ResolveLogDir(LogConfig{
			Path: filepath.Join(blocker, "logs"),
		})

		assert.Error(t, err)
	})
}

func TestNormalizeLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
		want  zerolog.Level
	}{
		{
			name:  "selects debug",
			level: "debug",
			want:  zerolog.DebugLevel,
		},
		{
			name:  "ignores the case",
			level: "INFO",
			want:  zerolog.InfoLevel,
		},
		{
			name:  "selects warn for warning",
			level: "warning",
			want:  zerolog.WarnLevel,
		},
		{
			name:  "selects error",
			level: "error",
			want:  zerolog.ErrorLevel,
		},
		{
			name:  "selects warn for an unknown level",
			level: "trace",
			want:  zerolog.WarnLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			level := normalizeLogLevel(tt.level)

			assert.Equal(t, tt.want, level)
		})
	}
}
