package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestRecordBreakAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/record-breaks",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/record-breaks/{id}",
			arrange: func(h *harness) (string, any) {
				return "/api/record-breaks/" + h.storeRecordBreak().ID, nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/record-breaks",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
	})
}

func TestRecordBreakListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the records an official heat broke", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		fixture := newRecordTableFixture(h.Faker, meet, h.Config.Timing.PoolLength, 1)
		table := h.storeRecordTable(fixture)

		heat := h.breakRecord(meet)

		breaks := h.Client.GET("/api/record-breaks").AssertOK().Decode[[]api.RecordBreakResponse]()
		require.Len(t, breaks, 1)
		competitor := meet.competitor(1, 1)
		previous := fixture.records[0]
		previous.TableID = table.ID
		assert.Equal(t, api.RecordBreakResponse{
			ID:           breaks[0].ID,
			TableID:      table.ID,
			TableName:    table.Name,
			Abbreviation: table.Abbreviation,
			Event:        meet.event.Number,
			Heat:         1,
			Lane:         1,
			Record: api.RecordResponse{
				TableID:    table.ID,
				Course:     previous.Course,
				Gender:     previous.Gender,
				RelayCount: 1,
				Distance:   previous.Distance,
				Stroke:     previous.Stroke,
				Time:       heat.Lanes[1].Time / 10,
				Name:       competitor.LastName + ", " + competitor.FirstName,
				Club:       competitor.AbNat,
				Date:       meet.event.ISODate(),
				BirthYear:  breaks[0].Record.BirthYear,
			},
			Previous: previous,
			BrokenAt: breaks[0].BrokenAt,
		}, breaks[0])
		assert.Equal(t, []domain.LaneRecord{
			{
				Abbreviation: table.Abbreviation,
				TableName:    table.Name,
				PreviousTime: previous.Time * 10,
				PreviousName: previous.Name,
			},
		}, heat.Lanes[1].Records, "the live heat shows the record at its lane")
	})

	t.Run("sends the records of an official heat to the webhooks", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withWebhookReceivers())
		receiver := newWebhookReceiver(t)
		request := newWebhookCreateRequest(h.Faker, receiver.URL())
		request.Events = []string{
			"record.set",
		}
		h.asAdmin().POST("/api/webhooks", request).AssertCreated()
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, h.Config.Timing.PoolLength, 1))

		h.breakRecord(meet)

		message := receiver.next(t)
		assert.Equal(t, "record.set", message.Type)
		var sent []api.RecordBreakResponse
		require.NoError(t, json.Unmarshal(message.Data, &sent))
		require.Len(t, sent, 1)
		assert.Equal(t, table.ID, sent[0].TableID)
		assert.Equal(t, 1, sent[0].Lane, "the records as the list of records shows them")
	})

	t.Run("returns an empty list without record breaks", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/record-breaks")

		response.AssertOK().AssertJSON([]api.RecordBreakResponse{})
	})
}

func TestRecordBreakDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the record break", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		recordBreak := h.storeRecordBreak()

		response := h.asAdmin().DELETE("/api/record-breaks/" + recordBreak.ID)

		response.AssertNoContent()
		h.Client.GET("/api/record-breaks").AssertOK().AssertJSON([]api.RecordBreakResponse{})
	})

	t.Run("returns not found for an unknown record break", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE("/api/record-breaks/" + h.Faker.LetterN(12))

		response.AssertNotFound()
	})
}

func TestRecordBreakClearHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes every record break", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.storeRecordBreak()

		response := h.asAdmin().DELETE("/api/record-breaks")

		response.AssertNoContent()
		h.Client.GET("/api/record-breaks").AssertOK().AssertJSON([]api.RecordBreakResponse{})
	})
}
