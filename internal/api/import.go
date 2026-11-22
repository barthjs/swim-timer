package api

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/barthjs/swim-timer/internal/tabular"
)

var (
	// allowedImportUploadTypes are the detected content types an imported table may have.
	allowedImportUploadTypes = []string{"text/plain"}
	// allowedImportUploadExtensions are the file extensions an imported table may have, compared in lowercase.
	allowedImportUploadExtensions = []string{".csv", ".tsv", ".txt"}
)

// tableSeparators maps the names of the separators of the API to their characters,
// since a tab cannot be named in a validation rule.
var tableSeparators = map[string]string{
	"semicolon": ";",
	"comma":     ",",
	"tab":       "\t",
}

type TableFormat struct {
	// Encoding is detected on import when empty.
	Encoding string `json:"encoding" validate:"omitempty,oneof=utf-8 windows-1252"`
	// Separator is detected on import when empty.
	Separator string `json:"separator" validate:"omitempty,oneof=semicolon comma tab"`
}

type ImportMapping struct {
	Format TableFormat `json:"format"`
	// Columns holds the header of the file by the key of a column.
	Columns map[string]string `json:"columns" validate:"max=100"`
	// Values translates the values of the file into the values of a column, by its key and the value in the file.
	Values map[string]map[string]string `json:"values" validate:"max=100"`
}

type ImportColumnResponse struct {
	Key      string `json:"key"`
	Required bool   `json:"required"`
	// Choices are the values the column accepts, empty for a column that accepts any text.
	Choices []string `json:"choices"`
	// Suggestions are further values the values of the file can be translated into, while the column accepts any text.
	Suggestions []string `json:"suggestions"`
}

type ImportPreviewResponse struct {
	Format  TableFormat   `json:"format"`
	Header  []string      `json:"header"`
	Sample  [][]string    `json:"sample"`
	Mapping ImportMapping `json:"mapping"`
	// Values holds the distinct values of every header with few of them, so they can be translated.
	Values  map[string][]string    `json:"values"`
	Columns []ImportColumnResponse `json:"columns"`
}

// readImportUpload reads the uploaded table of the "file" field and the JSON request of the "data" field into T,
// or writes the failure and returns false.
func (a *API) readImportUpload[T any](w http.ResponseWriter, r *http.Request, maxSizeMB int) ([]byte, T, bool) {
	var req T
	content, filename, ok := a.readUpload(w, r, maxSizeMB, allowedImportUploadTypes)
	if !ok {
		return nil, req, false
	}
	if !slices.Contains(allowedImportUploadExtensions, strings.ToLower(filepath.Ext(filename))) {
		a.failedValidationResponse(w, r, FieldErrors{"file": {
			{
				Tag: "unsupported_type",
			},
		}})
		return nil, req, false
	}

	if err := decodeFormJSON(r.FormValue("data"), &req); err != nil {
		a.badRequestResponse(w, r, err)
		return nil, req, false
	}
	errs, err := a.validate(r.Context(), req)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return nil, req, false
	}
	if errs != nil {
		a.failedValidationResponse(w, r, errs)
		return nil, req, false
	}
	return content, req, true
}

// errEmptyFormData reports a multipart form without the JSON of its request.
var errEmptyFormData = errors.New("the field data must hold the request as JSON")

// decodeFormJSON decodes the JSON of a form field into dst with the rules of readJSON.
func decodeFormJSON(data string, dst any) error {
	if strings.TrimSpace(data) == "" {
		return errEmptyFormData
	}
	err := json.Unmarshal([]byte(data), dst, json.RejectUnknownMembers(true), jsonv1.FormatDurationAsNano(true))
	if err == nil {
		return nil
	}
	if syntacticError, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return describeSyntacticError(syntacticError)
	}
	if semanticError, ok := errors.AsType[*json.SemanticError](err); ok {
		return describeSemanticError(semanticError)
	}
	return err
}

// toTabularMapping maps a mapping of the API onto the mapping of a table.
func toTabularMapping(mapping ImportMapping) tabular.Mapping {
	return tabular.Mapping{
		Format:  toTabularFormat(mapping.Format),
		Columns: mapping.Columns,
		Values:  mapping.Values,
	}
}

// toTabularFormat maps a format of the API onto the format of a table.
func toTabularFormat(format TableFormat) tabular.Format {
	return tabular.Format{
		Encoding:  tabular.Encoding(format.Encoding),
		Separator: tableSeparators[format.Separator],
	}
}

// toImportPreviewResponse maps the preview of a table and the columns of its target onto its response.
func toImportPreviewResponse(preview tabular.Preview, columns []tabular.ColumnInfo) ImportPreviewResponse {
	resp := ImportPreviewResponse{
		Format: toTableFormat(preview.Format),
		Header: preview.Header,
		Sample: preview.Sample,
		Mapping: ImportMapping{
			Format:  toTableFormat(preview.Mapping.Format),
			Columns: preview.Mapping.Columns,
			Values:  preview.Mapping.Values,
		},
		Values:  preview.Values,
		Columns: make([]ImportColumnResponse, 0, len(columns)),
	}
	for _, column := range columns {
		resp.Columns = append(resp.Columns, ImportColumnResponse{
			Key:         column.Key,
			Required:    column.Required,
			Choices:     column.Choices,
			Suggestions: column.Suggestions,
		})
	}
	return resp
}

// toTableFormat maps the format of a table onto its format in the API.
func toTableFormat(format tabular.Format) TableFormat {
	result := TableFormat{
		Encoding: string(format.Encoding),
	}
	for name, separator := range tableSeparators {
		if separator == format.Separator {
			result.Separator = name
		}
	}
	return result
}
