package config

import (
	"fmt"
	"net/netip"

	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// WebhookConfig holds the settings of the webhooks.
type WebhookConfig struct {
	AllowedNetworks    []netip.Prefix `mapstructure:"-" json:"-"`
	RawAllowedNetworks string         `mapstructure:"allowed_networks" json:"allowedNetworks" usage:"Comma-separated CIDRs or IPs of private or local networks webhooks may be delivered to"`
}

// validateWebhook normalizes and checks the settings of the webhooks.
func validateWebhook(cfg *WebhookConfig) error {
	var p structconf.Problems

	networks, err := ParseNetworks(cfg.RawAllowedNetworks)
	if err != nil {
		p.Add(fmt.Errorf("webhook.allowed_networks invalid: %w", err))
	} else {
		cfg.AllowedNetworks = networks
	}

	return p.Err()
}
