package cli

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/logging"
)

// state is what the setup prepares before a command runs.
type state struct {
	build      cmdline.Build
	cfg        config.Config
	configPath string
	logger     zerolog.Logger
}

// setup loads the configuration and builds the logger once before a command runs.
func (s *state) setup(cmd *cobra.Command) error {
	cfg, err := config.Load(cmd.Flags())
	if err != nil {
		return fmt.Errorf("loading the configuration: %w", err)
	}

	options, logDirErr := logOptions(cfg.Log)
	if logDirErr != nil {
		// A log directory that is not writable must not stop the application, so the log goes to stdout instead
		options = logging.Options{
			Level:   cfg.Log.Level,
			Channel: logging.ChannelStdout,
			Format:  logging.Format(cfg.Log.Format),
		}
	}
	logger, closer, err := logging.New(options)
	if err != nil {
		return err
	}
	cmdline.Defer(cmd, func() { _ = closer.Close() })
	if logDirErr != nil {
		logger.Warn().Err(logDirErr).Msg("writing the log to stdout only, since the log directory is not writable")
	}

	s.cfg = cfg
	s.configPath = structconf.FindFile(config.AppName, cmd.Flags())
	s.logger = logger
	return nil
}

// logOptions maps the log settings of the configuration to the logger options.
// The file channels write to the file named after the application in the log directory,
// which is created if missing. It fails if the log directory is not writable.
func logOptions(cfg config.LogConfig) (logging.Options, error) {
	options := logging.Options{
		Level:   cfg.Level,
		Channel: logging.Channel(cfg.Channel),
		Format:  logging.Format(cfg.Format),
	}
	if cfg.Channel == config.LogChannelStdout {
		return options, nil
	}

	dir, err := config.ResolveLogDir(cfg)
	if err != nil {
		return logging.Options{}, err
	}

	options.File = logging.FileOptions{
		Filename:   filepath.Join(dir, config.AppName+".log"),
		MaxSizeMB:  cfg.MaxSizeMB,
		MaxAgeDays: cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
	return options, nil
}
