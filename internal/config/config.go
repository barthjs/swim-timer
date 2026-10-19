package config

import (
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"

	"github.com/barthjs/swim-timer/internal/appdirs"
	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// AppName names the binary and the application's config and data directories.
const AppName = "swim-timer"

// AppTitle is the display name of the application.
const AppTitle = "Swim-Timer"

// AppRepository is the GitHub repository the releases are published in, as owner/name.
const AppRepository = "barthjs/swim-timer"

// Length limits of text settings.
const (
	maxShortText = 100
	maxText      = 255
	maxPath      = 1024
)

// Config holds the application configuration, one field per section.
// Its settings are declared by struct tags (see structconf).
type Config struct {
	App     AppConfig     `mapstructure:"app" json:"app"`
	Log     LogConfig     `mapstructure:"log" json:"log"`
	Server  ServerConfig  `mapstructure:"server" json:"server"`
	Storage StorageConfig `mapstructure:"storage" json:"storage"`
	Theme   ThemeConfig   `mapstructure:"theme" json:"theme"`
	Timing  TimingConfig  `mapstructure:"timing" json:"timing"`
	Webhook WebhookConfig `mapstructure:"webhook" json:"webhook"`
}

// Load resolves the configuration from command-line flags, environment variables, a config file,
// and the defaults (see structconf.Load), and validates it. A .env file in the working directory,
// if present, fills the environment first. The flags may be nil for a command that registers none.
func Load(flags *pflag.FlagSet) (Config, error) {
	// The .env file is optional, so an error reading it is ignored
	_ = godotenv.Load()

	var cfg Config
	if err := structconf.Load(AppName, flags, &cfg); err != nil {
		return Config{}, err
	}

	if err := Validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Defaults returns the configuration built from the struct-tag defaults alone. It is not validated yet,
// since settings without a default, such as secrets, may be required. Tests set what they need and then call Validate.
func Defaults() (Config, error) {
	var cfg Config
	if err := structconf.Defaults(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate validates every section of cfg and normalizes its values in place.
// It reports every invalid setting at once instead of stopping at the first one.
func Validate(cfg *Config) error {
	var p structconf.Problems

	p.Add(validateApp(&cfg.App))
	p.Add(validateLog(&cfg.Log))
	p.Add(validateServer(&cfg.Server))
	p.Add(validateStorage(&cfg.Storage))
	p.Add(validateTheme(&cfg.Theme))
	p.Add(validateTiming(&cfg.Timing))
	p.Add(validateWebhook(&cfg.Webhook))
	p.Add(structconf.CheckValues(cfg))

	return p.Err()
}

// ResolveDataPath returns path, or if it is empty, subdir below the application's data directory
// (see appdirs.DataDir), so a default location does not depend on the working directory.
func ResolveDataPath(path, subdir string) (string, error) {
	if path != "" {
		return path, nil
	}

	dir, err := appdirs.DataDir(AppName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, subdir), nil
}

// DataPaths returns the locations ResolveDataPath chooses from, in the order it tries them, without creating any:
// path alone, or if it is empty, subdir below each candidate of the data directory (see appdirs.DataDirs).
func DataPaths(path, subdir string) []string {
	if path != "" {
		return []string{
			path,
		}
	}

	dirs := appdirs.DataDirs(AppName)
	paths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		paths = append(paths, filepath.Join(dir, subdir))
	}
	return paths
}

// cleanPath trims and cleans a configured path. An empty path stays empty instead of becoming ".",
// since it selects the default location (see ResolveDataPath).
func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
