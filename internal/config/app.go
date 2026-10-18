package config

import (
	"strings"

	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// Environment is the deployment environment of the application.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// Language is the default language of the interface and the messages.
type Language string

const (
	LanguageEN Language = "en"
	LanguageDE Language = "de"
)

// AppConfig holds the application settings.
type AppConfig struct {
	Name     string      `mapstructure:"name" json:"name" default:"Swim-Timer" usage:"Application name shown in the interface, empty for the built-in name"`
	Env      Environment `mapstructure:"env" json:"env" default:"production" values:"development,staging,production" usage:"Environment"`
	Language Language    `mapstructure:"language" json:"language" default:"en" values:"en,de" usage:"Language of the user interface"`
	Secret   string      `mapstructure:"secret" json:"secret" secret:"true" usage:"Admin login password, empty allows the login without a password"`
}

// validateApp normalizes and checks the application settings.
func validateApp(cfg *AppConfig) error {
	var p structconf.Problems

	cfg.Env = normalizeEnv(string(cfg.Env))

	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.Name == "" {
		cfg.Name = AppTitle
	}
	if len(cfg.Name) > maxShortText {
		p.Addf("app.name must be at most %d characters", maxShortText)
	}

	return p.Err()
}

// normalizeEnv converts a string to an Environment. Unknown values select production.
func normalizeEnv(env string) Environment {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "development", "dev", "local":
		return EnvDevelopment
	case "staging":
		return EnvStaging
	default:
		return EnvProduction
	}
}
