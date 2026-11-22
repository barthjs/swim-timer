package domain

import (
	"strings"
	"time"
)

// Course is the pool length a record is set in.
type Course string

const (
	// CourseShort is a 25 m pool.
	CourseShort Course = "SCM"
	// CourseLong is a 50 m pool.
	CourseLong Course = "LCM"
)

// CourseOf returns the course of a pool length in meters, false for a pool of another length.
func CourseOf(poolLength int) (Course, bool) {
	switch poolLength {
	case 25:
		return CourseShort, true
	case 50:
		return CourseLong, true
	default:
		return "", false
	}
}

// RecordTable is a named list of records, stored as JSON, whose enabled tables are checked in their order.
type RecordTable struct {
	ID string `json:"id"`
	// Name is shown in the interface, Abbreviation in the overlays.
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Enabled      bool   `json:"enabled"`
}

// Record is the fastest time of a record table for a discipline and an age range.
// It carries JSON tags because a RecordBreak persists it as JSON.
type Record struct {
	Course Course `json:"course"`
	// Gender is the abbreviation of the category, as the meet data names it.
	Gender string `json:"gender"`
	// RelayCount is the number of swimmers per team, 1 for an individual event.
	RelayCount int `json:"relayCount"`
	// Distance is the distance each swimmer swims in meters, a relay leg for a relay.
	Distance int `json:"distance"`
	// Stroke is the abbreviation of the style, as the meet data names it.
	Stroke string `json:"stroke"`
	// AgeFrom and AgeTo bound the age the record applies to, 0 for an open bound.
	AgeFrom int `json:"ageFrom"`
	AgeTo   int `json:"ageTo"`
	// Time is the record time in hundredths of a second.
	Time int `json:"time"`
	// Approved reports whether the record is recognized, a record of a table counts either way.
	Approved bool   `json:"approved"`
	Name     string `json:"name"`
	Club     string `json:"club"`
	Place    string `json:"place"`
	// Date is the day the record was set as YYYY-MM-DD, empty if unknown.
	Date string `json:"date"`
	// BirthYear is the year of birth of the record holder, 0 if unknown.
	BirthYear int `json:"birthYear"`
}

// RecordDiscipline is what a record is set in, so records of the same discipline compete with each other.
type RecordDiscipline struct {
	Course     Course
	Gender     string
	RelayCount int
	Distance   int
	Stroke     string
}

// Discipline returns the discipline of the record, with the abbreviations in uppercase.
func (r *Record) Discipline() RecordDiscipline {
	return RecordDiscipline{
		Course:     r.Course,
		Gender:     strings.ToUpper(r.Gender),
		RelayCount: max(r.RelayCount, 1),
		Distance:   r.Distance,
		Stroke:     strings.ToUpper(r.Stroke),
	}
}

// SameRange reports whether two records apply to the same ages.
func (r *Record) SameRange(other *Record) bool {
	return r.AgeFrom == other.AgeFrom && r.AgeTo == other.AgeTo
}

// Covers reports whether the record applies to a competitor of the given age. An unknown age, 0,
// is only covered by a record without age bounds.
func (r *Record) Covers(age int) bool {
	if age <= 0 {
		return r.AgeFrom == 0 && r.AgeTo == 0
	}
	return (r.AgeFrom == 0 || age >= r.AgeFrom) && (r.AgeTo == 0 || age <= r.AgeTo)
}

// RecordBreak is a record set or equalled at the meet, stored as JSON, which the next heats are checked against.
type RecordBreak struct {
	ID string `json:"id"`
	// TableID names the record table the record belongs to, with its name and abbreviation at the time of the break.
	TableID      string `json:"tableId"`
	TableName    string `json:"tableName"`
	Abbreviation string `json:"abbreviation"`
	Event        int    `json:"event"`
	Heat         int    `json:"heat"`
	Lane         int    `json:"lane"`
	// Equalled reports a time equal to the record, which the competitor then holds together with its holder.
	Equalled bool `json:"equalled"`
	// Record is the new record, Previous the record it broke or equalled.
	Record   Record    `json:"record"`
	Previous Record    `json:"previous"`
	BrokenAt time.Time `json:"brokenAt"`
}

// LaneRecord is a record a lane of the live heat broke or equalled, as the overlays show it.
// It carries JSON tags because it is sent as part of the live heat.
type LaneRecord struct {
	Abbreviation string `json:"abbreviation"`
	TableName    string `json:"tableName"`
	// AgeFrom and AgeTo bound the age the record applies to, 0 for an open bound.
	AgeFrom  int  `json:"ageFrom"`
	AgeTo    int  `json:"ageTo"`
	Equalled bool `json:"equalled"`
	// PreviousTime is the time of the record before in milliseconds, like the times of the lanes, and PreviousName its holder.
	PreviousTime int    `json:"previousTime"`
	PreviousName string `json:"previousName"`
}
