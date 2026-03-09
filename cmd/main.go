package main

import (
	"context"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/server"
)

var Version = "dev"

func main() {
	// Pure CLI parameters (not part of the configuration)
	displayVersion := pflag.BoolP("version", "v", false, "Display version")
	daemon := pflag.BoolP("daemon", "d", false, "Start in the background")
	configPath := pflag.StringP("config", "c", "", "Path to configuration file")

	// Configuration parameters
	pflag.StringP("log-level", "l", "", "Logging level (debug, info, warn, error)")
	pflag.Int("pool-length", 50, "Length of the pool (default 50)")
	pflag.String("preset-dir", "", "Directory for camera presets")
	pflag.String("race-data-dir", "", "Directory for race data")
	pflag.String("serial-port", "", "Serial port for Omega connection")
	pflag.IntP("simulate", "s", 0, "Start the simulator for the given event ID (default 1 if flag is provided)")
	pflag.Lookup("simulate").NoOptDefVal = "1"
	pflag.StringP("host", "H", "", "Server host")
	pflag.IntP("port", "p", 0, "Server port")
	pflag.String("secret", "", "Application secret")

	pflag.Parse()

	if *displayVersion {
		printVersion()
		os.Exit(0)
	}

	logger := zerolog.
		New(os.Stdout).
		With().
		Timestamp().
		Logger()

	cfg, err := config.Load(*configPath, pflag.CommandLine)
	if err != nil {
		logger.Error().Err(err).Msg("")
		os.Exit(1)
	}

	logger = logger.Level(cfg.LogLevel)

	if *daemon {
		daemonize(logger)
	}

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.New(
		cfg.Server,
		logger,
		wg,
	)

	go runTray(cancel, cfg.LogLevel, logger, srv.URL())

	if err := srv.Run(ctx); err != nil {
		logger.Error().Err(err).Msg("server run failed")
		os.Exit(1)
	}
}
