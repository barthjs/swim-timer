package app_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
)

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
