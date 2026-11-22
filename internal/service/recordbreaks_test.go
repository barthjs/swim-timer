package service

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

func TestRecordBreakService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the records set at the meet", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		breaks, err := svc.List(t.Context())

		require.NoError(t, err)
		require.Len(t, breaks, 1)
		assert.Equal(t, 6000, breaks[0].Record.Time)
	})
}

func TestRecordBreakService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the record break, so the next heats are checked against the record before it", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)

		err = svc.Delete(t.Context(), breaks[0].ID)

		require.NoError(t, err)
		results, err := svc.Check(t.Context(), testRecordCheck(t, 2, 6050))
		require.NoError(t, err)
		assert.Len(t, results[1], 1, "the time below the record of the table counts again")
	})
}

func TestRecordBreakService_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes every record break", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		err = svc.Clear(t.Context())

		require.NoError(t, err)
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)
		assert.Empty(t, breaks)
	})
}

func TestRecordBreakService_Check(t *testing.T) {
	t.Parallel()

	t.Run("reports a broken record and stores it", func(t *testing.T) {
		t.Parallel()

		table := newTestRecordTable(t, true, testMeetRecord(6100))
		svc := newTestRecordBreakService(t, table)
		input := testRecordCheck(t, 1, 6000)

		results, err := svc.Check(t.Context(), input)

		require.NoError(t, err)
		assert.Equal(t, map[int][]domain.LaneRecord{
			1: {
				{
					Abbreviation: table.Table.Abbreviation,
					TableName:    table.Table.Name,
					PreviousTime: 61000,
					PreviousName: table.Records[0].Name,
				},
			},
		}, results)
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)
		require.Len(t, breaks, 1)
		competitor := input.Lanes[0].Competitor
		assert.Equal(t, domain.RecordBreak{
			ID:           breaks[0].ID,
			TableID:      table.Table.ID,
			TableName:    table.Table.Name,
			Abbreviation: table.Table.Abbreviation,
			Event:        7,
			Heat:         1,
			Lane:         1,
			Record: domain.Record{
				Course:     domain.CourseLong,
				Gender:     "O",
				RelayCount: 1,
				Distance:   100,
				Stroke:     "BA",
				Time:       6000,
				Name:       competitor.LastName + ", " + competitor.FirstName,
				Club:       competitor.AbNat,
				Date:       "2026-05-01",
				BirthYear:  2001,
			},
			Previous: table.Records[0],
			BrokenAt: breaks[0].BrokenAt,
		}, breaks[0])
	})

	t.Run("writes the stored records in the background", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		tableStore := store.NewRecordTableStore(storage)
		table := newTestRecordTable(t, true, testMeetRecord(6100))
		require.NoError(t, tableStore.Save(t.Context(), table.Table, table.Records))
		svc := NewRecordBreakService(zerolog.Nop(), store.NewRecordBreakStore(storage), tableStore, nil)

		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		svc.Wait()
		breaks, err := store.NewRecordBreakStore(storage).List(t.Context())
		require.NoError(t, err)
		assert.Len(t, breaks, 1)
	})

	t.Run("reports the records it did not store before to onSet", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		var reported [][]domain.RecordBreak
		svc.onSet = func(breaks []domain.RecordBreak) { reported = append(reported, breaks) }
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		_, err = svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)
		_, err = svc.Check(t.Context(), testRecordCheck(t, 1, 5990))
		require.NoError(t, err)

		require.Len(t, reported, 2, "a heat made official again with the same record reports nothing")
		assert.Equal(t, 6000, reported[0][0].Record.Time)
		assert.Equal(t, 5990, reported[1][0].Record.Time, "a corrected time reports its record anew")
	})

	t.Run("reports an equalled record", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6100))

		require.NoError(t, err)
		require.Len(t, results[1], 1)
		assert.True(t, results[1][0].Equalled)
	})

	t.Run("compares the hundredths of the time", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		input := testRecordCheck(t, 1, 6100)
		input.Lanes[0].Time = 61009

		results, err := svc.Check(t.Context(), input)

		require.NoError(t, err)
		require.Len(t, results[1], 1, "the thousandths of the device do not count")
		assert.True(t, results[1][0].Equalled)
	})

	t.Run("reports every age range of the table the competitor is in", func(t *testing.T) {
		t.Parallel()

		open := testMeetRecord(6100)
		group := testMeetRecord(6300)
		group.AgeFrom = 20
		group.AgeTo = 29
		other := testMeetRecord(6400)
		other.AgeFrom = 30
		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, open, group, other))

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		require.Len(t, results[1], 2)
		assert.Equal(t, 61000, results[1][0].PreviousTime)
		assert.Equal(t, 20, results[1][1].AgeFrom)
		assert.Equal(t, 63000, results[1][1].PreviousTime)
	})

	t.Run("counts only for the first enabled table with a record", func(t *testing.T) {
		t.Parallel()

		disabled := newTestRecordTable(t, false, testMeetRecord(6100))
		first := newTestRecordTable(t, true, testMeetRecord(6200))
		second := newTestRecordTable(t, true, testMeetRecord(6300))
		svc := newTestRecordBreakService(t, disabled, first, second)

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		require.Len(t, results[1], 1)
		assert.Equal(t, first.Table.Abbreviation, results[1][0].Abbreviation)
	})

	t.Run("takes the fastest record of an age range a table lists twice", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6300), testMeetRecord(6100)))

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		require.Len(t, results[1], 1)
		assert.Equal(t, 61000, results[1][0].PreviousTime)
	})

	t.Run("checks the next heats against a record set earlier at the meet", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		slower, err := svc.Check(t.Context(), testRecordCheck(t, 2, 6050))
		require.NoError(t, err)
		faster, err := svc.Check(t.Context(), testRecordCheck(t, 3, 5900))
		require.NoError(t, err)

		assert.Empty(t, slower, "a time below the table but above the record of the meet is no record")
		require.Len(t, faster[1], 1)
		assert.Equal(t, 60000, faster[1][0].PreviousTime)
	})

	t.Run("lets only the fastest lane of a heat set a record", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		input := testRecordCheck(t, 1, 6050)
		input.Lanes = append(input.Lanes,
			RecordCheckLane{
				Lane:       2,
				Time:       60000,
				Competitor: input.Lanes[0].Competitor,
			},
			RecordCheckLane{
				Lane:       3,
				Time:       60000,
				Competitor: input.Lanes[0].Competitor,
			},
		)

		results, err := svc.Check(t.Context(), input)

		require.NoError(t, err)
		assert.Empty(t, results[1], "the slower lane is above the record of the faster one")
		require.Len(t, results[2], 1)
		assert.False(t, results[2][0].Equalled)
		require.Len(t, results[3], 1)
		assert.True(t, results[3][0].Equalled, "an equal time of the same heat equals the new record")
	})

	t.Run("replaces the record breaks of a heat made official again", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6050))

		require.NoError(t, err)
		require.Len(t, results[1], 1, "the heat is not checked against its own earlier record")
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)
		require.Len(t, breaks, 1)
		assert.Equal(t, 6050, breaks[0].Record.Time)
	})

	t.Run("keeps the id and the time of a record the heat stored before", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		input := testRecordCheck(t, 1, 6000)
		_, err := svc.Check(t.Context(), input)
		require.NoError(t, err)
		before, err := svc.List(t.Context())
		require.NoError(t, err)

		_, err = svc.Check(t.Context(), input)

		require.NoError(t, err)
		after, err := svc.List(t.Context())
		require.NoError(t, err)
		assert.Equal(t, before, after)
	})

	t.Run("checks a heat made official again only against the records set before it", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)
		_, err = svc.Check(t.Context(), testRecordCheck(t, 2, 5950))
		require.NoError(t, err)

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6010))

		require.NoError(t, err)
		require.Len(t, results[1], 1, "the faster record of the later heat does not count against the corrected heat")
		assert.Equal(t, 61000, results[1][0].PreviousTime)
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)
		assert.Len(t, breaks, 2)
	})

	t.Run("counts for the next table when the lane breaks no record of the first", func(t *testing.T) {
		t.Parallel()

		first := newTestRecordTable(t, true, testMeetRecord(5900))
		second := newTestRecordTable(t, true, testMeetRecord(6100))
		svc := newTestRecordBreakService(t, first, second)

		results, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		require.Len(t, results[1], 1)
		assert.Equal(t, second.Table.Abbreviation, results[1][0].Abbreviation)
	})

	t.Run("takes the year of the check for an event without a date", func(t *testing.T) {
		t.Parallel()

		group := testMeetRecord(6100)
		group.AgeFrom = 30
		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, group))
		svc.now = func() time.Time { return time.Date(2031, time.May, 1, 0, 0, 0, 0, time.UTC) }
		input := testRecordCheck(t, 1, 6000)
		input.Event.Date = ""

		results, err := svc.Check(t.Context(), input)

		require.NoError(t, err)
		assert.Len(t, results[1], 1, "the competitor born in 2001 is 30 in the year of the check")
	})

	ignored := []struct {
		name   string
		change func(input *RecordCheckInput)
	}{
		{
			name:   "ignores a pool of another length",
			change: func(input *RecordCheckInput) { input.PoolLength = 25 },
		},
		{
			name:   "ignores a relay",
			change: func(input *RecordCheckInput) { input.Event.Distance.Relay = 4 },
		},
		{
			name:   "ignores an event without a style",
			change: func(input *RecordCheckInput) { input.Event.Style = nil },
		},
		{
			name:   "ignores another stroke",
			change: func(input *RecordCheckInput) { input.Event.Style.Abbreviation = "FR" },
		},
		{
			name:   "ignores another gender",
			change: func(input *RecordCheckInput) { input.Lanes[0].Competitor.AbCat = "W" },
		},
		{
			name:   "ignores a lane without a competitor",
			change: func(input *RecordCheckInput) { input.Lanes[0].Competitor = nil },
		},
		{
			name:   "ignores a slower time",
			change: func(input *RecordCheckInput) { input.Lanes[0].Time = 62000 },
		},
	}

	for _, tt := range ignored {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
			input := testRecordCheck(t, 1, 6000)
			tt.change(&input)

			results, err := svc.Check(t.Context(), input)

			require.NoError(t, err)
			assert.Empty(t, results)
			breaks, err := svc.List(t.Context())
			require.NoError(t, err)
			assert.Empty(t, breaks)
		})
	}
}

func TestRecordBreakService_Preview(t *testing.T) {
	t.Parallel()

	t.Run("reports a broken record without storing it", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))

		results, err := svc.Preview(t.Context(), testRecordCheck(t, 1, 6000))

		require.NoError(t, err)
		assert.Len(t, results[1], 1)
		breaks, err := svc.List(t.Context())
		require.NoError(t, err)
		assert.Empty(t, breaks)
	})

	t.Run("checks against the records of the other heats but not its own", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t, newTestRecordTable(t, true, testMeetRecord(6100)))
		_, err := svc.Check(t.Context(), testRecordCheck(t, 1, 6000))
		require.NoError(t, err)

		own, err := svc.Preview(t.Context(), testRecordCheck(t, 1, 6050))
		require.NoError(t, err)
		other, err := svc.Preview(t.Context(), testRecordCheck(t, 2, 6050))
		require.NoError(t, err)

		assert.Len(t, own[1], 1, "the heat is not checked against its own earlier record")
		assert.Empty(t, other, "another heat is checked against the record set before")
	})

	t.Run("ignores a check without an event", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordBreakService(t)
		input := testRecordCheck(t, 1, 6000)
		input.Event = nil

		results, err := svc.Preview(t.Context(), input)

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// testRecordCheck returns heat of event 7, the 100 m backstroke of the category "O" on 1 May 2026 in a 50 m pool,
// with a competitor born in 2001 in lane 1 who finished in the given hundredths.
func testRecordCheck(t *testing.T, heat, hundredths int) RecordCheckInput {
	t.Helper()

	faker := apitest.Faker(t)
	return RecordCheckInput{
		Event: &domain.Event{
			Number: 7,
			Date:   "26/05/01",
			Distance: &domain.Distance{
				Length: 100,
				Relay:  1,
			},
			Style: &domain.Style{
				Abbreviation: "BA",
			},
		},
		Heat:       heat,
		PoolLength: 50,
		Lanes: []RecordCheckLane{
			{
				Lane: 1,
				Time: hundredths * 10,
				Competitor: &domain.Competitor{
					LastName:  faker.LastName(),
					FirstName: faker.FirstName(),
					BirthYear: "2001",
					AbNat:     faker.Company(),
					AbCat:     "o",
				},
			},
		},
	}
}
