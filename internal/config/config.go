package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Server struct {
	Host string `mapstructure:"host" json:"host"`
	Port int    `mapstructure:"port" json:"port"`
}

type Config struct {
	ConfigFilePath string        `mapstructure:"-" json:"config_file_path"`
	LogLevel       zerolog.Level `mapstructure:"-" json:"-"`
	RawLogLevel    string        `mapstructure:"log_level" json:"log_level"`
	PoolLength     int           `mapstructure:"pool_length" json:"pool_length"`
	PresetsDir     string        `mapstructure:"presets_dir" json:"presets_dir"`
	RaceDataDir    string        `mapstructure:"race_data_dir" json:"race_data_dir"`
	SerialPort     string        `mapstructure:"serial_port" json:"serial_port"`
	SimulateEvent  int           `mapstructure:"simulate_event" json:"simulate_event"`
	Server         Server        `mapstructure:"server" json:"server"`
	Secret         string        `mapstructure:"secret" json:"secret"`
}

// Load reads the configuration from defaults, environment variables, config file, and CLI flags.
// Precedence order: defaults → config file → env vars → CLI flags
// A config file is searched in these locations (first found is used):
//   - CLI-specified path (if provided)
//   - ~/.config/swim-timer/config.yaml (User config directory)
//   - ./config.yaml (Current directory)
func Load(cliConfigPath string, flags *pflag.FlagSet) (Config, error) {
	// Load the .env file if it exists (errors are ignored as the file is optional)
	_ = godotenv.Load()

	v := viper.New()

	var defaultDataDir string
	if configDir, err := os.UserConfigDir(); err == nil {
		defaultDataDir = filepath.Join(configDir, "swim-timer")
	} else {
		defaultDataDir = "."
	}

	// Defaults
	v.SetDefault("log_level", "warn")
	v.SetDefault("pool_length", 50)
	v.SetDefault("presets_dir", filepath.Join(defaultDataDir, "data/presets"))
	v.SetDefault("race_data_dir", filepath.Join(defaultDataDir, "data/race-data"))
	v.SetDefault("simulate_event", 0)
	v.SetDefault("serial_port", "")
	v.SetDefault("server.host", "*")
	v.SetDefault("server.port", 8080)
	v.SetDefault("secret", "")

	// Try to find and load config file from multiple locations
	var configFileUsed string
	if cliConfigPath != "" {
		// Explicit CLI path takes precedence
		configFileUsed = cliConfigPath
		if info, err := os.Stat(cliConfigPath); err == nil && !info.IsDir() {
			v.SetConfigFile(cliConfigPath)
			_ = v.ReadInConfig()
		}
	} else {
		configPaths := []string{
			filepath.Join(defaultDataDir, "config.yaml"),
			"./config.yaml",
		}

		for _, path := range configPaths {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				v.SetConfigFile(path)
				if err := v.ReadInConfig(); err == nil {
					configFileUsed = path
					break
				}
			}
		}

		// If no config file found, create one at the default location if possible
		if configFileUsed == "" && defaultDataDir != "." {
			defaultConfigPath := filepath.Join(defaultDataDir, "config.yaml")
			if err := os.MkdirAll(defaultDataDir, 0755); err == nil {
				v.SetConfigFile(defaultConfigPath)
				if err := v.SafeWriteConfigAs(defaultConfigPath); err == nil {
					configFileUsed = defaultConfigPath
				}
			}
		}
	}

	// Bind environment variables
	_ = v.BindEnv("log_level", "LOG_LEVEL")
	_ = v.BindEnv("pool_length", "POOL_LENGTH")
	_ = v.BindEnv("presets_dir", "PRESET_DIR")
	_ = v.BindEnv("race_data_dir", "RACE_DATA_DIR")
	_ = v.BindEnv("simulate_event", "SIMULATE_EVENT")
	_ = v.BindEnv("serial_port", "SERIAL_PORT")
	_ = v.BindEnv("server.host", "SERVER_HOST")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("secret", "SECRET")

	// Bind CLI flags
	if flags != nil {
		_ = v.BindPFlag("log_level", flags.Lookup("log-level"))
		_ = v.BindPFlag("pool_length", flags.Lookup("pool-length"))
		_ = v.BindPFlag("presets_dir", flags.Lookup("preset-dir"))
		_ = v.BindPFlag("race_data_dir", flags.Lookup("race-data-dir"))
		_ = v.BindPFlag("serial_port", flags.Lookup("serial-port"))
		_ = v.BindPFlag("simulate_event", flags.Lookup("simulate-event"))
		_ = v.BindPFlag("server.host", flags.Lookup("host"))
		_ = v.BindPFlag("server.port", flags.Lookup("port"))
		_ = v.BindPFlag("secret", flags.Lookup("secret"))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	if configFileUsed != "" {
		absPath, err := filepath.Abs(configFileUsed)
		if err == nil {
			cfg.ConfigFilePath = absPath
		} else {
			cfg.ConfigFilePath = configFileUsed
		}
	}

	// Ensure data directories exist and are writable
	if err := ensureWritableDir(cfg.PresetsDir); err != nil {
		return Config{}, err
	}
	if err := ensureWritableDir(cfg.RaceDataDir); err != nil {
		return Config{}, err
	}

	cfg.LogLevel = parseLogLevel(cfg.RawLogLevel)
	cfg.Server.Host = normalizeHost(cfg.Server.Host)

	return cfg, nil
}

func ensureWritableDir(path string) error {
	if path == "" {
		return errors.New("directory path cannot be empty")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	return nil
}

// parseLogLevel converts a string to a zerolog level safely.
func parseLogLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.WarnLevel
	}
}

func normalizeHost(host string) string {
	switch strings.TrimSpace(host) {
	case "", "*", "0.0.0.0", "::":
		return ""
	default:
		return host
	}
}

// WriteConfig writes the configuration to the config file.
func WriteConfig(cfg Config) error {
	if cfg.ConfigFilePath == "" {
		return errors.New("config file path not set")
	}

	if err := ensureWritableDir(filepath.Dir(cfg.ConfigFilePath)); err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("log_level", cfg.RawLogLevel)
	v.Set("pool_length", cfg.PoolLength)
	v.Set("preset_dir", cfg.PresetsDir)
	v.Set("race_data_dir", cfg.RaceDataDir)
	v.Set("serial_port", cfg.SerialPort)
	v.Set("simulate_event", cfg.SimulateEvent)
	v.Set("server.host", cfg.Server.Host)
	v.Set("server.port", cfg.Server.Port)
	v.Set("secret", cfg.Secret)

	if err := v.WriteConfigAs(cfg.ConfigFilePath); err != nil {
		return err
	}

	return nil
}
