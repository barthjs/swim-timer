package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestLiveHeat_Copy(t *testing.T) {
	t.Parallel()

	t.Run("returns an equal live heat", func(t *testing.T) {
		t.Parallel()

		original := newLiveHeat(t)

		copied := original.Copy()

		assert.Equal(t, original, copied)
	})

	t.Run("keeps the original when the copy changes", func(t *testing.T) {
		t.Parallel()

		original := newLiveHeat(t)
		start := *original.StartTime

		copied := original.Copy()

		copied.Lanes[3].Time = 2000
		*copied.StartTime = start.Add(time.Hour)
		copied.Lanes[4] = &Lane{
			Number: 4,
		}
		assert.Equal(t, 1000, original.Lanes[3].Time, "changing a lane of the copy changes the original")
		assert.Equal(t, start, *original.StartTime, "changing the start of the copy changes the original")
		assert.Len(t, original.Lanes, 1, "adding a lane to the copy changes the original")
	})

	t.Run("returns empty lanes without a start", func(t *testing.T) {
		t.Parallel()

		copied := (&LiveHeat{}).Copy()

		assert.Nil(t, copied.StartTime)
		require.NotNil(t, copied.Lanes, "an empty map is encoded as an object, nil as null")
		assert.Empty(t, copied.Lanes)
	})
}

// newLiveHeat returns a started live heat with random values and one lane.
func newLiveHeat(t *testing.T) *LiveHeat {
	t.Helper()

	faker := apitest.Faker(t)
	start := faker.Date().UTC()
	return &LiveHeat{
		Event:     faker.IntRange(1, 999),
		Heat:      faker.IntRange(1, 9),
		HeatCount: faker.IntRange(1, 9),
		LapCount:  faker.IntRange(1, 9),
		Name:      faker.LetterN(12),
		Status:    HeatStatusStart,
		StartTime: &start,
		StopTime:  faker.IntRange(1, 600_000),
		Lanes: map[int]*Lane{
			3: {
				Number: 3,
				Time:   1000,
			},
		},
	}
}
