package app_test

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/update"
)

func TestNewStores(t *testing.T) {
	t.Parallel()

	t.Run("builds the stores on the storage", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		storage, err := app.NewStorage(zerolog.Nop(), cfg)
		require.NoError(t, err)

		stores := app.NewStores(t.Context(), zerolog.Nop(), storage)

		require.NotNil(t, stores.Meet)
		require.NotNil(t, stores.Preset)
		presets, err := stores.Preset.List(t.Context())
		require.NoError(t, err)
		assert.Empty(t, presets, "a fresh storage holds no presets")
		assert.Empty(t, stores.Meet.Snapshot().Events, "missing meet files leave the meet data empty")
	})
}

func TestNewStorage(t *testing.T) {
	t.Parallel()

	t.Run("uses the configured path", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)

		storage, err := app.NewStorage(zerolog.Nop(), cfg)

		require.NoError(t, err)
		require.NoError(t, storage.TestConnection(t.Context()))
		assert.DirExists(t, cfg.Storage.Path)
	})
}

func TestNewUpdater(t *testing.T) {
	t.Parallel()

	t.Run("rejects installs of a development build", func(t *testing.T) {
		t.Parallel()

		updater := app.NewUpdater("dev")

		assert.ErrorIs(t, updater.Supported(), update.ErrDevelopmentBuild)
	})
}
