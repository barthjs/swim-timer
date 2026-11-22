package api

import (
	"net/http"
	"time"

	"github.com/barthjs/swim-timer/internal/domain"
)

// recordBreakListHandler returns every record set at the meet.
//
//	@Summary		List record breaks
//	@Description	Returns every record set or equalled at the meet, the most recent first, with the record it broke. Times are in hundredths of a second.
//	@Tags			records
//	@Produce		json
//	@Success		200	{array}		RecordBreakResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/record-breaks [get]
func (a *API) recordBreakListHandler(w http.ResponseWriter, r *http.Request) {
	breaks, err := a.recordBreakService.List(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	resp := make([]RecordBreakResponse, 0, len(breaks))
	for _, b := range breaks {
		resp = append(resp, ToRecordBreakResponse(b))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

type RecordBreakResponse struct {
	ID           string `json:"id"`
	TableID      string `json:"tableId"`
	TableName    string `json:"tableName"`
	Abbreviation string `json:"abbreviation"`
	Event        int    `json:"event"`
	Heat         int    `json:"heat"`
	Lane         int    `json:"lane"`
	// Equalled reports a time equal to the record, which the competitor then holds together with its holder.
	Equalled bool           `json:"equalled"`
	Record   RecordResponse `json:"record"`
	Previous RecordResponse `json:"previous"`
	BrokenAt time.Time      `json:"brokenAt"`
}

// recordBreakDeleteHandler removes a record set at the meet.
//
//	@Summary		Delete record break
//	@Description	Removes the record break with the given id, such as after a disqualification. The next heats are checked against the record that stood before it (Admin only).
//	@Tags			records
//	@Param			id	path	string	true	"Record break id"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-breaks/{id} [delete]
func (a *API) recordBreakDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.recordBreakService.Delete(r.Context(), pathParam(r, "id")); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// recordBreakClearHandler removes every record set at the meet.
//
//	@Summary		Clear record breaks
//	@Description	Removes every record break, such as before the next meet (Admin only).
//	@Tags			records
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-breaks [delete]
func (a *API) recordBreakClearHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.recordBreakService.Clear(r.Context()); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToRecordBreakResponse maps a record break onto its response. The webhooks send it as the data of a record,
// so the wiring of the application uses it as well.
func ToRecordBreakResponse(b domain.RecordBreak) RecordBreakResponse {
	return RecordBreakResponse{
		ID:           b.ID,
		TableID:      b.TableID,
		TableName:    b.TableName,
		Abbreviation: b.Abbreviation,
		Event:        b.Event,
		Heat:         b.Heat,
		Lane:         b.Lane,
		Equalled:     b.Equalled,
		Record:       toRecordResponse(b.TableID, b.Record),
		Previous:     toRecordResponse(b.TableID, b.Previous),
		BrokenAt:     b.BrokenAt,
	}
}
