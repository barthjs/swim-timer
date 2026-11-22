package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/tabular"
)

const (
	// maxRecordTableUploadMB bounds the size of an imported record table.
	maxRecordTableUploadMB = 10
	// recordTableReloadRateLimit bounds how often the record tables can be reloaded per client.
	recordTableReloadRateLimit       = 10
	recordTableReloadRateLimitWindow = time.Minute
)

// recordTableListHandler returns every record table.
//
//	@Summary		List record tables
//	@Description	Returns every record table in the order they are checked, with the number of their records.
//	@Tags			records
//	@Produce		json
//	@Success		200	{object}	RecordTableListResponse
//	@Router			/record-tables [get]
func (a *API) recordTableListHandler(w http.ResponseWriter, r *http.Request) {
	list := a.recordTableService.List()

	resp := RecordTableListResponse{
		LoadedAt: optionalTime(list.LoadedAt),
		Tables:   make([]RecordTableResponse, 0, len(list.Tables)),
	}
	for _, table := range list.Tables {
		resp.Tables = append(resp.Tables, toRecordTableResponse(table))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// RecordTableListResponse lists the record tables in the order they are checked.
type RecordTableListResponse struct {
	LoadedAt *time.Time            `json:"loadedAt,omitzero"`
	Tables   []RecordTableResponse `json:"tables"`
}

type RecordTableResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Enabled      bool   `json:"enabled"`
	RecordCount  int    `json:"recordCount"`
}

// recordTableGetHandler returns one record table.
//
//	@Summary		Get record table
//	@Description	Returns the record table with the given id and the number of its records.
//	@Tags			records
//	@Produce		json
//	@Param			id	path		string	true	"Record table id"
//	@Success		200	{object}	RecordTableResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/record-tables/{id} [get]
func (a *API) recordTableGetHandler(w http.ResponseWriter, r *http.Request) {
	table, err := a.recordTableService.Get(pathParam(r, "id"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toRecordTableResponse(table), nil)
}

type RecordTableCreateRequest struct {
	Name         string        `json:"name" validate:"required,max=100"`
	Abbreviation string        `json:"abbreviation" validate:"required,max=10"`
	Enabled      bool          `json:"enabled"`
	Mapping      ImportMapping `json:"mapping"`
}

// recordTableCreateHandler creates a record table with the records of a file.
//
//	@Summary		Create record table
//	@Description	Creates a record table under a generated id at the end of the order and imports its records from the file in the "file" field, mapped as the RecordTableCreateRequest in the "data" field describes. A file that cannot be read, a mapping that does not fit, or invalid rows fail with 422 on the fields "file", "mapping.columns.<key>", "mapping.values.<key>", and "rows.<line>.<key>", and nothing is stored (Admin only).
//	@Tags			records
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Table of records"
//	@Param			data	formData	string	true	"RecordTableCreateRequest as JSON"
//	@Success		201		{object}	RecordTableResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables [post]
func (a *API) recordTableCreateHandler(w http.ResponseWriter, r *http.Request) {
	content, req, ok := a.readImportUpload[RecordTableCreateRequest](w, r, maxRecordTableUploadMB)
	if !ok {
		return
	}

	table, err := a.recordTableService.Create(r.Context(), service.RecordTableCreateInput{
		Name:         req.Name,
		Abbreviation: req.Abbreviation,
		Enabled:      req.Enabled,
		Content:      content,
		Mapping:      toTabularMapping(req.Mapping),
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusCreated, toRecordTableResponse(table), nil)
}

type RecordTableUpdateRequest struct {
	Name         string `json:"name" validate:"required,max=100"`
	Abbreviation string `json:"abbreviation" validate:"required,max=10"`
	Enabled      bool   `json:"enabled"`
}

// recordTableUpdateHandler replaces the name, the abbreviation, and the state of a record table.
//
//	@Summary		Update record table
//	@Description	Replaces the name, the abbreviation, and the state of the record table with the given id. Its records stay (Admin only).
//	@Tags			records
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Record table id"
//	@Param			request	body		RecordTableUpdateRequest	true	"Record table"
//	@Success		200		{object}	RecordTableResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/{id} [put]
func (a *API) recordTableUpdateHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[RecordTableUpdateRequest](w, r)
	if !ok {
		return
	}

	table, err := a.recordTableService.Update(r.Context(), pathParam(r, "id"), service.RecordTableUpdateInput{
		Name:         req.Name,
		Abbreviation: req.Abbreviation,
		Enabled:      req.Enabled,
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toRecordTableResponse(table), nil)
}

// recordTableDeleteHandler removes a record table.
//
//	@Summary		Delete record table
//	@Description	Removes the record table with the given id and its records. The records set at the meet stay (Admin only).
//	@Tags			records
//	@Param			id	path	string	true	"Record table id"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/{id} [delete]
func (a *API) recordTableDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.recordTableService.Delete(r.Context(), pathParam(r, "id")); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RecordTablePreviewRequest struct {
	// Format is the format of the file, whose empty fields are detected.
	Format TableFormat `json:"format"`
}

// recordTablePreviewHandler reads the start of a file of records and guesses its mapping.
//
//	@Summary		Preview record table
//	@Description	Reads the start of the file in the "file" field in the format of the RecordTablePreviewRequest in the "data" field, detects its empty fields, and guesses the mapping of its columns and values onto the columns of a record. A file that cannot be read fails with 422 on the field "file" (Admin only).
//	@Tags			records
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Table of records"
//	@Param			data	formData	string	true	"RecordTablePreviewRequest as JSON"
//	@Success		200		{object}	ImportPreviewResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/preview [post]
func (a *API) recordTablePreviewHandler(w http.ResponseWriter, r *http.Request) {
	content, req, ok := a.readImportUpload[RecordTablePreviewRequest](w, r, maxRecordTableUploadMB)
	if !ok {
		return
	}

	preview, err := a.recordTableService.Preview(content, toTabularFormat(req.Format))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toImportPreviewResponse(preview.Preview, preview.Columns), nil)
}

type RecordTableImportRequest struct {
	Mapping ImportMapping `json:"mapping"`
}

// recordTableImportHandler replaces the records of a record table with the records of a file.
//
//	@Summary		Import record table
//	@Description	Replaces the records of the record table with the given id with the records of the file in the "file" field, mapped as the RecordTableImportRequest in the "data" field describes. It fails like the creation of a record table and keeps the records on failure (Admin only).
//	@Tags			records
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		string	true	"Record table id"
//	@Param			file	formData	file	true	"Table of records"
//	@Param			data	formData	string	true	"RecordTableImportRequest as JSON"
//	@Success		200		{object}	RecordTableResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/{id}/import [post]
func (a *API) recordTableImportHandler(w http.ResponseWriter, r *http.Request) {
	content, req, ok := a.readImportUpload[RecordTableImportRequest](w, r, maxRecordTableUploadMB)
	if !ok {
		return
	}

	table, err := a.recordTableService.Import(r.Context(), pathParam(r, "id"), service.RecordTableImportInput{
		Content: content,
		Mapping: toTabularMapping(req.Mapping),
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toRecordTableResponse(table), nil)
}

type RecordTableReorderRequest struct {
	IDs []string `json:"ids" validate:"required,max=100,unique"`
}

// recordTableReorderHandler puts the record tables in a new order.
//
//	@Summary		Reorder record tables
//	@Description	Puts the record tables in the order of the given ids, which decides the table a record counts for. The ids must name every table once, otherwise it fails with 422 on the field "ids" (Admin only).
//	@Tags			records
//	@Accept			json
//	@Param			request	body	RecordTableReorderRequest	true	"Order of the record tables"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/order [put]
func (a *API) recordTableReorderHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[RecordTableReorderRequest](w, r)
	if !ok {
		return
	}

	if err := a.recordTableService.Reorder(r.Context(), req.IDs); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// recordTableReloadHandler reads every record table and the records set at the meet again.
//
//	@Summary		Reload record tables
//	@Description	Reads every record table and the records set at the meet again from the storage. A file that cannot be read fails with 400, naming the file and the line, and its previous data stays in use (Admin only).
//	@Tags			records
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/record-tables/reload [post]
func (a *API) recordTableReloadHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.recordTableService.Reload(r.Context()); err != nil {
		a.respondError(w, r, err)
		return
	}
	if err := a.recordBreakService.Reload(r.Context()); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RecordTableExportRequest struct {
	TableFormat
	// Columns holds the keys of the exported columns in their order, every column if empty.
	Columns []string `json:"columns" validate:"max=50"`
}

// recordTableExportHandler downloads the records of a record table as a CSV file.
//
//	@Summary		Export record table
//	@Description	Downloads the records of the record table with the given id as a CSV file with the keys of the columns as its header, which imports again without a mapping. The default is UTF-8 with a byte order mark, separated by semicolons, with every column.
//	@Tags			records
//	@Produce		text/csv
//	@Param			id			path	string	true	"Record table id"
//	@Param			encoding	query	string	false	"Encoding"	Enums(utf-8, windows-1252)
//	@Param			separator	query	string	false	"Separator"	Enums(semicolon, comma, tab)
//	@Param			columns		query	string	false	"Keys of the columns, comma-separated"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/record-tables/{id}/export [get]
func (a *API) recordTableExportHandler(w http.ResponseWriter, r *http.Request) {
	query := newQueryParser(r)
	req := RecordTableExportRequest{
		Encoding:  string(tabular.EncodingUTF8),
		Separator: "semicolon",
	}
	if encoding := query.optionalString("encoding"); encoding != nil {
		req.Encoding = *encoding
	}
	if separator := query.optionalString("separator"); separator != nil {
		req.Separator = *separator
	}
	if columns := query.optionalString("columns"); columns != nil {
		req.Columns = strings.Split(*columns, ",")
	}
	if query.err != nil {
		a.badRequestResponse(w, r, query.err)
		return
	}
	errs, err := a.validate(r.Context(), req)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
	if errs != nil {
		a.failedValidationResponse(w, r, errs)
		return
	}

	id := pathParam(r, "id")
	table, err := a.recordTableService.Get(id)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	download := &downloadWriter{
		writer:      w,
		contentType: "text/csv",
		filename:    table.Table.Abbreviation + ".csv",
	}
	err = a.recordTableService.Export(download, id, service.RecordTableExportInput{
		Format:  toTabularFormat(req.TableFormat),
		Columns: req.Columns,
	})
	if err == nil {
		return
	}
	if !download.started {
		a.respondError(w, r, err)
		return
	}
	// The headers and part of the table are already sent, so the client may only see a broken download
	a.logger.Warn().Err(err).Str("table", id).Msg("failed to export the record table")
}

// toRecordTableResponse maps a record table onto its response.
func toRecordTableResponse(table service.RecordTableState) RecordTableResponse {
	return RecordTableResponse{
		ID:           table.Table.ID,
		Name:         table.Table.Name,
		Abbreviation: table.Table.Abbreviation,
		Enabled:      table.Table.Enabled,
		RecordCount:  table.RecordCount,
	}
}
