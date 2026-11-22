package store

import (
	"context"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

// presetsFolder is the storage folder the presets are kept in.
const presetsFolder = "presets"

// PresetStore keeps every preset as a JSON file in the storage, named after the preset.
type PresetStore struct {
	files *jsonFiles[domain.Preset]
}

// NewPresetStore creates a PresetStore.
func NewPresetStore(storage media.Storage) *PresetStore {
	return &PresetStore{
		files: newJSONFiles[domain.Preset](storage, presetsFolder, "preset"),
	}
}

// List returns the names of all presets, sorted by name.
func (s *PresetStore) List(ctx context.Context) ([]string, error) {
	return s.files.names(ctx)
}

// Get returns the preset with the given name, or apperr.ErrNotFound if it does not exist.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Get(ctx context.Context, name string) (*domain.Preset, error) {
	return s.files.get(ctx, name)
}

// Save writes the preset under the given name, replacing an existing one.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Save(ctx context.Context, name string, preset domain.Preset) error {
	return s.files.save(ctx, name, preset)
}

// Delete removes the preset with the given name, or returns apperr.ErrNotFound if it does not exist.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Delete(ctx context.Context, name string) error {
	return s.files.delete(ctx, name)
}
