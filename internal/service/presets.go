package service

import (
	"context"
	"errors"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

// PresetService manages the presets of the AR overlay.
type PresetService struct {
	store *store.PresetStore
}

// NewPresetService creates a PresetService.
func NewPresetService(store *store.PresetStore) *PresetService {
	return &PresetService{
		store: store,
	}
}

// List returns the names of all presets.
func (s *PresetService) List(ctx context.Context) ([]string, error) {
	return s.store.List(ctx)
}

// Get returns the preset with the given name, or apperr.ErrNotFound if it does not exist
// or name cannot address a preset file.
func (s *PresetService) Get(ctx context.Context, name string) (*domain.Preset, error) {
	preset, err := s.store.Get(ctx, name)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	return preset, err
}

// Save creates or replaces the preset with the given name, which has to match the preset's own name,
// so a stored preset always carries its file name.
func (s *PresetService) Save(ctx context.Context, name string, preset domain.Preset) error {
	if preset.Name != name {
		return apperr.NewValidationError("name", "invalid")
	}

	err := s.store.Save(ctx, name, preset)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.NewValidationError("name", "invalid")
	}
	return err
}

// Delete removes the preset with the given name, or returns apperr.ErrNotFound if it does not exist
// or name cannot address a preset file.
func (s *PresetService) Delete(ctx context.Context, name string) error {
	err := s.store.Delete(ctx, name)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.ErrNotFound
	}
	return err
}
