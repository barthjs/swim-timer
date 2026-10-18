package app_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/serial"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("builds the application", func(t *testing.T) {
		t.Parallel()

		application, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), testDeps(t))

		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, application.Close()) })
		assert.Equal(t, http.StatusOK, serve(t, application.Handler(), "/api/ping").Code)
		assert.Equal(t, http.StatusOK, serve(t, application.Handler(), "/healthz").Code)
		// The device is not opened before Start, and it is optional, so the readiness check reports degraded
		ready := serve(t, application.Handler(), "/readyz")
		assert.Equal(t, http.StatusOK, ready.Code)
		assert.Contains(t, ready.Body.String(), `"status":"degraded"`)
	})

	t.Run("uses the given storage", func(t *testing.T) {
		t.Parallel()

		storage, err := media.New(media.Options{
			Driver:    media.DriverLocal,
			LocalPath: t.TempDir(),
		})
		require.NoError(t, err)
		cfg := testConfig(t)
		cfg.Storage = unusableStorage()
		deps := testDeps(t)
		deps.Storage = storage

		application, err := app.New(t.Context(), zerolog.Nop(), cfg, deps)

		require.NoError(t, err, "the storage of the configuration is not built")
		t.Cleanup(func() { _ = application.Close() })
	})

	recordings := []struct {
		name   string
		change func(cfg *config.Config)
		want   bool
	}{
		{
			name:   "records the timing data while it runs",
			change: func(*config.Config) {},
			want:   true,
		},
		{
			name: "records nothing with the recording off",
			change: func(cfg *config.Config) {
				cfg.Timing.Recording = false
			},
			want: false,
		},
		{
			name: "records nothing while it replays a recording",
			change: func(cfg *config.Config) {
				cfg.Timing.ReplayPath = "recordings/recording.jsonl"
			},
			want: false,
		},
	}

	for _, tt := range recordings {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig(t)
			tt.change(&cfg)
			deps := testDeps(t)

			application, err := app.New(t.Context(), zerolog.Nop(), cfg, deps)
			if errors.Is(err, serial.ErrUnsupported) {
				t.Skip("the platform has no virtual serial port")
			}
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			application.Start(ctx)
			cancel()
			require.NoError(t, application.Close())

			files, err := filepath.Glob(filepath.Join(cfg.Storage.Path, "recordings", "recording_*.jsonl"))
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(files) == 1)
		})
	}

	t.Run("rejects an unusable storage", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Storage = unusableStorage()

		_, err := app.New(t.Context(), zerolog.Nop(), cfg, testDeps(t))

		require.Error(t, err)
	})
}

// serve sends a GET request for path to handler and returns the recorded response.
func serve(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return recorder
}

// unusableStorage returns storage settings that New cannot build a storage from.
func unusableStorage() config.StorageConfig {
	return config.StorageConfig{
		Driver: media.DriverS3,
		S3: config.S3Config{
			Endpoint:  "not a url",
			Bucket:    "b",
			AccessKey: "a",
			SecretKey: "s",
		},
	}
}

func TestApp_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes the application twice", func(t *testing.T) {
		t.Parallel()

		application, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), testDeps(t))
		require.NoError(t, err)
		require.NoError(t, application.Close())

		err = application.Close()

		require.NoError(t, err)
	})

	t.Run("waits for the background work", func(t *testing.T) {
		t.Parallel()

		application, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), testDeps(t))
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		application.Start(ctx)

		closed := make(chan error, 1)
		go func() { closed <- application.Close() }()

		select {
		case err := <-closed:
			require.FailNow(t, "Close returned while the background work was still running", err)
		case <-time.After(200 * time.Millisecond):
		}
		cancel()
		select {
		case err := <-closed:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Close did not return after the background work was canceled")
		}
	})

	t.Run("releases the virtual port of the simulator once", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Timing.SimulateEvent = 1
		application, err := app.New(t.Context(), zerolog.Nop(), cfg, testDeps(t))
		if errors.Is(err, serial.ErrUnsupported) {
			t.Skip("the platform has no virtual serial port")
		}
		require.NoError(t, err)
		// Close waits for the simulation, which has closed the port with its context, so Close closes it a second time
		ctx, cancel := context.WithCancel(t.Context())
		application.Start(ctx)
		cancel()

		err = application.Close()

		assert.NoError(t, err, "the second close of the port does nothing")
	})
}

// testConfig returns the normalized default configuration with its storage and logs in a temporary directory.
func testConfig(t *testing.T) config.Config {
	t.Helper()

	root := t.TempDir()
	cfg, err := config.Defaults()
	require.NoError(t, err)
	cfg.Storage.Driver = media.DriverLocal
	cfg.Storage.Path = filepath.Join(root, "data")
	cfg.Log.Path = filepath.Join(root, "logs")
	require.NoError(t, config.Validate(&cfg))
	return cfg
}

// testDeps returns dependencies that keep the saved configuration in a temporary directory.
func testDeps(t *testing.T) app.Deps {
	t.Helper()

	return app.Deps{
		Version:        "test",
		ConfigFilePath: filepath.Join(t.TempDir(), "config.yaml"),
	}
}
