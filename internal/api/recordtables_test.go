package api_test

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
)

func TestRecordTableAccess(t *testing.T) {
	t.Parallel()

	// storeTable creates a record table for a route that needs one and returns its path with the suffix
	storeTable := func(h *harness, suffix string) string {
		meet := newMeetFixture(h.Faker, 1)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))
		return "/api/record-tables/" + table.ID + suffix
	}

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/record-tables",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/record-tables/{id}",
			arrange: func(h *harness) (string, any) {
				return storeTable(h, ""), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/record-tables",
			arrange: func(h *harness) (string, any) {
				fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
				return "/api/record-tables", recordUpload(h.t, fixture.content, newRecordTableCreateRequest(h.Faker, fixture))
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusCreated,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/record-tables/{id}",
			arrange: func(h *harness) (string, any) {
				return storeTable(h, ""), newRecordTableUpdateRequest(h)
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/record-tables/{id}",
			arrange: func(h *harness) (string, any) {
				return storeTable(h, ""), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/record-tables/preview",
			arrange: func(h *harness) (string, any) {
				fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
				return "/api/record-tables/preview", recordUpload(h.t, fixture.content, api.RecordTablePreviewRequest{})
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/record-tables/{id}/import",
			arrange: func(h *harness) (string, any) {
				fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
				return storeTable(h, "/import"), recordUpload(h.t, fixture.content, api.RecordTableImportRequest{
					Mapping: fixture.mapping,
				})
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/record-tables/order",
			arrange: func(h *harness) (string, any) {
				path := storeTable(h, "")
				return "/api/record-tables/order", api.RecordTableReorderRequest{
					IDs: []string{
						strings.TrimPrefix(path, "/api/record-tables/"),
					},
				}
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/record-tables/reload",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/record-tables/{id}/export",
			arrange: func(h *harness) (string, any) {
				return storeTable(h, "/export"), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestRecordTableListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the tables in their order with the number of their records", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		first := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 2))
		second := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))

		response := h.Client.GET("/api/record-tables")

		list := response.AssertOK().Decode[api.RecordTableListResponse]()
		assert.Equal(t, []api.RecordTableResponse{
			first,
			second,
		}, list.Tables)
		assert.Equal(t, 2, list.Tables[0].RecordCount)
		assert.NotNil(t, list.LoadedAt)
	})

	t.Run("returns an empty list without tables", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/record-tables")

		response.AssertOK().AssertJSONPath("tables", []any{})
	})
}

func TestRecordTableGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))

		response := h.Client.GET("/api/record-tables/" + table.ID)

		response.AssertOK().AssertJSON(table)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/record-tables/" + h.Faker.LetterN(12))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestRecordTableCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates the table with the records of the file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 3)
		request := newRecordTableCreateRequest(h.Faker, fixture)

		response := h.asAdmin().POST("/api/record-tables", recordUpload(t, fixture.content, request))

		table := response.AssertCreated().Decode[api.RecordTableResponse]()
		assert.NotEmpty(t, table.ID)
		assert.Equal(t, api.RecordTableResponse{
			ID:           table.ID,
			Name:         request.Name,
			Abbreviation: request.Abbreviation,
			Enabled:      true,
			RecordCount:  3,
		}, table)
		records := h.Client.GET("/api/records").AssertOK().Decode[api.RecordListResponse]().Records
		for i := range fixture.records {
			fixture.records[i].TableID = table.ID
		}
		assert.Equal(t, fixture.records, records)
	})

	t.Run("translates the values of the file through the mapping", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		content := strings.Replace(fixture.content, ";LCM;", ";Lang;", 1)
		content = strings.Replace(content, "\r\nLCM;", "\r\nLang;", 1)
		request := newRecordTableCreateRequest(h.Faker, fixture)
		request.Mapping.Values = map[string]map[string]string{
			"course": {
				"Lang": "LCM",
			},
		}

		response := h.asAdmin().POST("/api/record-tables", recordUpload(t, content, request))

		response.AssertCreated()
		h.Client.GET("/api/records").AssertOK().AssertJSONPath("records.0.course", "LCM")
	})

	t.Run("reports the rows that fail and stores nothing", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 2)
		lines := strings.Split(fixture.content, "\r\n")
		fields := strings.Split(lines[2], ";")
		fields[4] = h.Faker.LetterN(4)
		lines[2] = strings.Join(fields, ";")
		content := strings.Join(lines, "\r\n")

		response := h.asAdmin().POST("/api/record-tables", recordUpload(t, content, newRecordTableCreateRequest(h.Faker, fixture)))

		assertFieldError(t, response, "rows.3.time", "swimtime")
		h.Client.GET("/api/record-tables").AssertOK().AssertJSONCount("tables", 0)
	})

	t.Run("reports a mapping without a required column", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		request := newRecordTableCreateRequest(h.Faker, fixture)
		delete(request.Mapping.Columns, "distance")

		response := h.asAdmin().POST("/api/record-tables", recordUpload(t, fixture.content, request))

		assertFieldError(t, response, "mapping.columns.distance", "required")
	})

	t.Run("reports a file that cannot be read", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)

		response := h.asAdmin().POST("/api/record-tables", recordUpload(t, "\r\n", newRecordTableCreateRequest(h.Faker, fixture)))

		assertFieldError(t, response, "file", "empty")
	})

	t.Run("rejects a file of another type", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		upload := recordUpload(t, fixture.content, newRecordTableCreateRequest(h.Faker, fixture))
		upload.Files[0].Name = "records.xlsx"

		response := h.asAdmin().POST("/api/record-tables", upload)

		assertFieldError(t, response, "file", "unsupported_type")
	})

	t.Run("rejects an upload without the request", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		upload := recordUpload(t, fixture.content, nil)
		delete(upload.Fields, "data")

		response := h.asAdmin().POST("/api/record-tables", upload)

		response.AssertBadRequest()
	})

	invalid := []struct {
		field  string
		tag    string
		change func(request *api.RecordTableCreateRequest)
	}{
		{
			field:  "name",
			tag:    "required",
			change: func(request *api.RecordTableCreateRequest) { request.Name = "" },
		},
		{
			field:  "name",
			tag:    "max",
			change: func(request *api.RecordTableCreateRequest) { request.Name = strings.Repeat("x", 101) },
		},
		{
			field:  "abbreviation",
			tag:    "required",
			change: func(request *api.RecordTableCreateRequest) { request.Abbreviation = "" },
		},
		{
			field:  "abbreviation",
			tag:    "max",
			change: func(request *api.RecordTableCreateRequest) { request.Abbreviation = strings.Repeat("x", 11) },
		},
		{
			field:  "mapping.format.encoding",
			tag:    "oneof",
			change: func(request *api.RecordTableCreateRequest) { request.Mapping.Format.Encoding = "utf-16" },
		},
		{
			field:  "mapping.format.separator",
			tag:    "oneof",
			change: func(request *api.RecordTableCreateRequest) { request.Mapping.Format.Separator = "|" },
		},
	}

	for _, tt := range invalid {
		t.Run("rejects an invalid "+tt.field+" with "+tt.tag, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
			request := newRecordTableCreateRequest(h.Faker, fixture)
			tt.change(&request)

			response := h.asAdmin().POST("/api/record-tables", recordUpload(t, fixture.content, request))

			assertFieldError(t, response, tt.field, tt.tag)
		})
	}
}

func TestRecordTableUpdateHandler(t *testing.T) {
	t.Parallel()

	t.Run("replaces the name, the abbreviation, and the state and keeps the records", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 2))
		request := newRecordTableUpdateRequest(h)

		response := h.asAdmin().PUT("/api/record-tables/"+table.ID, request)

		response.AssertOK().AssertJSON(api.RecordTableResponse{
			ID:           table.ID,
			Name:         request.Name,
			Abbreviation: request.Abbreviation,
			Enabled:      request.Enabled,
			RecordCount:  2,
		})
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().PUT("/api/record-tables/"+h.Faker.LetterN(12), newRecordTableUpdateRequest(h))

		response.AssertNotFound()
	})

	t.Run("rejects an invalid name with required", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))
		request := newRecordTableUpdateRequest(h)
		request.Name = ""

		response := h.asAdmin().PUT("/api/record-tables/"+table.ID, request)

		assertFieldError(t, response, "name", "required")
	})
}

func TestRecordTableDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the table and its records", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))

		response := h.asAdmin().DELETE("/api/record-tables/" + table.ID)

		response.AssertNoContent()
		h.Client.GET("/api/record-tables/" + table.ID).AssertNotFound()
		h.Client.GET("/api/records").AssertOK().AssertJSONCount("records", 0)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE("/api/record-tables/" + h.Faker.LetterN(12))

		response.AssertNotFound()
	})
}

func TestRecordTablePreviewHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the start of the file with a guessed mapping", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		fixture := newRecordTableFixture(h.Faker, meet, 50, 2)

		response := h.asAdmin().POST("/api/record-tables/preview", recordUpload(t, fixture.content, api.RecordTablePreviewRequest{}))

		preview := response.AssertOK().Decode[api.ImportPreviewResponse]()
		assert.Equal(t, api.TableFormat{
			Encoding:  "windows-1252",
			Separator: "semicolon",
		}, preview.Format)
		assert.Equal(t, []string{"course", "gender", "distance", "stroke", "time", "holder"}, preview.Header)
		assert.Len(t, preview.Sample, 2)
		assert.Equal(t, "course", preview.Mapping.Columns["course"])
		assert.Equal(t, "holder", preview.Mapping.Columns["name"], "a column is also found by an alias")
		var stroke api.ImportColumnResponse
		for _, column := range preview.Columns {
			if column.Key == "stroke" {
				stroke = column
			}
		}
		assert.Equal(t, api.ImportColumnResponse{
			Key:      "stroke",
			Required: true,
			Choices:  []string{},
			Suggestions: []string{
				meet.style.Abbreviation,
			},
		}, stroke, "the styles of the meet are suggested")
	})

	t.Run("reads the file in the given format", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/record-tables/preview", recordUpload(t, "a;b,c\r\n", api.RecordTablePreviewRequest{
			Format: api.TableFormat{
				Encoding:  "utf-8",
				Separator: "comma",
			},
		}))

		response.AssertOK().AssertJSONPath("header", []any{"a;b", "c"}).AssertJSONPath("format.encoding", "utf-8")
	})

	t.Run("reports a file that cannot be read", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/record-tables/preview", recordUpload(t, "\r\n", api.RecordTablePreviewRequest{}))

		assertFieldError(t, response, "file", "empty")
	})
}

func TestRecordTableImportHandler(t *testing.T) {
	t.Parallel()

	t.Run("replaces the records of the table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		table := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))
		fixture := newRecordTableFixture(h.Faker, meet, 50, 3)

		response := h.asAdmin().POST("/api/record-tables/"+table.ID+"/import", recordUpload(t, fixture.content, api.RecordTableImportRequest{
			Mapping: fixture.mapping,
		}))

		response.AssertOK().AssertJSONPath("recordCount", 3).AssertJSONPath("name", table.Name)
		h.Client.GET("/api/records").AssertOK().AssertJSONCount("records", 3)
	})

	t.Run("keeps the records when the file fails", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		fixture := newRecordTableFixture(h.Faker, meet, 50, 2)
		table := h.storeRecordTable(fixture)

		response := h.asAdmin().POST("/api/record-tables/"+table.ID+"/import", recordUpload(t, "\r\n", api.RecordTableImportRequest{
			Mapping: fixture.mapping,
		}))

		response.AssertUnprocessable()
		h.Client.GET("/api/records").AssertOK().AssertJSONCount("records", 2)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		path := "/api/record-tables/" + h.Faker.LetterN(12) + "/import"

		response := h.asAdmin().POST(path, recordUpload(t, fixture.content, api.RecordTableImportRequest{
			Mapping: fixture.mapping,
		}))

		response.AssertNotFound()
	})
}

func TestRecordTableReorderHandler(t *testing.T) {
	t.Parallel()

	t.Run("puts the tables in the given order", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		first := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))
		second := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))

		response := h.asAdmin().PUT("/api/record-tables/order", api.RecordTableReorderRequest{
			IDs: []string{
				second.ID,
				first.ID,
			},
		})

		response.AssertNoContent()
		h.Client.GET("/api/record-tables").AssertOK().AssertJSONPath("tables.0.id", second.ID).AssertJSONPath("tables.1.id", first.ID)
	})

	t.Run("rejects ids that leave out a table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		first := h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))
		h.storeRecordTable(newRecordTableFixture(h.Faker, meet, 50, 1))

		response := h.asAdmin().PUT("/api/record-tables/order", api.RecordTableReorderRequest{
			IDs: []string{
				first.ID,
			},
		})

		assertFieldError(t, response, "ids", "invalid")
	})

	t.Run("rejects an id named twice with unique", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))

		response := h.asAdmin().PUT("/api/record-tables/order", api.RecordTableReorderRequest{
			IDs: []string{
				first.ID,
				first.ID,
			},
		})

		assertFieldError(t, response, "ids", "unique")
	})
}

func TestRecordTableReloadHandler(t *testing.T) {
	t.Parallel()

	t.Run("reads a table changed in the storage", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 3)
		require.NoError(t, h.Storage.Put(t.Context(), "records/manual.csv", strings.NewReader(fixture.content), -1))

		response := h.asAdmin().POST("/api/record-tables/reload", nil)

		response.AssertNoContent()
		h.Client.GET("/api/record-tables").AssertOK().
			AssertJSONPath("tables.0.id", "manual").
			AssertJSONPath("tables.0.enabled", false).
			AssertJSONPath("tables.0.recordCount", 3)
	})

	t.Run("reads the records set at the meet again", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.Client.GET("/api/record-breaks").AssertOK().AssertJSON([]api.RecordBreakResponse{})
		id := h.Faker.LetterN(12)
		require.NoError(t, h.Storage.Put(t.Context(), "records/breaks.json", strings.NewReader(`[{"id":"`+id+`"}]`), -1))

		response := h.asAdmin().POST("/api/record-tables/reload", nil)

		response.AssertNoContent()
		h.Client.GET("/api/record-breaks").AssertOK().AssertJSONPath("0.id", id)
	})

	t.Run("rejects a table that cannot be read and keeps the previous tables", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))
		content := "course;gender;distance;stroke;time\r\nLCM;W;x;FR;1:00.00\r\n"
		require.NoError(t, h.Storage.Put(t.Context(), "records/manual.csv", strings.NewReader(content), -1))

		response := h.asAdmin().POST("/api/record-tables/reload", nil)

		body := response.AssertBadRequest().Decode[api.ErrorResponse]()
		assert.Contains(t, body.Message, "manual.csv, line 2, column distance", "the message names the file and the position")
		h.Client.GET("/api/record-tables").AssertOK().AssertJSONCount("tables", 1)
	})

	t.Run("limits the reloads of a client", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		for range 10 {
			admin.POST("/api/record-tables/reload", nil).AssertNoContent()
		}

		response := admin.POST("/api/record-tables/reload", nil)

		response.AssertTooManyRequests()
	})
}

func TestRecordTableExportHandler(t *testing.T) {
	t.Parallel()

	t.Run("downloads the records in the format of the tables", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 2)
		table := h.storeRecordTable(fixture)

		response := h.Client.GET("/api/record-tables/" + table.ID + "/export")

		response.AssertOK().AssertHeader("Content-Type", "text/csv")
		assert.Contains(t, response.Header.Get("Content-Disposition"), table.Abbreviation+".csv")
		body := response.Body
		assert.True(t, bytes.HasPrefix(body, []byte("\xEF\xBB\xBF")), "a byte order mark lets spreadsheets read UTF-8")
		lines := strings.Split(strings.TrimPrefix(string(body), "\xEF\xBB\xBF"), "\r\n")
		assert.Equal(t, "course;gender;relayCount;distance;stroke;ageFrom;ageTo;time;approved;name;club;place;date;birthYear", lines[0])
		assert.Len(t, lines, 4, "the header, two records, and the final line break")
	})

	t.Run("downloads the selected columns in the given format", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		fixture := newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1)
		table := h.storeRecordTable(fixture)

		response := h.Client.GET("/api/record-tables/" + table.ID + "/export?encoding=windows-1252&separator=comma&columns=distance,name")

		response.AssertOK()
		assert.Equal(t, "distance,name\r\n"+strconv.Itoa(fixture.records[0].Distance)+","+fixture.records[0].Name+"\r\n", string(response.Body))
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/record-tables/" + h.Faker.LetterN(12) + "/export")

		response.AssertNotFound()
	})

	invalid := []struct {
		field string
		tag   string
		query string
	}{
		{
			field: "encoding",
			tag:   "oneof",
			query: "encoding=utf-16",
		},
		{
			field: "separator",
			tag:   "oneof",
			query: "separator=pipe",
		},
		{
			field: "columns",
			tag:   "oneof",
			query: "columns=other",
		},
	}

	for _, tt := range invalid {
		t.Run("rejects an invalid "+tt.field+" with "+tt.tag, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			table := h.storeRecordTable(newRecordTableFixture(h.Faker, newMeetFixture(h.Faker, 1), 50, 1))

			response := h.Client.GET("/api/record-tables/" + table.ID + "/export?" + tt.query)

			assertFieldError(t, response, tt.field, tt.tag)
		})
	}
}

// newRecordTableUpdateRequest returns new random names and a random state of a record table.
func newRecordTableUpdateRequest(h *harness) api.RecordTableUpdateRequest {
	return api.RecordTableUpdateRequest{
		Name:         h.Faker.LetterN(12),
		Abbreviation: strings.ToUpper(h.Faker.LetterN(3)),
		Enabled:      h.Faker.Bool(),
	}
}
