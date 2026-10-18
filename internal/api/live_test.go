package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/omega"
)

func TestLiveAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/live",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestLiveHeatGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the start list of a loaded heat", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: h.lapCount(meet),
			Event:    meet.event.Number,
			Heat:     2,
		})

		heat := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Status == domain.HeatStatusLoad }, "the heat is loaded")
		assert.Equal(t, meet.event.Number, heat.Event)
		assert.Equal(t, 2, heat.Heat)
		assert.Equal(t, 2, heat.HeatCount)
		assert.Equal(t, meet.displayName(), heat.Name)
		require.Len(t, heat.Lanes, meetLaneCount)
		competitor := meet.competitor(2, 5)
		lane := heat.Lanes[5]
		require.NotNil(t, lane)
		assert.Equal(t, competitor.FirstName, lane.FirstName)
		assert.Equal(t, competitor.LastName, lane.LastName)
		assert.Equal(t, competitor.BirthYear, lane.BirthYear)
		assert.Equal(t, competitor.AbNat, lane.Club)
	})

	// The frames of one heat follow each other, so this subtest sends several frames and checks the state after each
	t.Run("follows the frames through the heat", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		lapCount := h.lapCount(meet)
		event := meet.event.Number
		lane := h.Faker.IntRange(1, meetLaneCount)

		h.Device.Send(t,
			omega.FrameData{
				Kind:     omega.KindLoad,
				LapCount: lapCount,
				Event:    event,
				Heat:     1,
			},
			omega.FrameData{
				Kind:     omega.KindStart,
				LapCount: lapCount,
				Event:    event,
				Heat:     1,
			},
		)
		started := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Status == domain.HeatStatusStart }, "the heat starts")
		require.NotNil(t, started.StartTime)
		assert.WithinDuration(t, time.Now(), *started.StartTime, eventuallyTimeout, "a start without time of day starts at its arrival")

		// The device transmits hundredths of a second
		splitTime := time.Duration(h.Faker.IntRange(2_000, 4_000)) * 10 * time.Millisecond
		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindIntermediate,
			LapCount: lapCount,
			Event:    event,
			Heat:     1,
			Rank:     h.Faker.IntRange(1, meetLaneCount),
			Lane:     lane,
			Lap:      1,
			Time:     splitTime,
		})
		split := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Lanes[lane].Lap == 1 }, "the split time arrives")
		assert.Equal(t, int(splitTime.Milliseconds()), split.Lanes[lane].Time)
		assert.Equal(t, h.Config.Timing.PoolLength, split.Lanes[lane].Meter)
		assert.Equal(t, 1, split.Lanes[lane].Rank, "the rank comes from the times of the lap, not from the device")
		assert.False(t, split.Lanes[lane].Finished)

		finishTime := splitTime * time.Duration(lapCount)
		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindFinish,
			LapCount: lapCount,
			Event:    event,
			Heat:     1,
			Rank:     1,
			Lane:     lane,
			Lap:      lapCount,
			Time:     finishTime,
		})
		finished := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Lanes[lane].Finished }, "the swimmer finishes")
		assert.Equal(t, int(finishTime.Milliseconds()), finished.Lanes[lane].Time)
		assert.Equal(t, meet.distance.Length, finished.Lanes[lane].Meter)
		assert.Equal(t, 1, finished.Lanes[lane].Rank)

		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindOfficial,
			LapCount: lapCount,
			Event:    event,
			Heat:     1,
		})
		official := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Status == domain.HeatStatusOfficial }, "the heat is official")
		assert.True(t, official.Lanes[lane].Finished, "the result stays")
	})

	t.Run("returns an empty heat before the first frame", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/live")

		response.AssertOK().AssertJSONPath("status", "").AssertJSONPath("lanes", map[string]any{}).AssertJSONPathMissing("startTime")
	})

	t.Run("returns an unknown event without meet data", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		unknown := meet.event.Number + 1

		h.Device.Send(t, omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: 2,
			Event:    unknown,
			Heat:     1,
		})

		heat := h.waitForLive(func(heat domain.LiveHeat) bool { return heat.Event == unknown }, "the heat of the unknown event is shown")
		assert.Empty(t, heat.Name, "the heat has no meet data")
		assert.Empty(t, heat.Lanes)
	})
}

// lapCount is the number of laps of the meet's event in a pool of the configured length.
func (h *harness) lapCount(m meetFixture) int {
	return m.distance.Length / h.Config.Timing.PoolLength
}

// waitForLive waits until the live heat satisfies condition, since frames are handled in the background.
func (h *harness) waitForLive(condition func(domain.LiveHeat) bool, message string) domain.LiveHeat {
	h.t.Helper()

	var heat domain.LiveHeat
	require.Eventually(h.t, func() bool {
		heat = h.Client.GET("/api/live").Decode[domain.LiveHeat]()
		return condition(heat)
	}, eventuallyTimeout, eventuallyTick, message)
	return heat
}
