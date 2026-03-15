package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/barthjs/swim-timer/internal/data"
)

// PresetsStore is a simple file-based store for presets.
// Each preset is stored as a JSON file in the specified directory.
type PresetsStore struct {
	mu  sync.Mutex
	dir string
}

const presetExtension = ".json"

var ErrInvalidPreset = errors.New("invalid preset")

// NewPresetsStore creates a new PresetsStore.
func NewPresetsStore(dir string) *PresetsStore {
	return &PresetsStore{dir: dir}
}

// List returns a list of all available presets.
func (s *PresetsStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	presets := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == presetExtension {
			presets = append(presets, strings.TrimSuffix(entry.Name(), presetExtension))
		}
	}

	return presets, nil
}

// Get returns the contents of a preset.
func (s *PresetsStore) Get(name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.securePath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path)
}

// Save saves a preset to the store.
func (s *PresetsStore) Save(name string, data data.Preset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	path, err := s.securePath(name)
	if err != nil {
		return err
	}

	marshaled, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, marshaled, 0644)
}

// Delete removes a preset from the store.
func (s *PresetsStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.securePath(name)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

// securePath returns the absolute path to a preset file, ensuring it is within the store directory.
func (s *PresetsStore) securePath(name string) (string, error) {
	cleanName := filepath.Clean(name)
	absBase, err := filepath.Abs(s.dir)
	if err != nil {
		return "", err
	}

	target := filepath.Join(s.dir, cleanName+presetExtension)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", ErrInvalidPreset
	}

	return absTarget, nil
}
