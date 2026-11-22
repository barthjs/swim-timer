package api

import (
	"net/http"
	"time"

	"github.com/barthjs/swim-timer/internal/domain"
)

// recordListHandler returns every record of every record table.
//
//	@Summary		List records
//	@Description	Returns every record of every record table, in the order of the tables and of the records in their files. Times are in hundredths of a second.
//	@Tags			records
//	@Produce		json
//	@Success		200	{object}	RecordListResponse
//	@Router			/records [get]
func (a *API) recordListHandler(w http.ResponseWriter, r *http.Request) {
	list := a.recordTableService.ListRecords()

	resp := RecordListResponse{
		LoadedAt: optionalTime(list.LoadedAt),
		Records:  make([]RecordResponse, 0, len(list.Records)),
	}
	for _, entry := range list.Records {
		resp.Records = append(resp.Records, toRecordResponse(entry.Table.ID, entry.Record))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

type RecordListResponse struct {
	LoadedAt *time.Time       `json:"loadedAt,omitzero"`
	Records  []RecordResponse `json:"records"`
}

type RecordResponse struct {
	TableID string `json:"tableId"`
	Course  string `json:"course" enums:"SCM,LCM"`
	Gender  string `json:"gender"`
	// RelayCount is the number of swimmers per team, 1 for an individual event.
	RelayCount int `json:"relayCount"`
	// Distance is the distance each swimmer swims in meters, a relay leg for a relay.
	Distance int    `json:"distance"`
	Stroke   string `json:"stroke"`
	// AgeFrom is the lowest age the record applies to, 0 for an open bound.
	AgeFrom int `json:"ageFrom"`
	// AgeTo is the highest age the record applies to, 0 for an open bound.
	AgeTo int `json:"ageTo"`
	// Time is the record time in hundredths of a second.
	Time     int    `json:"time"`
	Approved bool   `json:"approved"`
	Name     string `json:"name"`
	Club     string `json:"club"`
	Place    string `json:"place"`
	// Date is the day the record was set as YYYY-MM-DD, empty if unknown.
	Date string `json:"date"`
	// BirthYear is the year of birth of the record holder, 0 if unknown.
	BirthYear int `json:"birthYear"`
}

// toRecordResponse maps a record of the table with the given id onto its response.
func toRecordResponse(tableID string, record domain.Record) RecordResponse {
	return RecordResponse{
		TableID:    tableID,
		Course:     string(record.Course),
		Gender:     record.Gender,
		RelayCount: record.RelayCount,
		Distance:   record.Distance,
		Stroke:     record.Stroke,
		AgeFrom:    record.AgeFrom,
		AgeTo:      record.AgeTo,
		Time:       record.Time,
		Approved:   record.Approved,
		Name:       record.Name,
		Club:       record.Club,
		Place:      record.Place,
		Date:       record.Date,
		BirthYear:  record.BirthYear,
	}
}
