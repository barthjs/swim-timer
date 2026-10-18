package config

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strings"

	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// ServerConfig holds the HTTP server settings.
type ServerConfig struct {
	Host                 string       `mapstructure:"host" json:"host" default:"*" usage:"Listen host, * for all interfaces"`
	Port                 int          `mapstructure:"port" json:"port" default:"8080" usage:"Listen port"`
	TrustedProxies       []*net.IPNet `mapstructure:"-" json:"-"`
	RawTrustedProxies    string       `mapstructure:"trusted_proxies" json:"trustedProxies" default:"127.0.0.0/8,::1/128" usage:"Comma-separated CIDRs or IPs of proxies whose forwarding headers are trusted"`
	CORSAllowedOrigins   []string     `mapstructure:"cors_allowed_origins" json:"corsAllowedOrigins" usage:"Comma-separated origins allowed to call the API from another domain, * for any, empty disables CORS"`
	CORSAllowCredentials bool         `mapstructure:"cors_allow_credentials" json:"corsAllowCredentials" default:"false" usage:"Allow cross-origin requests to send cookies, requires explicit origins"`
	SSLEnabled           bool         `mapstructure:"ssl_enabled" json:"sslEnabled" default:"false" usage:"Serve over HTTPS, with a self-signed certificate if ssl_cert_file and ssl_key_file are empty"`
	SSLCertFile          string       `mapstructure:"ssl_cert_file" json:"sslCertFile" usage:"PEM-encoded TLS certificate file, requires ssl_key_file"`
	SSLKeyFile           string       `mapstructure:"ssl_key_file" json:"sslKeyFile" usage:"PEM-encoded private key file of ssl_cert_file"`
}

// validateServer normalizes and checks the HTTP server settings.
func validateServer(cfg *ServerConfig) error {
	var p structconf.Problems

	cfg.Host = normalizeHost(cfg.Host)
	cfg.SSLCertFile = strings.TrimSpace(cfg.SSLCertFile)
	cfg.SSLKeyFile = strings.TrimSpace(cfg.SSLKeyFile)

	if cfg.Port < 1 || cfg.Port > 65535 {
		p.Addf("server.port must be between 1 and 65535, got %d", cfg.Port)
	}

	trustedProxies, err := normalizeTrustedProxies(cfg.RawTrustedProxies)
	if err != nil {
		p.Add(fmt.Errorf("server.trusted_proxies invalid: %w", err))
	} else {
		cfg.TrustedProxies = trustedProxies
	}

	origins, err := normalizeOrigins(cfg.CORSAllowedOrigins)
	if err != nil {
		p.Add(fmt.Errorf("server.cors_allowed_origins invalid: %w", err))
	} else {
		cfg.CORSAllowedOrigins = origins
	}
	if cfg.CORSAllowCredentials && slices.Contains(cfg.CORSAllowedOrigins, "*") {
		p.Addf("server.cors_allow_credentials requires explicit origins instead of *")
	}

	if (cfg.SSLCertFile == "") != (cfg.SSLKeyFile == "") {
		p.Addf("server.ssl_cert_file and server.ssl_key_file must be set together")
	}
	if len(cfg.SSLCertFile) > maxPath {
		p.Addf("server.ssl_cert_file must be at most %d characters", maxPath)
	}
	if len(cfg.SSLKeyFile) > maxPath {
		p.Addf("server.ssl_key_file must be at most %d characters", maxPath)
	}

	return p.Err()
}

// normalizeHost maps the spellings of "all interfaces" to the empty host.
func normalizeHost(host string) string {
	switch strings.TrimSpace(host) {
	case "", "*", "0.0.0.0", "::":
		return ""
	default:
		return host
	}
}

// normalizeTrustedProxies converts a comma-separated list of CIDRs or plain IPs into the networks
// the server middleware compares the client addresses with, see ParseNetworks.
func normalizeTrustedProxies(raw string) ([]*net.IPNet, error) {
	prefixes, err := ParseNetworks(raw)
	if err != nil {
		return nil, err
	}

	networks := make([]*net.IPNet, 0, len(prefixes))
	for _, prefix := range prefixes {
		networks = append(networks, &net.IPNet{
			IP:   prefix.Addr().AsSlice(),
			Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
		})
	}
	return networks, nil
}

// ParseNetworks converts a comma-separated list of CIDRs or plain IPs into networks, so every setting of networks
// is read the same way. Plain IPs are treated as single-host networks.
func ParseNetworks(raw string) ([]netip.Prefix, error) {
	var networks []netip.Prefix
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if network, err := netip.ParsePrefix(entry); err == nil {
			networks = append(networks, network.Masked())
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid network entry %q", entry)
		}
		networks = append(networks, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return networks, nil
}

// normalizeOrigins trims the origins, drops empty entries, and rejects anything but "*" or scheme://host[:port].
func normalizeOrigins(values []string) ([]string, error) {
	origins := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := ParseOriginList(value)
		if err != nil {
			return nil, err
		}
		origins = append(origins, parsed...)
	}
	return origins, nil
}

// ParseOriginList reads allowed CORS origins separated by commas, trims each and its trailing slash, and drops
// empty ones. Every origin must be "*" or scheme://host[:port], so the API checks an entry as the configuration reads it.
func ParseOriginList(value string) ([]string, error) {
	var origins []string
	for part := range strings.SplitSeq(value, ",") {
		origin := strings.TrimRight(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		if origin != "*" && !isOrigin(origin) {
			return nil, fmt.Errorf("%q is not an origin like https://example.com", origin)
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

// isOrigin reports whether value is a bare scheme://host[:port] over http or https.
func isOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != "" && parsed.Path == "" && parsed.RawQuery == ""
}
