package domain

import (
	"strings"
	"time"
)

// unscheduledTime is the time the timing software exports for an event without a scheduled time.
const unscheduledTime = "00:00"

// Event represents an event of the meet from LSTRACE.TXT, a group of heats over one distance and style,
// where the timing software numbers every phase as its own event.
type Event struct {
	Number     int // event number, the key the device transmits
	PhaseID    int // see Phase.ID
	Phase      *Phase
	HeatCount  int // number of heats of the event
	DistanceID int // see Distance.ID
	Distance   *Distance
	StyleID    int // see Style.ID
	Style      *Style
	AbCat      string // see Category.Abbreviation
	Category   *Category
	Date       string // scheduled date as exported in yy/mm/dd, e.g. "26/05/01" for 1 May 2026
	Time       string // scheduled start time as exported, e.g. "16:00", or "00:00" without one
	Order      int    // position in the file, which lists the events in the order of the program
}

// DisplayName combines the distance, style, and category that are known into the name of the event.
func (e *Event) DisplayName() string {
	parts := make([]string, 0, 3)
	if e.Distance != nil && strings.TrimSpace(e.Distance.Name) != "" {
		parts = append(parts, strings.TrimSpace(e.Distance.Name))
	}
	if e.Style != nil && strings.TrimSpace(e.Style.Name) != "" {
		parts = append(parts, strings.TrimSpace(e.Style.Name))
	}
	if e.Category != nil && strings.TrimSpace(e.Category.Name) != "" {
		parts = append(parts, strings.TrimSpace(e.Category.Name))
	}
	return strings.Join(parts, " ")
}

// PhaseName returns the name of the phase of the event, or an empty string while it is unknown.
func (e *Event) PhaseName() string {
	if e.Phase == nil {
		return ""
	}
	return strings.TrimSpace(e.Phase.Name)
}

// ISODate returns the scheduled date of the export as YYYY-MM-DD, or another format trimmed but unchanged.
func (e *Event) ISODate() string {
	return ParseExportDate(e.Date)
}

// TrimmedTime returns the scheduled start time without the padding of the export,
// or an empty string for the time "00:00" of an event without one.
func (e *Event) TrimmedTime() string {
	trimmed := strings.TrimSpace(e.Time)
	if trimmed == unscheduledTime {
		return ""
	}
	return trimmed
}

// ParseExportDate converts a date exported as yy/mm/dd into YYYY-MM-DD,
// or returns the trimmed value if it does not have that format.
func ParseExportDate(value string) string {
	trimmed := strings.TrimSpace(value)
	parsed, err := time.Parse("06/01/02", trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.Format(time.DateOnly)
}
