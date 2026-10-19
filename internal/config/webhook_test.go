package config

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhook(t *testing.T) {
	t.Parallel()

	t.Run("parses the allowed networks", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Webhook.RawAllowedNetworks = "192.168.1.0/24"

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, []netip.Prefix{
			netip.MustParsePrefix("192.168.1.0/24"),
		}, cfg.Webhook.AllowedNetworks)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "invalid network",
				change:  func(c *Config) { c.Webhook.RawAllowedNetworks = "not-a-network" },
				wantErr: "webhook.allowed_networks",
			},
		})
	})
}
