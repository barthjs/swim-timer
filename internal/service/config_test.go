package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
)

func TestConfigService_Save(t *testing.T) {
	t.Parallel()

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
}

func TestConfigService_View(t *testing.T) {
	t.Parallel()

	t.Run("returns the configuration without secrets and the defaults", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t, func(cfg *config.Config) {
			useS3(cfg)
			cfg.Storage.S3.SecretKey = apitest.Faker(t).Password(true, true, true, false, false, 16)
			cfg.Server.Host = "127.0.0.1"
		})
		svc := NewConfigService(cfg, ConfigOptions{})

		view, err := svc.View()

		require.NoError(t, err)
		defaults, err := config.Defaults()
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", view.Config.Server.Host)
		assert.Equal(t, defaults.Server.Port, view.Defaults.Server.Port)
		assert.Empty(t, view.Defaults.Server.Host, "the defaults are normalized like a loaded configuration")
		assert.Empty(t, view.Defaults.App.Secret)
		assert.Empty(t, view.Config.App.Secret)
		assert.Empty(t, view.Config.Storage.S3.SecretKey)
		assert.True(t, view.Secrets["app.secret"])
		assert.True(t, view.Secrets["storage.s3.secretKey"])
		assert.Equal(t, cfg.App.Secret, svc.Get().App.Secret, "viewing does not change the configuration")
		assert.Equal(t, ConfigFile{}, view.File, "without a file found at startup, a save picks the location")
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
