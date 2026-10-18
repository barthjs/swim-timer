package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
)

func TestConfigService_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the configuration", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")
		current := testConfig(t, nil)
		svc := NewConfigService(current, ConfigOptions{
			FilePath: configPath,
		})
		name := apitest.Faker(t).Company()
		next := testConfig(t, nil)
		next.App.Name = "  " + name + "  "
		next.App.Secret = ""

		err := svc.Save(next, nil)

		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(name), svc.Get().App.Name, "the saved configuration is normalized")
		assert.Equal(t, current.App.Secret, svc.Get().App.Secret, "an empty secret keeps its value")
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), strings.TrimSpace(name))
		assert.Contains(t, string(data), current.App.Secret, "the kept secret is written as well")
	})

	t.Run("replaces a secret", func(t *testing.T) {
		t.Parallel()

		svc := NewConfigService(testConfig(t, nil), ConfigOptions{
			FilePath: filepath.Join(t.TempDir(), "config.yaml"),
		})
		replaced := testConfig(t, nil)

		err := svc.Save(replaced, nil)

		require.NoError(t, err)
		assert.Equal(t, replaced.App.Secret, svc.Get().App.Secret)
	})

	t.Run("clears a secret", func(t *testing.T) {
		t.Parallel()

		svc := NewConfigService(testConfig(t, nil), ConfigOptions{
			FilePath: filepath.Join(t.TempDir(), "config.yaml"),
		})
		cleared := testConfig(t, nil)
		cleared.App.Secret = ""

		err := svc.Save(cleared, []string{"app.secret"})

		require.NoError(t, err)
		assert.Empty(t, svc.Get().App.Secret)
	})

	t.Run("keeps a stored secret for the selected driver", func(t *testing.T) {
		t.Parallel()

		current := testConfig(t, func(cfg *config.Config) {
			useS3(cfg)
			cfg.Storage.S3.SecretKey = apitest.Faker(t).Password(true, true, true, false, false, 16)
		})
		svc := NewConfigService(current, ConfigOptions{
			FilePath: filepath.Join(t.TempDir(), "config.yaml"),
		})
		// The admin interface sends the configuration without its secrets
		next := testConfig(t, nil)
		useS3(&next)

		err := svc.Save(next, nil)

		require.NoError(t, err)
		assert.Equal(t, current.Storage.S3.SecretKey, svc.Get().Storage.S3.SecretKey)
	})

	t.Run("rejects an invalid configuration", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{
			FilePath: configPath,
		})
		next := testConfig(t, nil)
		next.Server.Port = 0

		err := svc.Save(next, nil)

		_, ok := errors.AsType[*InvalidConfigError](err)
		require.True(t, ok, "%v", err)
		assert.NotEqual(t, 0, svc.Get().Server.Port)
		_, statErr := os.Stat(configPath)
		assert.ErrorIs(t, statErr, os.ErrNotExist, "nothing is written")
	})

	t.Run("rejects a missing secret of a selected driver", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*config.Config, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "s3 secret key",
				change: func(cfg *config.Config, _ *gofakeit.Faker) { useS3(cfg) },
				field:  "storage.s3.secretKey",
				tag:    "required_if",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				configPath := filepath.Join(t.TempDir(), "config.yaml")
				svc := NewConfigService(testConfig(t, nil), ConfigOptions{
					FilePath: configPath,
				})
				next := testConfig(t, nil)
				tt.change(&next, apitest.Faker(t))

				err := svc.Save(next, nil)

				assertValidation(t, err, tt.field, tt.tag)
				_, statErr := os.Stat(configPath)
				assert.ErrorIs(t, statErr, os.ErrNotExist, "nothing is written")
			})
		}
	})
}

func TestConfigService_View(t *testing.T) {
	t.Parallel()

	t.Run("returns the configuration without secrets", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t, func(cfg *config.Config) {
			useS3(cfg)
			cfg.Storage.S3.SecretKey = apitest.Faker(t).Password(true, true, true, false, false, 16)
		})
		svc := NewConfigService(cfg, ConfigOptions{})

		view, err := svc.View()

		require.NoError(t, err)
		assert.Empty(t, view.Config.App.Secret)
		assert.Empty(t, view.Config.Storage.S3.SecretKey)
		assert.True(t, view.Secrets["app.secret"])
		assert.True(t, view.Secrets["storage.s3.secretKey"])
		assert.Equal(t, cfg.App.Secret, svc.Get().App.Secret, "viewing does not change the configuration")
		assert.Equal(t, ConfigFile{}, view.File, "without a file found at startup, a save picks the location")
	})

	t.Run("returns the defaults", func(t *testing.T) {
		t.Parallel()

		svc := NewConfigService(testConfig(t, func(cfg *config.Config) {
			cfg.Server.Host = "127.0.0.1"
		}), ConfigOptions{})

		view, err := svc.View()

		require.NoError(t, err)
		defaults, err := config.Defaults()
		require.NoError(t, err)
		assert.Equal(t, defaults.Server.Port, view.Defaults.Server.Port)
		assert.Equal(t, "127.0.0.1", view.Config.Server.Host)
		assert.Empty(t, view.Defaults.Server.Host, "the defaults are normalized like a loaded configuration")
		assert.Empty(t, view.Defaults.App.Secret)
	})

	t.Run("returns the sources before the first save", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{
			FilePath: configPath,
			Sources: map[string]structconf.Source{
				"app.name": {
					Kind: structconf.SourceDefault,
				},
			},
		})

		view, err := svc.View()

		require.NoError(t, err)
		assert.Equal(t, structconf.SourceDefault, view.Sources["app.name"].Kind)
		assert.Equal(t, ConfigFile{
			Path:   configPath,
			Exists: false,
		}, view.File, "the file does not exist before the first save")
	})

	t.Run("returns the file as the source after a save", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")
		env := structconf.Source{
			Kind: structconf.SourceEnv,
			Name: "LOG_LEVEL",
		}
		flag := structconf.Source{
			Kind: structconf.SourceFlag,
			Name: "--server-port",
		}
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{
			FilePath: configPath,
			Sources: map[string]structconf.Source{
				"log.level":   env,
				"server.port": flag,
				"app.name": {
					Kind: structconf.SourceDefault,
				},
			},
		})
		require.NoError(t, svc.Save(svc.Get(), nil))

		view, err := svc.View()

		require.NoError(t, err)
		assert.Equal(t, structconf.Source{
			Kind: structconf.SourceFile,
			Name: configPath,
		}, view.Sources["app.name"], "the saved file holds every setting")
		assert.Equal(t, env, view.Sources["log.level"], "the environment still overrides the file")
		assert.Equal(t, flag, view.Sources["server.port"], "a flag still overrides the file")
		assert.Equal(t, ConfigFile{
			Path:   configPath,
			Exists: true,
		}, view.File)
	})
}

func TestConfigService_TestStorage(t *testing.T) {
	// Not parallel because one subtest changes the environment
	t.Run("changes nothing in the tested directory", func(t *testing.T) {
		t.Parallel()

		parent := t.TempDir()
		stale := filepath.Join(parent, ".upload-1.tmp")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o644))
		old := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(stale, old, old))
		missing := filepath.Join(parent, "missing")
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{})

		err := svc.TestStorage(t.Context(), config.StorageConfig{
			Driver: media.DriverLocal,
			Path:   missing,
		})

		require.NoError(t, err, "a missing directory below a writable parent passes")
		assert.NoDirExists(t, missing, "the test creates no directory")
		assert.FileExists(t, stale, "the test removes no temporary files of the tested directory")
	})

	t.Run("creates no data directory", func(t *testing.T) {
		// The data directory lies below the user directories, which only the environment selects
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("LocalAppData", home)
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{})

		err := svc.TestStorage(t.Context(), config.StorageConfig{
			Driver: media.DriverLocal,
		})

		require.NoError(t, err, "a missing data directory below a writable home passes")
		entries, err := os.ReadDir(home)
		require.NoError(t, err)
		assert.Empty(t, entries, "the test creates no directory")
	})

	t.Run("rejects a file in place of the directory", func(t *testing.T) {
		t.Parallel()

		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
		svc := NewConfigService(testConfig(t, nil), ConfigOptions{})

		err := svc.TestStorage(t.Context(), config.StorageConfig{
			Driver: media.DriverLocal,
			Path:   file,
		})

		assert.Error(t, err)
	})
}

// testConfig returns the normalized default configuration with a random admin secret and the given changes.
func testConfig(t *testing.T, mutate func(*config.Config)) config.Config {
	t.Helper()

	cfg, err := config.Defaults()
	require.NoError(t, err)
	cfg.App.Secret = apitest.Faker(t).Password(true, true, true, false, false, 16)
	if mutate != nil {
		mutate(&cfg)
	}
	require.NoError(t, config.Validate(&cfg))
	return cfg
}

// useS3 selects the S3 storage with every setting except the secret key.
func useS3(cfg *config.Config) {
	cfg.Storage.Driver = media.DriverS3
	cfg.Storage.S3.Endpoint = "https://s3.example.org"
	cfg.Storage.S3.Bucket = "bucket"
	cfg.Storage.S3.AccessKey = "access"
}
