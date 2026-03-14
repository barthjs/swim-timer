package data

import "time"

type RaceStatus string

const (
	RaceStatusLoad     RaceStatus = "load"
	RaceStatusStart    RaceStatus = "start"
	RaceStatusOfficial RaceStatus = "official"
)

// Lane holds the data for a single swimming lane
type Lane struct {
	Number    int    `json:"number"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Birthyear string `json:"birthyear"`
	Club      string `json:"club"`
	Lap       int    `json:"lap"`
	Meter     int    `json:"meter"`
	Time      int    `json:"time"`
	Rank      int    `json:"rank"`
	Finished  bool   `json:"finished"`
}

// RaceState holds the current state of the race
type RaceState struct {
	Event     int           `json:"event"`
	Heat      int           `json:"heat"`
	MaxHeat   int           `json:"max_heat"`
	LapCount  int           `json:"lap_count"`
	Name      string        `json:"name"`
	Status    RaceStatus    `json:"status"`
	StartTime *time.Time    `json:"start_time,omitzero"`
	Lanes     map[int]*Lane `json:"lanes"` // Sorted by lane number
}

// Copy returns a deep copy of the RaceState
func (s *RaceState) Copy() *RaceState {
	cp := &RaceState{
		Event:    s.Event,
		Heat:     s.Heat,
		MaxHeat:  s.MaxHeat,
		LapCount: s.LapCount,
		Name:     s.Name,
		Status:   s.Status,
		Lanes:    make(map[int]*Lane, len(s.Lanes)),
	}
	if s.StartTime != nil {
		cp.StartTime = new(time.Time)
		*cp.StartTime = *s.StartTime
	}
	for k, v := range s.Lanes {
		cp.Lanes[k] = new(*v)
	}
	return cp
}
