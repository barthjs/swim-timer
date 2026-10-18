package domain

import (
	"path/filepath"
	"slices"
	"strings"
)

// MeetFile identifies one of the Swiss Timing export files the meet data is read from.
type MeetFile string

const (
	MeetFileCategories   MeetFile = "LSTCAT"
	MeetFileCompetitors  MeetFile = "LSTCONC"
	MeetFileDistances    MeetFile = "LSTLONG"
	MeetFileEvents       MeetFile = "LSTRACE"
	MeetFilePhases       MeetFile = "LSTROUND"
	MeetFileStartEntries MeetFile = "LSTSTART"
	MeetFileStyles       MeetFile = "LSTSTYLE"
)

// meetFiles lists every meet file in the order they are loaded.
var meetFiles = []MeetFile{
	MeetFileCategories,
	MeetFileCompetitors,
	MeetFileDistances,
	MeetFileEvents,
	MeetFilePhases,
	MeetFileStartEntries,
	MeetFileStyles,
}

// MeetFiles returns every meet file in the order they are loaded.
// The slice is a copy, so callers cannot change the list.
func MeetFiles() []MeetFile {
	return slices.Clone(meetFiles)
}

// ParseMeetFile reports the meet file with the exact given identifier.
func ParseMeetFile(id string) (MeetFile, bool) {
	for _, file := range meetFiles {
		if string(file) == id {
			return file, true
		}
	}
	return "", false
}

// MatchMeetFile reports the meet file an uploaded filename belongs to, ignoring its extension and letter case.
func MatchMeetFile(filename string) (MeetFile, bool) {
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	return ParseMeetFile(strings.ToUpper(baseName))
}
