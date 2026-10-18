package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/appdirs"
	"github.com/barthjs/swim-timer/internal/config/structconf"
)

// LogChannel selects where the log is written.
type LogChannel string

const (
	LogChannelStdout LogChannel = "stdout"
	LogChannelFile   LogChannel = "file"
	LogChannelBoth   LogChannel = "both"
)

// LogFormat selects how log entries are written.
type LogFormat string

const (
	LogFormatJSON    LogFormat = "json"
	LogFormatConsole LogFormat = "console"
)

// LogConfig holds the log settings.
type LogConfig struct {
	Level      zerolog.Level `mapstructure:"-" json:"-"`
	RawLevel   string        `mapstructure:"level" json:"level" default:"warn" values:"debug,info,warn,error" usage:"Log level"`
	Channel    LogChannel    `mapstructure:"channel" json:"channel" default:"stdout" values:"stdout,file,both" usage:"Log output channel"`
	Path       string        `mapstructure:"path" json:"path" usage:"Directory of the log files, empty for the log directory of the system or the user"`
	Format     LogFormat     `mapstructure:"format" json:"format" default:"json" values:"json,console" usage:"Log format"`
	MaxSizeMB  int           `mapstructure:"max_size_mb" json:"maxSizeMb" default:"100" usage:"Log file size in megabytes that triggers a rotation before the daily one"`
	MaxAgeDays int           `mapstructure:"max_age_days" json:"maxAgeDays" default:"30" usage:"Days to keep rotated log files, -1 keeps them forever"`
	Compress   bool          `mapstructure:"compress" json:"compress" default:"true" usage:"Compress rotated log files with gzip"`
}

// validateLog normalizes and checks the log settings.
func validateLog(cfg *LogConfig) error {
	var p structconf.Problems

	const (
		defaultMaxSizeMB  = 100
		defaultMaxAgeDays = 30
	)

	cfg.Level = normalizeLogLevel(cfg.RawLevel)
	cfg.RawLevel = cfg.Level.String()

	// An empty path stays empty instead of becoming ".", since it selects the default location
	cfg.Path = strings.TrimSpace(cfg.Path)
	if cfg.Path != "" {
		cfg.Path = filepath.Clean(cfg.Path)
	}
	if len(cfg.Path) > maxPath {
		p.Addf("log.path must be at most %d characters", maxPath)
	}

	// Zero leaves the log rotation at its defaults, like an omitted setting
	if cfg.MaxSizeMB == 0 {
		cfg.MaxSizeMB = defaultMaxSizeMB
	}
	if cfg.MaxAgeDays == 0 {
		cfg.MaxAgeDays = defaultMaxAgeDays
	}
	if cfg.MaxSizeMB < 1 {
		p.Addf("log.max_size_mb must be at least 1, got %d", cfg.MaxSizeMB)
	}
	if cfg.MaxAgeDays < -1 {
		p.Addf("log.max_age_days must be at least 1, or -1 to keep rotated files forever, got %d", cfg.MaxAgeDays)
	}

	return p.Err()
}

// ResolveLogDir returns the writable directory of the log files, creating it if missing: log.path if set,
// else the log directory of the system or the user (see appdirs.LogDir).
func ResolveLogDir(cfg LogConfig) (string, error) {
	if cfg.Path == "" {
		dir, err := appdirs.LogDir(AppName)
		if err != nil {
			return "", fmt.Errorf("resolving the log directory: %w", err)
		}
		return dir, nil
	}

	// The log file is opened lazily, so a directory that is not writable would only lose entries silently
	if !appdirs.DirWritable(cfg.Path) {
		return "", fmt.Errorf("the log directory %s is not writable", cfg.Path)
	}
	return cfg.Path, nil
}

// normalizeLogLevel converts a string to a zerolog.Level. Unknown values select warn.
func normalizeLogLevel(level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.WarnLevel
	}
}
