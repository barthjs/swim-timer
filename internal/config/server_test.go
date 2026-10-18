package config

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServer(t *testing.T) {
	t.Parallel()

	t.Run("normalizes the origins", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Server.CORSAllowedOrigins = []string{" https://a.test/ ", "", "http://b.test:8080,https://c.test", "*"}

		err := Validate(&cfg)

		require.NoError(t, err)
		assert.Equal(t, []string{"https://a.test", "http://b.test:8080", "https://c.test", "*"}, cfg.Server.CORSAllowedOrigins)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name:    "port zero",
				change:  func(c *Config) { c.Server.Port = 0 },
				wantErr: "server.port",
			},
			{
				name:    "certificate without key",
				change:  func(c *Config) { c.Server.SSLCertFile = "cert.pem" },
				wantErr: "ssl_key_file",
			},
			{
				name:    "invalid proxy",
				change:  func(c *Config) { c.Server.RawTrustedProxies = "not-an-ip" },
				wantErr: "server.trusted_proxies",
			},
			{
				name:    "origin with path",
				change:  func(c *Config) { c.Server.CORSAllowedOrigins = []string{"https://a.test/app"} },
				wantErr: "server.cors_allowed_origins",
			},
			{
				name:    "origin without scheme",
				change:  func(c *Config) { c.Server.CORSAllowedOrigins = []string{"a.test"} },
				wantErr: "server.cors_allowed_origins",
			},
			{
				name:    "origin with query",
				change:  func(c *Config) { c.Server.CORSAllowedOrigins = []string{"https://a.test?x=1"} },
				wantErr: "server.cors_allowed_origins",
			},
			{
				name: "wildcard with credentials",
				change: func(c *Config) {
					c.Server.CORSAllowedOrigins = []string{"*"}
					c.Server.CORSAllowCredentials = true
				},
				wantErr: "server.cors_allow_credentials",
			},
		})
	})
}

func TestNormalizeHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "selects all interfaces for an empty host",
			host: "",
			want: "",
		},
		{
			name: "selects all interfaces for the wildcard",
			host: "*",
			want: "",
		},
		{
			name: "selects all interfaces for the unspecified ipv4 address",
			host: "0.0.0.0",
			want: "",
		},
		{
			name: "selects all interfaces for the unspecified ipv6 address",
			host: "::",
			want: "",
		},
		{
			name: "keeps a specific host",
			host: "127.0.0.1",
			want: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host := normalizeHost(tt.host)

			assert.Equal(t, tt.want, host)
		})
	}
}

func TestNormalizeTrustedProxies(t *testing.T) {
	t.Parallel()

	t.Run("returns the networks", func(t *testing.T) {
		t.Parallel()

		networks, err := normalizeTrustedProxies(" 10.0.0.0/8, 192.168.1.5 ,,::1")

		require.NoError(t, err)
		got := make([]string, 0, len(networks))
		for _, network := range networks {
			got = append(got, network.String())
		}
		assert.Equal(t, []string{"10.0.0.0/8", "192.168.1.5/32", "::1/128"}, got)
	})

	t.Run("treats a plain ip as a single-host network", func(t *testing.T) {
		t.Parallel()

		networks, err := normalizeTrustedProxies("192.168.1.5")

		require.NoError(t, err)
		require.Len(t, networks, 1)
		assert.True(t, networks[0].Contains(net.ParseIP("192.168.1.5")), "a plain IP is a single-host network")
		assert.False(t, networks[0].Contains(net.ParseIP("192.168.1.6")), "a plain IP is a single-host network")
	})

	t.Run("rejects an invalid entry", func(t *testing.T) {
		t.Parallel()

		_, err := normalizeTrustedProxies("10.0.0.0/8,nope")

		assert.ErrorContains(t, err, "nope")
	})
}

func TestParseNetworks(t *testing.T) {
	t.Parallel()

	t.Run("returns the networks", func(t *testing.T) {
		t.Parallel()

		networks, err := ParseNetworks(" 10.0.0.0/8, 192.168.1.5 ,,fd00::1/64")

		require.NoError(t, err)
		assert.Equal(t, []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/8"),
			netip.MustParsePrefix("192.168.1.5/32"),
			netip.MustParsePrefix("fd00::/64"),
		}, networks, "a plain IP is a single-host network and a CIDR is masked")
	})

	t.Run("returns no networks for an empty list", func(t *testing.T) {
		t.Parallel()

		networks, err := ParseNetworks("")

		require.NoError(t, err)
		assert.Empty(t, networks)
	})

	t.Run("rejects an invalid entry", func(t *testing.T) {
		t.Parallel()

		_, err := ParseNetworks("10.0.0.0/8,nope")

		assert.ErrorContains(t, err, "nope")
	})
}

func TestParseOriginList(t *testing.T) {
	t.Parallel()

	t.Run("returns the trimmed origins without empty entries", func(t *testing.T) {
		t.Parallel()

		origins, err := ParseOriginList(" https://a.test:8443/ ,, *,http://b.test")

		require.NoError(t, err)
		assert.Equal(t, []string{"https://a.test:8443", "*", "http://b.test"}, origins)
	})

	t.Run("rejects an invalid origin", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "host without scheme",
				value: "a.test",
			},
			{
				name:  "path",
				value: "https://a.test/app",
			},
			{
				name:  "query",
				value: "https://a.test?x=1",
			},
			{
				name:  "other scheme",
				value: "https://b.test,ftp://a.test",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := ParseOriginList(tt.value)

				assert.Error(t, err)
			})
		}
	})
}
