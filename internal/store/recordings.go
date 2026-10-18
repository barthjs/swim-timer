package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

const (
	// RecordingsFolder is the storage folder the recordings are kept in.
	// The timing service writes into it directly on the local driver, since a recording grows while it runs.
	RecordingsFolder = "recordings"
	// recordingExtension is the extension of a recording file, which is part of the name of the recording.
	recordingExtension = ".jsonl"
)

// RecordingStore keeps the recordings of the timing data as JSON Lines files in the storage.
type RecordingStore struct {
	storage media.Storage

	mu sync.Mutex
}

// NewRecordingStore creates a RecordingStore.
func NewRecordingStore(storage media.Storage) *RecordingStore {
	return &RecordingStore{
		storage: storage,
	}
}

// List returns all recordings, sorted by name.
func (s *RecordingStore) List(ctx context.Context) ([]domain.Recording, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths, err := s.storage.List(ctx, RecordingsFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing recordings: %w", err)
	}

	recordings := make([]domain.Recording, 0, len(paths))
	for _, p := range paths {
		name, ok := fileName(RecordingsFolder, p, recordingExtension)
		if !ok {
			continue
		}
		recordings = append(recordings, domain.Recording{
			Name: name + recordingExtension,
			Path: p,
		})
	}
	slices.SortFunc(recordings, func(a, b domain.Recording) int { return strings.Compare(a.Name, b.Name) })

	return recordings, nil
}

// Get opens the recording at a storage path for reading, or returns apperr.ErrNotFound if it does not exist.
// It takes the path, since the configuration names the replayed recording by its path. The caller must close it.
func (s *RecordingStore) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	reader, err := s.storage.Get(ctx, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading recording %q: %w", p, err)
	}

	return reader, nil
}

// Create writes data as the recording with the given name, replacing an existing one.
// A name that cannot address a recording file fails with ErrInvalidName.
func (s *RecordingStore) Create(ctx context.Context, name string, data []byte) (domain.Recording, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.Path(name)
	if err != nil {
		return domain.Recording{}, err
	}

	if err := s.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return domain.Recording{}, fmt.Errorf("store: saving recording %q: %w", name, err)
	}

	return domain.Recording{
		Name: name,
		Path: p,
	}, nil
}

// Delete removes the recording with the given name, or returns apperr.ErrNotFound if it does not exist.
// A name that cannot address a recording file fails with ErrInvalidName.
func (s *RecordingStore) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.Path(name)
	if err != nil {
		return err
	}

	exists, err := s.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking recording %q: %w", name, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting recording %q: %w", name, err)
	}

	return nil
}

// Path returns the storage path of the recording with the given name, or ErrInvalidName for a name
// without the recording extension or one that cannot address a recording file.
func (s *RecordingStore) Path(name string) (string, error) {
	base, found := strings.CutSuffix(name, recordingExtension)
	if !found {
		return "", ErrInvalidName
	}
	return filePath(RecordingsFolder, base, recordingExtension)
}
