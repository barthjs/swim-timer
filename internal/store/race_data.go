package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/barthjs/swim-timer/internal/data"
)

const (
	IDCategories = "LSTCAT"
	IDPersons    = "LSTCONC"
	IDDistances  = "LSTLONG"
	IDRaces      = "LSTRACE"
	IDStarts     = "LSTSTART"
	IDStyles     = "LSTSTYLE"
)

var AllowedRaceFiles = []string{
	IDCategories,
	IDPersons,
	IDDistances,
	IDRaces,
	IDStarts,
	IDStyles,
}

// RaceDataStore stores race data files. It provides thread-safe access
// to the data and allows loading from and saving to CSV files.
type RaceDataStore struct {
	mu         sync.RWMutex
	dir        string
	categories map[string]*data.Category // by Abbreviation
	distances  map[int]*data.Distance    // by ID
	persons    map[int]*data.Person      // by ID
	races      map[int]*data.Race        // by Event
	starts     []*data.StartEntry
	startsIdx  map[string][]*data.StartEntry // by "event-heat"
	styles     map[int]*data.Style           // by ID
}

// NewRaceDataStore creates a new RaceDataStore instance.
func NewRaceDataStore(dir string) *RaceDataStore {
	return &RaceDataStore{
		dir:        dir,
		categories: make(map[string]*data.Category),
		distances:  make(map[int]*data.Distance),
		persons:    make(map[int]*data.Person),
		races:      make(map[int]*data.Race),
		starts:     make([]*data.StartEntry, 0),
		startsIdx:  make(map[string][]*data.StartEntry),
		styles:     make(map[int]*data.Style),
	}
}

// LoadAll loads all relevant race data files from the csv files.
// It skips missing files and returns an error only if a file exists but fails to parse.
func (s *RaceDataStore) LoadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range AllowedRaceFiles {
		path := filepath.Join(s.dir, id)
		if !fileExists(path) {
			continue
		}

		var err error
		switch id {
		case IDCategories:
			err = s.loadCategories(path)
		case IDDistances:
			err = s.loadDistances(path)
		case IDPersons:
			err = s.loadPersons(path)
		case IDRaces:
			err = s.loadRaces(path)
		case IDStarts:
			err = s.loadStarts(path)
		case IDStyles:
			err = s.loadStyles(path)
		}

		if err != nil {
			return err
		}
	}

	for _, r := range s.races {
		r.Distance = s.distances[r.DistanceID]
		r.Style = s.styles[r.StyleID]
		r.Category = s.categories[r.AbCat]
	}

	for _, start := range s.starts {
		start.Person = s.persons[start.IDBIB]
	}

	s.buildStartsIndex()

	return nil
}

func (s *RaceDataStore) loadCategories(fileName string) error {
	return loadMap(fileName, s.categories, func(c *data.Category) string { return c.Abbreviation })
}

func (s *RaceDataStore) loadDistances(fileName string) error {
	return loadMap(fileName, s.distances, func(d *data.Distance) int { return d.ID })
}

func (s *RaceDataStore) loadPersons(fileName string) error {
	return loadMap(fileName, s.persons, func(p *data.Person) int { return p.BIB })
}

func (s *RaceDataStore) loadRaces(fileName string) error {
	return loadMap(fileName, s.races, func(r *data.Race) int { return r.Event })
}

func (s *RaceDataStore) loadStarts(fileName string) error {
	var items []*data.StartEntry
	if err := readCSV(fileName, &items); err != nil {
		return err
	}
	s.starts = items
	return nil
}

func (s *RaceDataStore) loadStyles(fileName string) error {
	return loadMap(fileName, s.styles, func(st *data.Style) int { return st.ID })
}

// GetCategory returns the category with the given abbreviation if it exists.
func (s *RaceDataStore) GetCategory(abbreviation string) (*data.Category, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.categories[strings.TrimSpace(abbreviation)]
	return c, ok
}

// GetDistance returns the distance with the given ID if it exists.
func (s *RaceDataStore) GetDistance(id int) (*data.Distance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.distances[id]
	return d, ok
}

// GetPerson returns the person with the given ID if it exists.
func (s *RaceDataStore) GetPerson(id int) (*data.Person, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.persons[id]
	return p, ok
}

// GetRace returns the race with the given event number if it exists.
func (s *RaceDataStore) GetRace(event int) (*data.Race, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.races[event]
	return r, ok
}

// GetStarts returns a slice of all start entries.
func (s *RaceDataStore) GetStarts() []*data.StartEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.starts
}

// GetStartsByEventHeat returns the start entries for the given event and heat.
func (s *RaceDataStore) GetStartsByEventHeat(event, heat int) []*data.StartEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startsIdx[startsKey(event, heat)]
}

// GetStyle returns the style with the given ID if it exists.
func (s *RaceDataStore) GetStyle(id int) (*data.Style, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.styles[id]
	return st, ok
}

// GetFileStatuses returns a map of all allowed race data files and their current status.
func (s *RaceDataStore) GetFileStatuses() map[string]bool {
	statuses := make(map[string]bool)
	for _, id := range AllowedRaceFiles {
		statuses[id] = fileExists(filepath.Join(s.dir, id))
	}
	return statuses
}

// IsAllowed checks if the given fileID is one of the allowed race data files.
func (s *RaceDataStore) IsAllowed(fileID string) bool {
	for _, id := range AllowedRaceFiles {
		if id == fileID {
			return true
		}
	}
	return false
}

// SaveFile saves a race data file to the storage directory.
func (s *RaceDataStore) SaveFile(fileID string, r io.Reader) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	dstPath := filepath.Join(s.dir, fileID+".csv")
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	if _, err := io.Copy(dst, r); err != nil {
		return err
	}

	return nil
}

// MatchFileID matches a filename to an allowed file ID.
func (s *RaceDataStore) MatchFileID(filename string) (string, bool) {
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	baseName = strings.ToUpper(baseName)

	for _, id := range AllowedRaceFiles {
		if id == baseName {
			return id, true
		}
	}
	return "", false
}

// buildStartsIndex populates the indexed map for fast event+heat lookups.
func (s *RaceDataStore) buildStartsIndex() {
	clear(s.startsIdx)
	for _, start := range s.starts {
		key := startsKey(start.Event, start.Heat)
		s.startsIdx[key] = append(s.startsIdx[key], start)
	}
}

// startsKey builds the lookup key for the starts index.
func startsKey(event, heat int) string {
	return fmt.Sprintf("%d-%d", event, heat)
}
