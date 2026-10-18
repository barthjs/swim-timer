package domain

// The phases in the catalog of the timing software, see Event.PhaseID.
const (
	// PhaseTimedFinal ranks the competitors of all heats by their times, without a final.
	PhaseTimedFinal = 0
	// PhaseHeats qualifies the fastest competitors for a semifinal or final.
	PhaseHeats = 1
	// PhaseSemifinal qualifies the fastest competitors for the final.
	PhaseSemifinal = 2
	// PhaseFinal decides the ranking.
	PhaseFinal = 3
	// PhaseSwimOff decides between competitors with equal times for a place in the next phase.
	PhaseSwimOff = 4
	// PhaseReswim repeats a heat, for example after a technical failure.
	PhaseReswim = 5
)

// Phase represents a phase of an event (from LSTROUND.TXT), such as heats or final.
// The file is a fixed catalog of the timing software, the same for every meet.
type Phase struct {
	ID           int    `csv:"idRound"`    // see Event.PhaseID
	Name         string `csv:"TITLE"`      // display name, e.g. "Final"
	Abbreviation string `csv:"RoundAbrev"` // abbreviation, e.g. "FIN"
}
