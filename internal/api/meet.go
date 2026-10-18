package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/service"
)

const (
	// maxMeetUploadMB bounds the size of a single meet file.
	maxMeetUploadMB = 10
	// meetReloadRateLimit bounds how often the meet data can be reloaded per client.
	meetReloadRateLimit       = 10
	meetReloadRateLimitWindow = time.Minute
)

var (
	// allowedMeetUploadTypes are the detected content types a meet file may have.
	allowedMeetUploadTypes = []string{"text/plain"}
	// allowedMeetUploadExtensions are the file extensions a meet file may have, compared in lowercase.
	// The timing software exports every meet file as .TXT.
	allowedMeetUploadExtensions = []string{".txt"}
)

// meetOverviewGetHandler returns the whole meet.
//
//	@Summary		Get meet overview
//	@Description	Returns the events, start entries, and competitors of the loaded meet data in the order of the program. Withdrawn start entries are left out.
//	@Tags			meet
//	@Produce		json
//	@Success		200	{object}	MeetResponse
//	@Router			/meet [get]
func (a *API) meetOverviewGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, r, http.StatusOK, toMeetResponse(a.meetService.GetOverview()), nil)
}

// MeetResponse is the whole meet. Competitors are listed once and referenced by id from the start entries.
type MeetResponse struct {
	LoadedAt     *time.Time           `json:"loadedAt,omitzero"`
	Events       []EventResponse      `json:"events"`
	StartEntries []StartEntryResponse `json:"startEntries"`
	Competitors  []CompetitorResponse `json:"competitors"`
}

// EventResponse is an event of the meet with its references resolved.
type EventResponse struct {
	Number int `json:"number"`
	// Phase is the phase of the event, such as heats or final, and PhaseName its name, empty if unknown.
	Phase     int    `json:"phase"`
	PhaseName string `json:"phaseName"`
	HeatCount int    `json:"heatCount"`
	Name      string `json:"name"`
	Date      string `json:"date"`
	// Time is the scheduled start time, empty for an event without one.
	Time     string            `json:"time"`
	Distance *DistanceResponse `json:"distance"`
	Style    *StyleResponse    `json:"style"`
	Category *CategoryResponse `json:"category"`
}

// DistanceResponse is the distance of an event.
type DistanceResponse struct {
	Name       string `json:"name"`
	Length     int    `json:"length"`
	RelayCount int    `json:"relayCount"`
}

// StyleResponse is the swimming style of an event.
type StyleResponse struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

// CategoryResponse is the category of an event.
type CategoryResponse struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

// StartEntryResponse places a competitor in a heat and lane. The heat is counted from 1.
type StartEntryResponse struct {
	Event         int `json:"event"`
	Phase         int `json:"phase"`
	Heat          int `json:"heat"`
	Lane          int `json:"lane"`
	RelayPosition int `json:"relayPosition"`
	CompetitorID  int `json:"competitorId"`
}

// CompetitorResponse is a competitor of the meet.
type CompetitorResponse struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	BirthYear string `json:"birthYear"`
	Club      string `json:"club"`
	Category  string `json:"category"`
}

// meetEventListHandler returns every event of the meet.
//
//	@Summary		List meet events
//	@Description	Returns the events of the loaded meet data in the order of the program.
//	@Tags			meet
//	@Produce		json
//	@Success		200	{object}	MeetEventListResponse
//	@Router			/meet/events [get]
func (a *API) meetEventListHandler(w http.ResponseWriter, r *http.Request) {
	list := a.meetService.ListEvents()

	resp := MeetEventListResponse{
		LoadedAt: toLoadedAtResponse(list.LoadedAt),
		Events:   make([]EventResponse, 0, len(list.Events)),
	}
	for _, event := range list.Events {
		resp.Events = append(resp.Events, toEventResponse(event))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetEventListResponse is every event of the meet.
type MeetEventListResponse struct {
	LoadedAt *time.Time      `json:"loadedAt,omitzero"`
	Events   []EventResponse `json:"events"`
}

// meetEventGetHandler returns one event of the meet.
//
//	@Summary		Get meet event
//	@Description	Returns the event with the given number from the loaded meet data.
//	@Tags			meet
//	@Produce		json
//	@Param			number	path		int	true	"Event number"
//	@Success		200		{object}	MeetEventResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/meet/events/{number} [get]
func (a *API) meetEventGetHandler(w http.ResponseWriter, r *http.Request) {
	number, err := pathInt(r, "number")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	event, err := a.meetService.GetEvent(number)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, MeetEventResponse{
		LoadedAt: toLoadedAtResponse(event.LoadedAt),
		Event:    toEventResponse(event.Event),
	}, nil)
}

// MeetEventResponse is one event of the meet.
type MeetEventResponse struct {
	LoadedAt *time.Time    `json:"loadedAt,omitzero"`
	Event    EventResponse `json:"event"`
}

// meetHeatGetHandler returns an event and the start entries of one of its heats.
//
//	@Summary		Get meet heat
//	@Description	Returns the event with the given number and the start entries of its heat, counted from 1, with their competitors. The start entries are ordered by lane and relay position, and withdrawn start entries are left out.
//	@Tags			meet
//	@Produce		json
//	@Param			number	path		int	true	"Event number"
//	@Param			heat	path		int	true	"Heat, counted from 1"
//	@Success		200		{object}	MeetHeatResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Router			/meet/events/{number}/heats/{heat} [get]
func (a *API) meetHeatGetHandler(w http.ResponseWriter, r *http.Request) {
	number, err := pathInt(r, "number")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}
	heatNumber, err := pathInt(r, "heat")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	heat, err := a.meetService.GetHeat(number, heatNumber)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	resp := MeetHeatResponse{
		LoadedAt:     toLoadedAtResponse(heat.LoadedAt),
		Event:        toEventResponse(heat.Event),
		Heat:         heatNumber,
		StartEntries: make([]HeatStartEntryResponse, 0, len(heat.StartEntries)),
	}
	for _, entry := range heat.StartEntries {
		entryResp := HeatStartEntryResponse{
			Lane:          entry.Lane,
			RelayPosition: entry.Relais,
		}
		if entry.Competitor != nil {
			competitor := toCompetitorResponse(entry.Competitor)
			entryResp.Competitor = &competitor
		}
		resp.StartEntries = append(resp.StartEntries, entryResp)
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetHeatResponse is an event and the start entries of one of its heats. The heat is counted from 1.
type MeetHeatResponse struct {
	LoadedAt     *time.Time               `json:"loadedAt,omitzero"`
	Event        EventResponse            `json:"event"`
	Heat         int                      `json:"heat"`
	StartEntries []HeatStartEntryResponse `json:"startEntries"`
}

// HeatStartEntryResponse places a competitor in a lane of a heat. Competitor is null if it does not exist.
type HeatStartEntryResponse struct {
	Lane          int                 `json:"lane"`
	RelayPosition int                 `json:"relayPosition"`
	Competitor    *CompetitorResponse `json:"competitor"`
}

// meetCompetitorListHandler returns every competitor of the meet.
//
//	@Summary		List meet competitors
//	@Description	Returns the competitors of the loaded meet data, ordered by name.
//	@Tags			meet
//	@Produce		json
//	@Success		200	{object}	MeetCompetitorListResponse
//	@Router			/meet/competitors [get]
func (a *API) meetCompetitorListHandler(w http.ResponseWriter, r *http.Request) {
	list := a.meetService.ListCompetitors()

	resp := MeetCompetitorListResponse{
		LoadedAt:    toLoadedAtResponse(list.LoadedAt),
		Competitors: make([]CompetitorResponse, 0, len(list.Competitors)),
	}
	for _, competitor := range list.Competitors {
		resp.Competitors = append(resp.Competitors, toCompetitorResponse(competitor))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetCompetitorListResponse is every competitor of the meet.
type MeetCompetitorListResponse struct {
	LoadedAt    *time.Time           `json:"loadedAt,omitzero"`
	Competitors []CompetitorResponse `json:"competitors"`
}

// meetStartEntryListHandler returns the start list of the meet.
//
//	@Summary		List meet start entries
//	@Description	Returns the start entries of the loaded meet data in the order of the heats, with heats counted from 1. Withdrawn start entries are left out.
//	@Tags			meet
//	@Produce		json
//	@Success		200	{object}	MeetStartEntryListResponse
//	@Router			/meet/start-entries [get]
func (a *API) meetStartEntryListHandler(w http.ResponseWriter, r *http.Request) {
	list := a.meetService.ListStartEntries()

	resp := MeetStartEntryListResponse{
		LoadedAt:     toLoadedAtResponse(list.LoadedAt),
		StartEntries: make([]StartEntryResponse, 0, len(list.StartEntries)),
	}
	for _, entry := range list.StartEntries {
		resp.StartEntries = append(resp.StartEntries, toStartEntryResponse(entry))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetStartEntryListResponse is the start list of the meet.
type MeetStartEntryListResponse struct {
	LoadedAt     *time.Time           `json:"loadedAt,omitzero"`
	StartEntries []StartEntryResponse `json:"startEntries"`
}

// toMeetResponse maps the meet overview to its response.
func toMeetResponse(overview service.MeetOverview) MeetResponse {
	resp := MeetResponse{
		LoadedAt:     toLoadedAtResponse(overview.LoadedAt),
		Events:       make([]EventResponse, 0, len(overview.Events)),
		StartEntries: make([]StartEntryResponse, 0, len(overview.StartEntries)),
		Competitors:  make([]CompetitorResponse, 0, len(overview.Competitors)),
	}
	for _, event := range overview.Events {
		resp.Events = append(resp.Events, toEventResponse(event))
	}
	for _, entry := range overview.StartEntries {
		resp.StartEntries = append(resp.StartEntries, toStartEntryResponse(entry))
	}
	for _, competitor := range overview.Competitors {
		resp.Competitors = append(resp.Competitors, toCompetitorResponse(competitor))
	}

	return resp
}

// toLoadedAtResponse maps the time the meet data was loaded to its response, nil before the first load.
func toLoadedAtResponse(loadedAt time.Time) *time.Time {
	if loadedAt.IsZero() {
		return nil
	}
	return &loadedAt
}

// toEventResponse maps an event to its response with its references resolved.
func toEventResponse(event *domain.Event) EventResponse {
	resp := EventResponse{
		Number:    event.Number,
		Phase:     event.PhaseID,
		PhaseName: event.PhaseName(),
		HeatCount: event.HeatCount,
		Name:      event.DisplayName(),
		Date:      event.ISODate(),
		Time:      event.TrimmedTime(),
	}
	if event.Distance != nil {
		resp.Distance = &DistanceResponse{
			Name:       event.Distance.Name,
			Length:     event.Distance.Length,
			RelayCount: event.Distance.Relay,
		}
	}
	if event.Style != nil {
		resp.Style = &StyleResponse{
			Name:         event.Style.Name,
			Abbreviation: event.Style.Abbreviation,
		}
	}
	if event.Category != nil {
		resp.Category = &CategoryResponse{
			Name:         event.Category.Name,
			Abbreviation: event.Category.Abbreviation,
		}
	}
	return resp
}

// toStartEntryResponse maps a start entry of the meet to its response.
func toStartEntryResponse(entry service.MeetStartEntry) StartEntryResponse {
	return StartEntryResponse{
		Event:         entry.Event,
		Phase:         entry.Phase,
		Heat:          entry.Heat,
		Lane:          entry.Lane,
		RelayPosition: entry.RelayPosition,
		CompetitorID:  entry.CompetitorID,
	}
}

// toCompetitorResponse maps a competitor to its response.
func toCompetitorResponse(competitor *domain.Competitor) CompetitorResponse {
	return CompetitorResponse{
		ID:        competitor.ID,
		FirstName: competitor.FirstName,
		LastName:  competitor.LastName,
		BirthYear: competitor.BirthYear,
		Club:      competitor.Club(),
		Category:  competitor.AbCat,
	}
}

// meetFileListHandler reports for every meet file whether it is available.
//
//	@Summary		List meet files
//	@Description	Reports for every meet file whether it is available and the name it is stored under (Admin only).
//	@Tags			meet
//	@Produce		json
//	@Success		200	{array}		MeetFileResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/meet/files [get]
func (a *API) meetFileListHandler(w http.ResponseWriter, r *http.Request) {
	names, err := a.meetService.ListFiles(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	resp := make([]MeetFileResponse, 0, len(domain.MeetFiles()))
	for _, file := range domain.MeetFiles() {
		resp = append(resp, MeetFileResponse{
			File:      string(file),
			Filename:  names[file],
			Available: names[file] != "",
		})
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetFileResponse reports whether a meet file is available and under which name it is stored.
type MeetFileResponse struct {
	File      string `json:"file"`
	Filename  string `json:"filename,omitempty"`
	Available bool   `json:"available"`
}

// meetFileSaveHandler replaces one meet file without reloading the meet data.
//
//	@Summary		Save meet file
//	@Description	Replaces the given meet file. The meet data is not reloaded (Admin only).
//	@Tags			meet
//	@Accept			multipart/form-data
//	@Param			file	path		string	true	"Meet file"	Enums(LSTCAT, LSTCONC, LSTLONG, LSTRACE, LSTROUND, LSTSTART, LSTSTYLE)
//	@Param			file	formData	file	true	"File content"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/meet/files/{file} [put]
func (a *API) meetFileSaveHandler(w http.ResponseWriter, r *http.Request) {
	file, ok := domain.ParseMeetFile(chi.URLParam(r, "file"))
	if !ok {
		a.notFoundResponse(w, r)
		return
	}

	data, filename, ok := a.readUpload(w, r, maxMeetUploadMB, allowedMeetUploadTypes)
	if !ok {
		return
	}
	if !meetUploadExtensionAllowed(filename) {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "unsupported_type",
			},
		}})
		return
	}

	if err := a.meetService.SaveFile(r.Context(), file, data); err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// meetFileUploadHandler stores every uploaded file whose name matches a meet file.
// Each file is reported on its own, so one unknown or broken file does not reject the others.
// It does not reload the meet data.
//
//	@Summary		Upload meet files
//	@Description	Stores every file of the "files" field whose name, ignoring extension and case, matches a meet file. The meet data is not reloaded (Admin only).
//	@Tags			meet
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			files	formData	file	true	"Meet files"
//	@Success		200		{array}		MeetFileUploadResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Security		SessionAuth
//	@Router			/meet/files [post]
func (a *API) meetFileUploadHandler(w http.ResponseWriter, r *http.Request) {
	maxFile := int64(maxMeetUploadMB) << 20

	// The route limits the body to every meet file at its maximum size, so a larger body fails here,
	// while files beyond maxFile in memory are kept in temporary files.
	if !a.parseMultipartForm(w, r, maxFile, "files") {
		return
	}

	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		a.failedValidationResponse(w, r, FieldErrors{"files": {
			{
				Tag: "required",
			},
		}})
		return
	}

	resp := make([]MeetFileUploadResponse, 0, len(headers))
	for _, header := range headers {
		resp = append(resp, a.storeMeetFileUpload(r, header, maxFile))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// MeetFileUploadResponse reports for an uploaded file which meet file it was stored as, or why it was rejected.
type MeetFileUploadResponse struct {
	Filename string `json:"filename"`
	File     string `json:"file,omitempty"`
	Error    string `json:"error,omitempty"`
}

// storeMeetFileUpload matches one uploaded file to a meet file and stores it,
// reporting the outcome instead of writing a response.
func (a *API) storeMeetFileUpload(r *http.Request, header *multipart.FileHeader, maxFile int64) MeetFileUploadResponse {
	result := MeetFileUploadResponse{
		Filename: header.Filename,
	}

	file, ok := domain.MatchMeetFile(header.Filename)
	if !ok {
		result.Error = "unknown_file"
		return result
	}
	result.File = string(file)

	if header.Size > maxFile {
		result.Error = "too_large"
		return result
	}

	f, err := header.Open()
	if err != nil {
		a.logger.Error().Err(err).Str("filename", header.Filename).Msg("failed to open uploaded meet file")
		result.Error = "server_error"
		return result
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, maxFile))
	if err != nil {
		a.logger.Error().Err(err).Str("filename", header.Filename).Msg("failed to read uploaded meet file")
		result.Error = "server_error"
		return result
	}

	if !meetUploadAllowed(header.Filename, data) {
		result.Error = "unsupported_type"
		return result
	}

	if err := a.meetService.SaveFile(r.Context(), file, data); err != nil {
		a.logger.Error().Err(err).Str("filename", header.Filename).Msg("failed to store meet file")
		result.Error = "server_error"
		return result
	}

	return result
}

// meetUploadAllowed reports whether an uploaded file has one of the extensions
// the timing software exports and its content is plain text.
func meetUploadAllowed(filename string, data []byte) bool {
	return meetUploadExtensionAllowed(filename) && uploadTypeAllowed(data, allowedMeetUploadTypes)
}

// meetUploadExtensionAllowed reports whether an uploaded file has one of the extensions the timing software exports.
func meetUploadExtensionAllowed(filename string) bool {
	return slices.Contains(allowedMeetUploadExtensions, strings.ToLower(filepath.Ext(filename)))
}

// meetReloadHandler reads all meet files again.
//
//	@Summary		Reload meet data
//	@Description	Reads all meet files again. The live heat takes the new data with the next frame of the timing device. A file that cannot be read fails with 400, naming the file and the line, and the previous meet data stays in use (Admin only).
//	@Tags			meet
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/meet/reload [post]
func (a *API) meetReloadHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.meetService.Reload(r.Context()); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
