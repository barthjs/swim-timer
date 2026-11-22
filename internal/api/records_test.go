package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/api"
)

func TestRecordAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/records",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestRecordListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the records of every table with their table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		firstFixture := newRecordTableFixture(h.Faker, meet, 50, 2)
		secondFixture := newRecordTableFixture(h.Faker, meet, 25, 1)
		first := h.storeRecordTable(firstFixture)
		second := h.storeRecordTable(secondFixture)

		response := h.Client.GET("/api/records")

		list := response.AssertOK().Decode[api.RecordListResponse]()
		expected := make([]api.RecordResponse, 0, 3)
		for _, record := range firstFixture.records {
			record.TableID = first.ID
			expected = append(expected, record)
		}
		for _, record := range secondFixture.records {
			record.TableID = second.ID
			expected = append(expected, record)
		}
		assert.Equal(t, expected, list.Records)
		assert.NotNil(t, list.LoadedAt)
	})

	t.Run("returns an empty list without tables", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/records")

		response.AssertOK().AssertJSONPath("records", []any{})
	})
}
