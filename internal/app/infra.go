package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/ratelimiter"
	"github.com/barthjs/swim-timer/internal/recording"
	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/session"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/update"
)

const sessionCookieName = config.AppName + "_session"

// infra holds the infrastructure of the application, taken from Deps or built from the configuration.
type infra struct {
	Storage     media.Storage
	RateLimiter ratelimiter.Limiter
	Session     *scs.SessionManager
	// Recorder writes the recordings, nil if the recording is off.
	Recorder *recording.Writer
	Updater  service.Updater
}

// newInfra builds the infrastructure of the application from what Deps does not pass in
// and registers what it opens with the closers of the application.
func (a *App) newInfra(logger zerolog.Logger, cfg config.Config, deps Deps) (infra, error) {
	in := infra{
		Storage: deps.Storage,
		Updater: deps.Updater,
	}

	var err error
	if in.Storage == nil {
		if in.Storage, err = NewStorage(logger, cfg); err != nil {
			return infra{}, err
		}
	}

	if in.RateLimiter, err = ratelimiter.New(); err != nil {
		return infra{}, fmt.Errorf("initializing rate limiter: %w", err)
	}

	if in.Session, err = session.New(sessionCookieName, cfg.Server.SSLEnabled); err != nil {
		return infra{}, fmt.Errorf("initializing session manager: %w", err)
	}
	sessions := in.Session
	a.closers = append(a.closers, closerFunc(func() error {
		session.StopCleanup(sessions)
		return nil
	}))

	in.Recorder = newRecorder(logger, cfg, deps.Version)

	if in.Updater == nil {
		in.Updater = NewUpdater(deps.Version)
	}

	return in, nil
}

// newRecorder returns the writer of the recordings in the local storage, or nil while nothing is recorded.
func newRecorder(logger zerolog.Logger, cfg config.Config, version string) *recording.Writer {
	// A replay would copy the replayed recording. The s3 driver cannot append to a file a recording grows in,
	// and a storage path that cannot be resolved switches the recording off, since it must not stop the timing.
	if !cfg.Timing.Recording || cfg.Timing.ReplayPath != "" {
		return nil
	}
	if cfg.Storage.Driver != media.DriverLocal {
		logger.Info().Msg("the recording is off, since only the local storage driver can record the timing data")
		return nil
	}

	options, err := StorageOptions(cfg.Storage)
	if err != nil {
		logger.Warn().Err(err).Msg("the recording is off, since the storage path cannot be resolved")
		return nil
	}
	dir := filepath.Join(options.LocalPath, store.RecordingsFolder)
	return recording.NewWriter(logger, recorderOptions(cfg.Timing, dir, version))
}

// NewStorage builds the configured file storage.
func NewStorage(logger zerolog.Logger, cfg config.Config) (media.Storage, error) {
	options, err := StorageOptions(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("resolving the storage path: %w", err)
	}

	storage, err := media.New(options)
	if err != nil {
		return nil, fmt.Errorf("initializing media storage: %w", err)
	}
	if options.Driver == media.DriverLocal {
		logger.Info().Str("path", options.LocalPath).Msg("using local storage")
	}
	return storage, nil
}

// NewUpdater returns the updater for the releases of the application,
// picking the archives of the architecture the binary was built for.
func NewUpdater(version string) *update.Updater {
	return update.New(update.Options{
		Repository: config.AppRepository,
		Binary:     config.AppName,
		Version:    version,
		Arch:       cmdline.ResolveArch(),
		Token:      os.Getenv("GITHUB_TOKEN"),
	})
}
