package config

import (
	"regexp"
	"strings"

	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// hexColorPattern matches a hex color with three or six digits.
var hexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// ThemeConfig holds the theme settings.
type ThemeConfig struct {
	Primary   string `mapstructure:"primary" json:"primary" usage:"Primary color (hex), empty for the built-in color"`
	Secondary string `mapstructure:"secondary" json:"secondary" usage:"Secondary color (hex), empty for the built-in color"`
	Success   string `mapstructure:"success" json:"success" usage:"Success color (hex), empty for the built-in color"`
	Info      string `mapstructure:"info" json:"info" usage:"Info color (hex), empty for the built-in color"`
	Warning   string `mapstructure:"warning" json:"warning" usage:"Warning color (hex), empty for the built-in color"`
	Error     string `mapstructure:"error" json:"error" usage:"Error color (hex), empty for the built-in color"`
	Neutral   string `mapstructure:"neutral" json:"neutral" usage:"Neutral color (hex), empty for the built-in color"`
}

// validateTheme normalizes and checks the theme settings.
func validateTheme(cfg *ThemeConfig) error {
	var p structconf.Problems

	fields := map[string]*string{
		"theme.primary":   &cfg.Primary,
		"theme.secondary": &cfg.Secondary,
		"theme.success":   &cfg.Success,
		"theme.info":      &cfg.Info,
		"theme.warning":   &cfg.Warning,
		"theme.error":     &cfg.Error,
		"theme.neutral":   &cfg.Neutral,
	}

	for key, value := range fields {
		*value = strings.TrimSpace(*value)
		if *value != "" && !hexColorPattern.MatchString(*value) {
			p.Addf("%s must be a hex color (e.g. #0ea5e9), got %q", key, *value)
		}
	}

	return p.Err()
}
