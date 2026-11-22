package app

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/store"
)

// Stores holds the stores of the application, all reading from the same file storage.
type Stores struct {
	Meet        *store.MeetStore
	Preset      *store.PresetStore
	RecordBreak *store.RecordBreakStore
	RecordTable *store.RecordTableStore
	Recording   *store.RecordingStore
	Webhook     *store.WebhookStore
}

// NewStores builds the stores on storage and loads their data, leaving a store empty whose data fails to load,
// so the application still starts and the files can be replaced.
func NewStores(ctx context.Context, logger zerolog.Logger, storage media.Storage) Stores {
	meetStore := store.NewMeetStore(storage)
	if err := meetStore.LoadAll(ctx); err != nil {
		logger.Warn().Err(err).Msg("failed to load meet data")
	}
	recordTableStore := store.NewRecordTableStore(storage)
	if err := recordTableStore.LoadAll(ctx); err != nil {
		logger.Warn().Err(err).Msg("failed to load record tables")
	}
	return Stores{
		Meet:        meetStore,
		Preset:      store.NewPresetStore(storage),
		RecordBreak: store.NewRecordBreakStore(storage),
		RecordTable: recordTableStore,
		Recording:   store.NewRecordingStore(storage),
		Webhook:     store.NewWebhookStore(storage),
	}
}
