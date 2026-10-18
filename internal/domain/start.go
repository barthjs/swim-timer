package domain

// StartEntry represents a single entry in the start list (from LSTSTART.TXT).
// On long distances, two entries can share a lane of the same heat.
// Such heats are timed by hand, and the lane shows the entry read last.
type StartEntry struct {
	Event        int         `csv:"event"`  // see Event.Number
	PhaseID      int         `csv:"round"`  // see Event.PhaseID
	Heat         int         `csv:"heat"`   // heat number counted from 0, while the device counts from 1, -1 if withdrawn
	Lane         int         `csv:"lane"`   // lane number
	Relais       int         `csv:"relais"` // relay position, 0 in every individual event of the exported data
	CompetitorID int         `csv:"idBib"`  // see Competitor.ID, despite the column name not the start number
	Competitor   *Competitor `csv:"-"`
}

// withdrawnHeat is the heat the timing software moves a start entry to when the competitor withdraws.
const withdrawnHeat = -1

// Withdrawn reports whether the competitor withdrew from the event.
// The timing software keeps the entry in the start list, but moves it out of every heat.
func (s *StartEntry) Withdrawn() bool {
	return s.Heat == withdrawnHeat
}
