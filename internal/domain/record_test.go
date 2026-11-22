package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCourseOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		poolLength int
		want       Course
		wantOK     bool
	}{
		{
			name:       "returns the short course of a 25 m pool",
			poolLength: 25,
			want:       CourseShort,
			wantOK:     true,
		},
		{
			name:       "returns the long course of a 50 m pool",
			poolLength: 50,
			want:       CourseLong,
			wantOK:     true,
		},
		{
			name:       "rejects a pool of another length",
			poolLength: 33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			course, ok := CourseOf(tt.poolLength)

			assert.Equal(t, tt.want, course)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestRecord_Discipline(t *testing.T) {
	t.Parallel()

	t.Run("returns the discipline with uppercase abbreviations", func(t *testing.T) {
		t.Parallel()

		record := &Record{
			Course:     CourseLong,
			Gender:     "w",
			RelayCount: 4,
			Distance:   400,
			Stroke:     "fr",
		}

		discipline := record.Discipline()

		assert.Equal(t, RecordDiscipline{
			Course:     CourseLong,
			Gender:     "W",
			RelayCount: 4,
			Distance:   400,
			Stroke:     "FR",
		}, discipline)
	})

	t.Run("counts a record without a relay count as individual", func(t *testing.T) {
		t.Parallel()

		record := &Record{}

		discipline := record.Discipline()

		assert.Equal(t, 1, discipline.RelayCount)
	})
}

func TestRecord_SameRange(t *testing.T) {
	t.Parallel()

	t.Run("compares both bounds", func(t *testing.T) {
		t.Parallel()

		record := &Record{
			AgeFrom: 10,
			AgeTo:   12,
		}

		assert.True(t, record.SameRange(&Record{
			AgeFrom: 10,
			AgeTo:   12,
		}))
		assert.False(t, record.SameRange(&Record{
			AgeFrom: 10,
		}))
	})
}

func TestRecord_Covers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ageFrom int
		ageTo   int
		age     int
		want    bool
	}{
		{
			name: "covers every age without bounds",
			age:  14,
			want: true,
		},
		{
			name:    "covers an age inside the bounds",
			ageFrom: 12,
			ageTo:   14,
			age:     14,
			want:    true,
		},
		{
			name:    "covers an age above an open upper bound",
			ageFrom: 20,
			age:     65,
			want:    true,
		},
		{
			name:  "covers an age below an open lower bound",
			ageTo: 10,
			age:   8,
			want:  true,
		},
		{
			name:    "rejects an age outside the bounds",
			ageFrom: 12,
			ageTo:   14,
			age:     15,
		},
		{
			name: "covers an unknown age without bounds",
			want: true,
		},
		{
			name:    "rejects an unknown age with bounds",
			ageFrom: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := &Record{
				AgeFrom: tt.ageFrom,
				AgeTo:   tt.ageTo,
			}

			assert.Equal(t, tt.want, record.Covers(tt.age))
		})
	}
}
