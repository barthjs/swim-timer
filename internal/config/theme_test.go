package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTheme(t *testing.T) {
	t.Parallel()

	t.Run("trims the colors", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Theme.Primary = " #abc "

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, "#abc", cfg.Theme.Primary)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "named color",
				change:  func(c *Config) { c.Theme.Primary = "blue" },
				wantErr: "theme.primary",
			},
			{
				name:    "hex without hash",
				change:  func(c *Config) { c.Theme.Neutral = "abcdef" },
				wantErr: "theme.neutral",
			},
		})
	})
}
