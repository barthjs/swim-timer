package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMeetFile(t *testing.T) {
	t.Parallel()

	t.Run("returns the meet file", func(t *testing.T) {
		t.Parallel()

		got, ok := ParseMeetFile("LSTCAT")

		assert.True(t, ok)
		assert.Equal(t, MeetFileCategories, got)
	})

	t.Run("rejects a lowercase identifier", func(t *testing.T) {
		t.Parallel()

		_, ok := ParseMeetFile("lstconc")

		assert.False(t, ok)
	})
}

func TestMatchMeetFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     MeetFile
		wantOK   bool
	}{
		{
			name:     "returns the competitors for an upper case name with extension",
			filename: "LSTCONC.TXT",
			want:     MeetFileCompetitors,
			wantOK:   true,
		},
		{
			name:     "returns the start entries for a lower case name",
			filename: "lststart.csv",
			want:     MeetFileStartEntries,
			wantOK:   true,
		},
		{
			name:     "returns the events for a mixed case name",
			filename: "LstRace.Txt",
			want:     MeetFileEvents,
			wantOK:   true,
		},
		{
			name:     "returns the styles for a name without extension",
			filename: "LSTSTYLE",
			want:     MeetFileStyles,
			wantOK:   true,
		},
		{
			name:     "rejects an unknown file",
			filename: "LSTREC.TXT",
			wantOK:   false,
		},
		{
			name:     "rejects a name with a further part",
			filename: "LSTCONC.backup.TXT",
			wantOK:   false,
		},
		{
			name:     "rejects an empty name",
			filename: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := MatchMeetFile(tt.filename)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
