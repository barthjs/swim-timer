package store

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path"
	"slices"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

// recordBreakFile is the file that keeps the records set at the meet.
const recordBreakFile = recordsFolder + "/breaks.json"

// RecordBreakStore keeps the records set at the meet in one JSON file in the storage.
// It holds them in memory, as read by LoadAll or on first use, and writes the file with every change,
// except for ReplaceHeat, whose changes Flush writes, so checking a heat never waits for the storage.
type RecordBreakStore struct {
	storage media.Storage

	// writeMu serializes the writes of the file, so an older state never replaces a newer one. It is taken before mu.
	writeMu sync.Mutex
	mu      sync.Mutex
	// breaks holds the stored record breaks once loaded is set.
	breaks []domain.RecordBreak
	loaded bool
	// version counts the changes in memory, written the ones in the file.
	version uint64
	written uint64
}

// NewRecordBreakStore creates a RecordBreakStore.
func NewRecordBreakStore(storage media.Storage) *RecordBreakStore {
	return &RecordBreakStore{
		storage: storage,
	}
}

// List returns every record break, the most recent first.
func (s *RecordBreakStore) List(ctx context.Context) ([]domain.RecordBreak, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	breaks, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(breaks, func(a, b domain.RecordBreak) int {
		return cmp.Or(b.BrokenAt.Compare(a.BrokenAt), cmp.Compare(a.Lane, b.Lane))
	})
	return breaks, nil
}

// Delete removes the record break with the given id, or returns apperr.ErrNotFound if it does not exist.
func (s *RecordBreakStore) Delete(ctx context.Context, id string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	breaks, err := s.read(ctx)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(breaks, func(b domain.RecordBreak) bool { return b.ID == id })
	if index < 0 {
		return apperr.ErrNotFound
	}
	return s.write(ctx, slices.Delete(breaks, index, index+1))
}

// Clear removes every record break.
func (s *RecordBreakStore) Clear(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.write(ctx, nil)
}

// LoadAll reads the stored record breaks again and keeps the previous ones when the file fails with a DataFileError.
func (s *RecordBreakStore) LoadAll(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	breaks, err := s.readFile(ctx)
	if err != nil {
		return err
	}
	s.breaks, s.loaded = breaks, true
	s.version++
	s.written = s.version
	return nil
}

// ReplaceHeat drops the record breaks of a heat in memory and keeps the ones replace returns for it instead,
// and reports whether they differ from the previous ones, which Flush then writes.
// replace receives the record breaks the heat had before and those of every other heat,
// so it checks a heat against them under the same lock,
// and a heat that is made official again replaces its own records instead of adding to them.
func (s *RecordBreakStore) ReplaceHeat(
	ctx context.Context,
	event, heat int,
	replace func(previous, others []domain.RecordBreak) []domain.RecordBreak,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	breaks, err := s.read(ctx)
	if err != nil {
		return false, err
	}
	var previous, others []domain.RecordBreak
	for _, b := range breaks {
		if b.Event == event && b.Heat == heat {
			previous = append(previous, b)
		} else {
			others = append(others, b)
		}
	}
	added := replace(slices.Clone(previous), slices.Clone(others))
	if slices.Equal(added, previous) {
		return false, nil
	}
	s.breaks = append(others, added...)
	s.version++
	return true, nil
}

// Flush writes the record breaks in memory if ReplaceHeat changed them since the last write.
// Concurrent calls write one after the other, and a call that finds a newer state written does nothing.
func (s *RecordBreakStore) Flush(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.Lock()
	breaks, version := slices.Clone(s.breaks), s.version
	s.mu.Unlock()
	if version == s.written {
		return nil
	}

	if err := s.writeFile(ctx, breaks); err != nil {
		return err
	}
	s.written = version
	return nil
}

// read returns a copy of the record breaks in memory while the caller holds the lock, read from the file on first use.
func (s *RecordBreakStore) read(ctx context.Context) ([]domain.RecordBreak, error) {
	if !s.loaded {
		breaks, err := s.readFile(ctx)
		if err != nil {
			return nil, err
		}
		s.breaks, s.loaded = breaks, true
	}
	return slices.Clone(s.breaks), nil
}

// readFile returns the record breaks of the file, none if it does not exist.
func (s *RecordBreakStore) readFile(ctx context.Context) ([]domain.RecordBreak, error) {
	content, err := media.ReadAll(ctx, s.storage, recordBreakFile)
	if errors.Is(err, media.ErrNotFound) {
		return []domain.RecordBreak{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: reading record breaks: %w", err)
	}

	var breaks []domain.RecordBreak
	if err := json.Unmarshal(content, &breaks); err != nil {
		return nil, &DataFileError{
			File: path.Base(recordBreakFile),
			Err:  err,
		}
	}
	return breaks, nil
}

// write replaces the stored record breaks while the caller holds both locks.
func (s *RecordBreakStore) write(ctx context.Context, breaks []domain.RecordBreak) error {
	if err := s.writeFile(ctx, breaks); err != nil {
		return err
	}
	s.breaks, s.loaded = slices.Clone(breaks), true
	s.version++
	s.written = s.version
	return nil
}

// writeFile writes the record breaks to the file while the caller holds writeMu.
func (s *RecordBreakStore) writeFile(ctx context.Context, breaks []domain.RecordBreak) error {
	if breaks == nil {
		breaks = []domain.RecordBreak{}
	}
	content, err := json.Marshal(breaks, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("store: encoding record breaks: %w", err)
	}
	if err := s.storage.Put(ctx, recordBreakFile, bytes.NewReader(content), int64(len(content))); err != nil {
		return fmt.Errorf("store: saving record breaks: %w", err)
	}
	return nil
}
