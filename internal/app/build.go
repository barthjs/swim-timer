package app

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/cmdline"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/update"
)

// Stores holds the stores of the application, all reading from the same file storage.
type Stores struct {
	Meet      *store.MeetStore
	Preset    *store.PresetStore
	Recording *store.RecordingStore
}

// NewStores builds the stores on storage and loads the meet data. Meet data that fails to load
// leaves the meet store empty, so the application still starts and the files can be replaced.
func NewStores(ctx context.Context, logger zerolog.Logger, storage media.Storage) Stores {
	meetStore := store.NewMeetStore(storage)
	if err := meetStore.LoadAll(ctx); err != nil {
		logger.Warn().Err(err).Msg("failed to load meet data")
	}
	return Stores{
		Meet:      meetStore,
		Preset:    store.NewPresetStore(storage),
		Recording: store.NewRecordingStore(storage),
	}
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
