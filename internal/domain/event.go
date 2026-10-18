package domain

import (
	"strings"
	"time"
)

// unscheduledTime is the time the timing software exports for an event without a scheduled time.
const unscheduledTime = "00:00"

// Event represents an event of the meet (from LSTRACE.TXT), a group of heats over one distance and style,
// for example 200 m breaststroke male. The timing software numbers every phase of an event,
// such as heats and final, as its own event.
type Event struct {
	Number     int       `csv:"event"` // event number, the key the device transmits
	PhaseID    int       `csv:"round"` // see Phase.ID
	Phase      *Phase    `csv:"-"`
	HeatCount  int       `csv:"nbHeat"` // number of heats of the event
	DistanceID int       `csv:"idLen"`  // see Distance.ID
	Distance   *Distance `csv:"-"`
	StyleID    int       `csv:"idStyle"` // see Style.ID
	Style      *Style    `csv:"-"`
	AbCat      string    `csv:"abCat"` // see Category.Abbreviation
	Category   *Category `csv:"-"`
	Date       string    `csv:"date"` // scheduled date as exported in yy/mm/dd, e.g. "26/05/01" for 1 May 2026
	Time       string    `csv:"time"` // scheduled start time as exported, e.g. "16:00", or "00:00" without one
	Order      int       `csv:"-"`    // position in the file, which lists the events in the order of the program
}

// DisplayName combines distance, style, and category into the name the overlays and the meet overview show,
// for example "200 m Medley Women". Parts whose reference is missing are left out.
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

// PhaseName returns the name of the phase of the event, such as "Final",
// or an empty string if the phases are not loaded or the phase is unknown.
func (e *Event) PhaseName() string {
	if e.Phase == nil {
		return ""
	}
	return strings.TrimSpace(e.Phase.Name)
}

// ISODate returns the scheduled date as YYYY-MM-DD. The timing software exports dates as yy/mm/dd.
// A value in another format is returned trimmed but otherwise unchanged, so nothing is lost.
func (e *Event) ISODate() string {
	return ParseExportDate(e.Date)
}

// TrimmedTime returns the scheduled start time without the padding of the export, e.g. "16:00",
// or an empty string for an event without a scheduled time. The timing software exports such an event,
// for example a final added during the meet, with the time "00:00".
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
