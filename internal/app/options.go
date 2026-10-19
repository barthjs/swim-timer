package app

import (
	"path"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/webhook"
)

// StorageOptions maps the storage settings onto the options of the media package.
// An empty path of the local driver selects the application's data directory.
func StorageOptions(cfg config.StorageConfig) (media.Options, error) {
	options := media.Options{
		Driver:         cfg.Driver,
		LocalPath:      cfg.Path,
		S3Endpoint:     cfg.S3.Endpoint,
		S3Region:       cfg.S3.Region,
		S3Bucket:       cfg.S3.Bucket,
		S3AccessKey:    cfg.S3.AccessKey,
		S3SecretKey:    cfg.S3.SecretKey,
		S3UsePathStyle: cfg.S3.UsePathStyle,
	}
	if cfg.Driver != media.DriverLocal {
		return options, nil
	}

	localPath, err := config.ResolveDataPath(cfg.Path, "")
	if err != nil {
		return media.Options{}, err
	}
	options.LocalPath = localPath

	return options, nil
}

// authOptions maps the app settings onto the options of the auth service.
func authOptions(cfg config.AppConfig) service.AuthOptions {
	return service.AuthOptions{
		Password: cfg.Secret,
	}
}

// simulatorOptions maps the timing settings onto the options of the simulator service.
func simulatorOptions(cfg config.TimingConfig) service.SimulatorOptions {
	return service.SimulatorOptions{
		Event:      cfg.SimulateEvent,
		PoolLength: cfg.PoolLength,
		ReplayPath: cfg.ReplayPath,
	}
}

// timingOptions maps the timing settings onto the options of the timing service.
// The status shows only the name of a replayed recording.
func timingOptions(cfg config.TimingConfig) service.TimingOptions {
	replayFile := ""
	if cfg.ReplayPath != "" {
		replayFile = path.Base(cfg.ReplayPath)
	}

	return service.TimingOptions{
		PoolLength:     cfg.PoolLength,
		ConfiguredPort: cfg.SerialPort,
		SimulateEvent:  cfg.SimulateEvent,
		ReplayFile:     replayFile,
	}
}

// recorderOptions maps the timing settings, the directory of the recordings, and the version of the application
// onto the options of the recording writer.
func recorderOptions(cfg config.TimingConfig, dir, version string) recording.WriterOptions {
	source := recording.SourceDevice
	if cfg.SimulateEvent > 0 {
		source = recording.SourceSimulator
	}

	return recording.WriterOptions{
		Dir: dir,
		Session: recording.Session{
			Version:  version,
			Source:   source,
			Port:     cfg.SerialPort,
			BaudRate: cfg.BaudRate,
		},
	}
}

// recordingOptions maps the timing settings onto the options of the recording service.
func recordingOptions(cfg config.TimingConfig) service.RecordingOptions {
	return service.RecordingOptions{
		ReplayPath: cfg.ReplayPath,
	}
}

// apiOptions maps the configuration onto the options of the API.
func apiOptions(cfg config.Config) api.Options {
	return api.Options{
		Name:     cfg.App.Name,
		Language: string(cfg.App.Language),
		Theme: api.ThemeSettings{
			Primary:   cfg.Theme.Primary,
			Secondary: cfg.Theme.Secondary,
			Success:   cfg.Theme.Success,
			Info:      cfg.Theme.Info,
			Warning:   cfg.Theme.Warning,
			Error:     cfg.Theme.Error,
			Neutral:   cfg.Theme.Neutral,
		},
		TrustedProxies:       cfg.Server.TrustedProxies,
		CORSAllowedOrigins:   cfg.Server.CORSAllowedOrigins,
		CORSAllowCredentials: cfg.Server.CORSAllowCredentials,
		Docs:                 true,
	}
}

// webhookOptions maps the configuration and the version of the application onto the options of the webhook client.
func webhookOptions(cfg config.Config, version string) webhook.Options {
	return webhook.Options{
		UserAgent:       config.AppName + "/" + version,
		AllowedNetworks: cfg.Webhook.AllowedNetworks,
	}
}
