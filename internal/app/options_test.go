package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/service"
)

func TestStorageOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the s3 settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.StorageConfig{
			Driver: media.DriverS3,
			S3: config.S3Config{
				Endpoint:     f.URL(),
				Region:       f.LetterN(8),
				Bucket:       f.LetterN(8),
				AccessKey:    f.LetterN(16),
				SecretKey:    f.LetterN(32),
				UsePathStyle: true,
			},
		}

		got, err := StorageOptions(cfg)

		require.NoError(t, err)
		assert.Equal(t, media.Options{
			Driver:         media.DriverS3,
			S3Endpoint:     cfg.S3.Endpoint,
			S3Region:       cfg.S3.Region,
			S3Bucket:       cfg.S3.Bucket,
			S3AccessKey:    cfg.S3.AccessKey,
			S3SecretKey:    cfg.S3.SecretKey,
			S3UsePathStyle: cfg.S3.UsePathStyle,
		}, got)
	})

	t.Run("keeps the configured path of the local driver", func(t *testing.T) {
		t.Parallel()

		path := t.TempDir()

		got, err := StorageOptions(config.StorageConfig{
			Driver: media.DriverLocal,
			Path:   path,
		})

		require.NoError(t, err)
		assert.Equal(t, path, got.LocalPath)
	})
}

func TestAuthOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the admin password", func(t *testing.T) {
		t.Parallel()

		cfg := config.AppConfig{
			Secret: apitest.Faker(t).LetterN(16),
		}

		got := authOptions(cfg)

		assert.Equal(t, service.AuthOptions{
			Password: cfg.Secret,
		}, got)
	})
}

func TestSimulatorOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the simulated event, the pool length, and the replay path", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.TimingConfig{
			PoolLength:    f.IntRange(25, 50),
			SimulateEvent: f.IntRange(1, 99),
			ReplayPath:    "recordings/" + f.LetterN(8) + ".jsonl",
		}

		got := simulatorOptions(cfg)

		assert.Equal(t, service.SimulatorOptions{
			Event:      cfg.SimulateEvent,
			PoolLength: cfg.PoolLength,
			ReplayPath: cfg.ReplayPath,
		}, got)
	})
}

func TestTimingOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the timing settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.TimingConfig{
			SerialPort:    f.LetterN(8),
			PoolLength:    f.IntRange(25, 50),
			SimulateEvent: f.IntRange(1, 99),
		}

		got := timingOptions(cfg)

		assert.Equal(t, service.TimingOptions{
			PoolLength:     cfg.PoolLength,
			ConfiguredPort: cfg.SerialPort,
			SimulateEvent:  cfg.SimulateEvent,
		}, got)
	})

	t.Run("shows only the name of the replayed recording", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(8) + ".jsonl"
		cfg := config.TimingConfig{
			ReplayPath: "recordings/" + name,
		}

		got := timingOptions(cfg)

		assert.Equal(t, name, got.ReplayFile)
	})
}

func TestRecorderOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the recording of a device", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.TimingConfig{
			SerialPort: f.LetterN(8),
			BaudRate:   19200,
		}
		dir := t.TempDir()
		version := f.AppVersion()

		got := recorderOptions(cfg, dir, version)

		assert.Equal(t, recording.WriterOptions{
			Dir: dir,
			Session: recording.Session{
				Version:  version,
				Source:   recording.SourceDevice,
				Port:     cfg.SerialPort,
				BaudRate: cfg.BaudRate,
			},
		}, got)
	})

	t.Run("marks the recording of the simulator", func(t *testing.T) {
		t.Parallel()

		cfg := config.TimingConfig{
			SimulateEvent: apitest.Faker(t).IntRange(1, 99),
		}

		got := recorderOptions(cfg, t.TempDir(), "")

		assert.Equal(t, recording.SourceSimulator, got.Session.Source)
	})
}

func TestRecordingOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the replay path", func(t *testing.T) {
		t.Parallel()

		cfg := config.TimingConfig{
			ReplayPath: "recordings/" + apitest.Faker(t).LetterN(8) + ".jsonl",
		}

		got := recordingOptions(cfg)

		assert.Equal(t, service.RecordingOptions{
			ReplayPath: cfg.ReplayPath,
		}, got)
	})
}

func TestAPIOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the settings the api reads", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg, err := config.Defaults()
		require.NoError(t, err)
		cfg.App.Name = f.LetterN(8)
		cfg.App.Language = config.LanguageDE
		cfg.Theme.Primary = f.Color()
		cfg.Server.CORSAllowedOrigins = []string{
			f.URL(),
		}
		cfg.Server.CORSAllowCredentials = true

		got := apiOptions(cfg)

		assert.Equal(t, api.Options{
			Name:     cfg.App.Name,
			Language: string(cfg.App.Language),
			Theme: api.ThemeSettings{
				Primary:   cfg.Theme.Primary,
				Secondary: cfg.Theme.Secondary,
				Success:   cfg.Theme.Success,
				Info:      cfg.Theme.Info,
				Warning:   cfg.Theme.Warning,
				Error:     cfg.Theme.Error,
				Neutral:   cfg.Theme.Neutral,
			},
			TrustedProxies:       cfg.Server.TrustedProxies,
			CORSAllowedOrigins:   cfg.Server.CORSAllowedOrigins,
			CORSAllowCredentials: cfg.Server.CORSAllowCredentials,
			Docs:                 true,
		}, got)
	})

}
