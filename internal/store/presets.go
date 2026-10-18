package store

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

const (
	// presetsFolder is the storage folder the presets are kept in.
	presetsFolder = "presets"
	// presetExtension is the extension of a preset file.
	presetExtension = ".json"
)

// PresetStore keeps every preset as a JSON file in the storage, named after the preset.
type PresetStore struct {
	storage media.Storage

	mu sync.Mutex
}

// NewPresetStore creates a PresetStore.
func NewPresetStore(storage media.Storage) *PresetStore {
	return &PresetStore{
		storage: storage,
	}
}

// List returns the names of all presets, sorted by name.
func (s *PresetStore) List(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths, err := s.storage.List(ctx, presetsFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing presets: %w", err)
	}

	names := make([]string, 0, len(paths))
	for _, p := range paths {
		name, ok := fileName(presetsFolder, p, presetExtension)
		if !ok {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)

	return names, nil
}

// Get returns the preset with the given name, or apperr.ErrNotFound if it does not exist.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Get(ctx context.Context, name string) (*domain.Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(presetsFolder, name, presetExtension)
	if err != nil {
		return nil, err
	}

	data, err := media.ReadAll(ctx, s.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading preset %q: %w", name, err)
	}

	var preset domain.Preset
	if err := json.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("store: decoding preset %q: %w", name, err)
	}

	return &preset, nil
}

// Save writes the preset under the given name, replacing an existing one.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Save(ctx context.Context, name string, preset domain.Preset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(presetsFolder, name, presetExtension)
	if err != nil {
		return err
	}

	// Sorted lane keys keep the file stable between saves
	data, err := json.Marshal(preset, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return fmt.Errorf("store: encoding preset %q: %w", name, err)
	}

	if err := s.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving preset %q: %w", name, err)
	}

	return nil
}

// Delete removes the preset with the given name, or returns apperr.ErrNotFound if it does not exist.
// A name that cannot address a preset file fails with ErrInvalidName.
func (s *PresetStore) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(presetsFolder, name, presetExtension)
	if err != nil {
		return err
	}

	exists, err := s.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking preset %q: %w", name, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting preset %q: %w", name, err)
	}

	return nil
}
