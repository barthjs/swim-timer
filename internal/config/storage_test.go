package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/media"
)

func TestValidateStorage(t *testing.T) {
	t.Parallel()

	t.Run("keeps a blank path empty", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Storage.Path = "  "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Empty(t, cfg.Storage.Path, "a blank path stays empty instead of becoming the working directory")
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "unknown driver",
				change:  func(c *Config) { c.Storage.Driver = "ftp" },
				wantErr: "storage.driver",
			},
			{
				name:    "s3 without bucket",
				change:  func(c *Config) { c.Storage.Driver = media.DriverS3 },
				wantErr: "s3",
			},
		})
	})
}
