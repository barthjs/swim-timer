package domain

import "time"

// HeatStatus is the state of the live heat as announced by the timing device.
type HeatStatus string

const (
	HeatStatusLoad     HeatStatus = "load"
	HeatStatusStart    HeatStatus = "start"
	HeatStatusOfficial HeatStatus = "official"
)

// Lane holds the data for a single lane of the live heat.
type Lane struct {
	Number    int    `json:"number"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	BirthYear string `json:"birthYear"`
	Club      string `json:"club"`
	Lap       int    `json:"lap"`
	Meter     int    `json:"meter"`
	Time      int    `json:"time"`
	Rank      int    `json:"rank"`
	Finished  bool   `json:"finished"`
}

// LiveHeat holds the state of the heat the timing device currently reports.
// It is serialized as-is on the websocket and REST path without a DTO mapping,
// because every received timing frame produces a new state that must reach the overlays immediately.
type LiveHeat struct {
	Event     int    `json:"event"`
	Heat      int    `json:"heat"`
	HeatCount int    `json:"heatCount"`
	LapCount  int    `json:"lapCount"`
	Name      string `json:"name"`
	Phase     int    `json:"phase"`
	PhaseName string `json:"phaseName"`
	// FirstLane is the number of the first lane of the pool, 0 or 1, so overlays can map their lanes to it.
	FirstLane int        `json:"firstLane"`
	Status    HeatStatus `json:"status"`
	StartTime *time.Time `json:"startTime,omitzero"`
	// StopTime is the heat time in milliseconds the timer stops at once every lane the device reports in use
	// has finished, the highest finish time among them. It is 0 while a lane in use has not finished.
	StopTime int           `json:"stopTime,omitzero"`
	Lanes    map[int]*Lane `json:"lanes"`
}

// Copy returns a deep copy of the LiveHeat.
func (h *LiveHeat) Copy() *LiveHeat {
	cp := &LiveHeat{
		Event:     h.Event,
		Heat:      h.Heat,
		HeatCount: h.HeatCount,
		LapCount:  h.LapCount,
		Name:      h.Name,
		Phase:     h.Phase,
		PhaseName: h.PhaseName,
		FirstLane: h.FirstLane,
		Status:    h.Status,
		StopTime:  h.StopTime,
		Lanes:     make(map[int]*Lane, len(h.Lanes)),
	}
	if h.StartTime != nil {
		cp.StartTime = new(time.Time)
		*cp.StartTime = *h.StartTime
	}
	for k, v := range h.Lanes {
		cp.Lanes[k] = new(*v)
	}
	return cp
}
