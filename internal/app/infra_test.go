package app_test

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/update"
)

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
