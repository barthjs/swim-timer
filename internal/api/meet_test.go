package api_test

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestMeetAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/meet",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/events",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/events/{number}",
			arrange: func(h *harness) (string, any) {
				meet := newMeetFixture(h.Faker, 1)
				h.loadMeet(meet)
				return meetEventPath(meet.event.Number), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/events/{number}/heats/{heat}",
			arrange: func(h *harness) (string, any) {
				meet := newMeetFixture(h.Faker, 1)
				h.loadMeet(meet)
				return meetHeatPath(meet.event.Number, 1), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/competitors",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/start-entries",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meet/files",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/meet/files/{file}",
			arrange: func(h *harness) (string, any) {
				meet := newMeetFixture(h.Faker, 1)
				body := apitest.Multipart{
					Files: []apitest.File{
						{
							Field:   "file",
							Name:    "LSTSTYLE.TXT",
							Content: windows1252(h.t, meet.csv()[domain.MeetFileStyles]),
						},
					},
				}
				return meetFilePath(domain.MeetFileStyles), body
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/meet/files",
			arrange: func(h *harness) (string, any) {
				meet := newMeetFixture(h.Faker, 1)
				body := apitest.Multipart{
					Files: meet.files(h.t),
				}
				return "/api/meet/files", body
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/meet/reload",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
	})
}

func TestMeetOverviewGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the meet", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET("/api/meet")

		response.AssertOK().
			AssertJSONCount("events", 1).
			AssertJSONCount("startEntries", 2*meetLaneCount).
			AssertJSONCount("competitors", 2*meetLaneCount).
			AssertJSONPath("events.0.number", meet.event.Number).
			AssertJSONPath("events.0.heatCount", 2).
			AssertJSONPath("events.0.name", meet.displayName()).
			AssertJSONPath("events.0.distance", map[string]any{
				"name":       meet.distance.Name,
				"length":     meet.distance.Length,
				"relayCount": 1,
			}).
			AssertJSONPath("events.0.style.abbreviation", meet.style.Abbreviation).
			AssertJSONPath("events.0.category.name", meet.category.Name).
			AssertJSONPath("events.0.time", meet.event.Time).
			AssertJSONPath("startEntries.0.heat", 1)
		overview := response.Decode[struct {
			LoadedAt string `json:"loadedAt"`
			Events   []struct {
				Date string `json:"date"`
			} `json:"events"`
			Competitors []struct {
				ID        int    `json:"id"`
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
				Club      string `json:"club"`
			} `json:"competitors"`
		}]()
		assert.NotEmpty(t, overview.LoadedAt)
		assert.Equal(t, domain.ParseExportDate(meet.event.Date), overview.Events[0].Date, "the export date as an ISO date")
		expected := meet.competitor(2, 4)
		found := false
		for _, competitor := range overview.Competitors {
			if competitor.ID == expected.ID {
				found = true
				assert.Equal(t, expected.FirstName, competitor.FirstName)
				assert.Equal(t, expected.LastName, competitor.LastName)
				assert.Equal(t, expected.AbNat, competitor.Club)
			}
		}
		assert.True(t, found, "every competitor is listed")
	})

	t.Run("returns an empty meet before loading", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/meet")

		// Missing export files are an empty meet, not an error, so loadedAt is set as well
		response.AssertOK().AssertJSONPath("events", []any{}).AssertJSONPath("startEntries", []any{}).AssertJSONPath("competitors", []any{})
	})
}

func TestMeetEventListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the events", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET("/api/meet/events")

		response.AssertOK().
			AssertJSONCount("events", 1).
			AssertJSONPath("events.0.number", meet.event.Number).
			AssertJSONPath("events.0.heatCount", 2).
			AssertJSONPath("events.0.name", meet.displayName())
		list := response.Decode[struct {
			LoadedAt string `json:"loadedAt"`
		}]()
		assert.NotEmpty(t, list.LoadedAt)
	})

	t.Run("returns an empty list before loading", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/meet/events")

		response.AssertOK().AssertJSONPath("events", []any{})
	})
}

func TestMeetEventGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET(meetEventPath(meet.event.Number))

		response.AssertOK().
			AssertJSONPath("event.number", meet.event.Number).
			AssertJSONPath("event.phase", meet.phase.ID).
			AssertJSONPath("event.heatCount", 2).
			AssertJSONPath("event.name", meet.displayName()).
			AssertJSONPath("event.style.abbreviation", meet.style.Abbreviation)
		event := response.Decode[struct {
			LoadedAt string `json:"loadedAt"`
		}]()
		assert.NotEmpty(t, event.LoadedAt)
	})

	t.Run("rejects an invalid number", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/meet/events/" + h.Faker.LetterN(8))

		response.AssertBadRequest()
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)

		response := h.Client.GET(meetEventPath(meet.event.Number + 1))

		response.AssertNotFound()
	})
}

func TestMeetHeatGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the start entries of the heat with their competitors", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)
		expected := meet.competitor(2, 1)

		response := h.Client.GET(meetHeatPath(meet.event.Number, 2))

		response.AssertOK().
			AssertJSONPath("event.number", meet.event.Number).
			AssertJSONPath("heat", 2).
			AssertJSONCount("startEntries", meetLaneCount).
			AssertJSONPath("startEntries.0.lane", 1).
			AssertJSONPath("startEntries.0.relayPosition", 0).
			AssertJSONPath("startEntries.0.competitor.id", expected.ID).
			AssertJSONPath("startEntries.0.competitor.firstName", expected.FirstName).
			AssertJSONPath("startEntries.0.competitor.lastName", expected.LastName).
			AssertJSONPath("startEntries.0.competitor.club", expected.AbNat)
		heat := response.Decode[struct {
			LoadedAt string `json:"loadedAt"`
		}]()
		assert.NotEmpty(t, heat.LoadedAt)
	})

	t.Run("rejects an invalid path parameter", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path func(f *gofakeit.Faker, number int) string
		}{
			{
				name: "number",
				path: func(f *gofakeit.Faker, _ int) string {
					return "/api/meet/events/" + f.LetterN(8) + "/heats/1"
				},
			},
			{
				name: "heat",
				path: func(f *gofakeit.Faker, number int) string {
					return meetEventPath(number) + "/heats/" + f.LetterN(8)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				meet := newMeetFixture(h.Faker, 1)
				h.loadMeet(meet)

				response := h.Client.GET(tt.path(h.Faker, meet.event.Number))

				response.AssertBadRequest()
			})
		}
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)

		response := h.Client.GET(meetHeatPath(meet.event.Number+1, 1))

		response.AssertNotFound()
	})

	t.Run("returns not found for a heat after the last heat", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET(meetHeatPath(meet.event.Number, meet.event.HeatCount+1))

		response.AssertNotFound()
	})
}

func TestMeetCompetitorListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the competitors", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET("/api/meet/competitors")

		response.AssertOK().AssertJSONCount("competitors", 2*meetLaneCount)
		list := response.Decode[struct {
			LoadedAt    string `json:"loadedAt"`
			Competitors []struct {
				ID        int    `json:"id"`
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
				Club      string `json:"club"`
			} `json:"competitors"`
		}]()
		assert.NotEmpty(t, list.LoadedAt)
		expected := meet.competitor(1, 3)
		found := false
		for _, competitor := range list.Competitors {
			if competitor.ID == expected.ID {
				found = true
				assert.Equal(t, expected.FirstName, competitor.FirstName)
				assert.Equal(t, expected.LastName, competitor.LastName)
				assert.Equal(t, expected.AbNat, competitor.Club)
			}
		}
		assert.True(t, found, "every competitor is listed")
	})

	t.Run("returns an empty list before loading", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/meet/competitors")

		response.AssertOK().AssertJSONPath("competitors", []any{})
	})
}

func TestMeetStartEntryListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the start entries", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)
		h.loadMeet(meet)

		response := h.Client.GET("/api/meet/start-entries")

		response.AssertOK().
			AssertJSONCount("startEntries", 2*meetLaneCount).
			AssertJSONPath("startEntries.0.event", meet.event.Number).
			AssertJSONPath("startEntries.0.phase", meet.phase.ID).
			AssertJSONPath("startEntries.0.heat", 1).
			AssertJSONPath("startEntries.0.lane", 1).
			AssertJSONPath("startEntries.0.competitorId", meet.competitor(1, 1).ID)
		list := response.Decode[struct {
			LoadedAt string `json:"loadedAt"`
		}]()
		assert.NotEmpty(t, list.LoadedAt)
	})

	t.Run("returns an empty list before loading", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/meet/start-entries")

		response.AssertOK().AssertJSONPath("startEntries", []any{})
	})
}

func TestMeetFileListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the files", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.loadMeet(newMeetFixture(h.Faker, 1))

		response := h.asAdmin().GET("/api/meet/files")

		response.AssertOK()
		for i, file := range domain.MeetFiles() {
			response.AssertJSONPath(strconv.Itoa(i), map[string]any{"file": string(file), "filename": string(file) + ".TXT", "available": true})
		}
	})

	t.Run("reports missing files as unavailable", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/meet/files")

		response.AssertOK().AssertJSONCount("", len(domain.MeetFiles()))
		for i, file := range domain.MeetFiles() {
			response.AssertJSONPath(strconv.Itoa(i), map[string]any{"file": string(file), "available": false})
		}
	})
}

func TestMeetFileSaveHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)

		h.asAdmin().PUT(meetFilePath(domain.MeetFileCategories), apitest.Multipart{
			Files: []apitest.File{
				{
					Field:   "file",
					Name:    h.Faker.LetterN(6) + ".txt",
					Content: windows1252(t, meet.csv()[domain.MeetFileCategories]),
				},
			},
		}).AssertNoContent()

		stored, err := h.Storage.Exists(t.Context(), "meet/LSTCAT.TXT")
		require.NoError(t, err)
		assert.True(t, stored)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*apitest.Multipart, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name: "missing file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files = nil
					m.Fields = map[string]string{"comment": f.Word()}
				},
				field: "file",
				tag:   "required",
			},
			{
				name:   "wrong extension",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) { m.Files[0].Name = f.LetterN(6) + ".xlsx" },
				field:  "file",
				tag:    "unsupported_type",
			},
			{
				name: "binary content",
				change: func(m *apitest.Multipart, _ *gofakeit.Faker) {
					m.Files[0].Content = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
				},
				field: "file",
				tag:   "unsupported_type",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				upload := apitest.Multipart{
					Files: []apitest.File{
						{
							Field:   "file",
							Name:    h.Faker.LetterN(6) + ".txt",
							Content: []byte(h.Faker.LoremIpsumSentence(6)),
						},
					},
				}
				tt.change(&upload, h.Faker)

				response := h.asAdmin().PUT(meetFilePath(domain.MeetFileStyles), upload)

				assertFieldError(t, response, tt.field, tt.tag)
			})
		}
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		t.Parallel()

		for _, tt := range malformedForms() {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				boundary := h.Faker.LetterN(16)
				admin := h.asAdmin().WithHeader("Content-Type", tt.contentType(boundary))

				response := admin.PUT(meetFilePath(domain.MeetFileStyles), strings.NewReader(tt.body(h, boundary)))

				response.AssertBadRequest().AssertJSONPath("code", "bad_request")
			})
		}
	})

	t.Run("rejects an oversized body", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		body, contentType := endlessUpload(h, "file")

		response := h.asAdmin().WithHeader("Content-Type", contentType).PUT(meetFilePath(domain.MeetFileStyles), body)

		assertFieldError(t, response, "file", "too_large")
	})

	t.Run("returns not found for an unknown file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().PUT("/api/meet/files/LSTUNKNOWN", apitest.Multipart{
			Files: []apitest.File{
				{
					Field:   "file",
					Name:    "a.txt",
					Content: []byte(h.Faker.LoremIpsumSentence(6)),
				},
			},
		})

		response.AssertStatus(http.StatusNotFound)
	})
}

func TestMeetFileUploadHandler(t *testing.T) {
	t.Parallel()

	t.Run("reports every file and stores the valid ones", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		unknownName := h.Faker.LetterN(8) + ".txt"
		files := []apitest.File{
			{
				Field:   "files",
				Name:    "lstcat.txt",
				Content: windows1252(t, meet.csv()[domain.MeetFileCategories]),
			},
			{
				Field:   "files",
				Name:    unknownName,
				Content: []byte(h.Faker.LoremIpsumSentence(6)),
			},
			{
				Field:   "files",
				Name:    "LSTLONG.pdf",
				Content: []byte(h.Faker.LoremIpsumSentence(6)),
			},
			{
				Field:   "files",
				Name:    "LSTSTYLE.txt",
				Content: []byte{0x00, 0x01, 0x02, 0xff},
			},
			{
				Field:   "files",
				Name:    "LSTRACE.csv",
				Content: windows1252(t, meet.csv()[domain.MeetFileEvents]),
			},
		}

		response := h.asAdmin().POST("/api/meet/files", apitest.Multipart{
			Files: files,
		})

		response.AssertOK().AssertJSON([]map[string]string{
			{"filename": "lstcat.txt", "file": "LSTCAT"},
			{"filename": unknownName, "error": "unknown_file"},
			{"filename": "LSTLONG.pdf", "file": "LSTLONG", "error": "unsupported_type"},
			{"filename": "LSTSTYLE.txt", "file": "LSTSTYLE", "error": "unsupported_type"},
			{"filename": "LSTRACE.csv", "file": "LSTRACE", "error": "unsupported_type"},
		})
		stored, err := h.Storage.Exists(t.Context(), "meet/LSTCAT.TXT")
		require.NoError(t, err)
		assert.True(t, stored, "the valid file is stored despite the others")
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/meet/files", apitest.Multipart{
			Fields: map[string]string{"comment": h.Faker.Word()},
		})

		assertFieldError(t, response, "files", "required")
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		t.Parallel()

		for _, tt := range malformedForms() {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				boundary := h.Faker.LetterN(16)
				admin := h.asAdmin().WithHeader("Content-Type", tt.contentType(boundary))

				response := admin.POST("/api/meet/files", strings.NewReader(tt.body(h, boundary)))

				response.AssertBadRequest().AssertJSONPath("code", "bad_request")
			})
		}
	})

	t.Run("rejects an oversized body", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		body, contentType := endlessUpload(h, "files")

		response := h.asAdmin().WithHeader("Content-Type", contentType).POST("/api/meet/files", body)

		assertFieldError(t, response, "files", "too_large")
	})

	t.Run("reports a file over the size limit of a single file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		// One byte more than the 10 MiB a single meet file may have, far below the limit of the request
		oversized := bytes.Repeat([]byte("a"), 10<<20+1)
		files := []apitest.File{
			{
				Field:   "files",
				Name:    "LSTSTYLE.TXT",
				Content: oversized,
			},
			{
				Field:   "files",
				Name:    "LSTCAT.TXT",
				Content: windows1252(t, meet.csv()[domain.MeetFileCategories]),
			},
		}

		response := h.asAdmin().POST("/api/meet/files", apitest.Multipart{
			Files: files,
		})

		response.AssertOK().AssertJSON([]map[string]string{
			{"filename": "LSTSTYLE.TXT", "file": "LSTSTYLE", "error": "too_large"},
			{"filename": "LSTCAT.TXT", "file": "LSTCAT"},
		})
		oversizedStored, err := h.Storage.Exists(t.Context(), "meet/LSTSTYLE.TXT")
		require.NoError(t, err)
		assert.False(t, oversizedStored)
		stored, err := h.Storage.Exists(t.Context(), "meet/LSTCAT.TXT")
		require.NoError(t, err)
		assert.True(t, stored, "the other file is stored despite the oversized one")
	})

	t.Run("keeps the loaded meet until a reload", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)

		h.asAdmin().POST("/api/meet/files", apitest.Multipart{
			Files: meet.files(t),
		}).AssertOK()

		h.Client.GET("/api/meet").AssertJSONCount("events", 0)
	})
}

// malformedForm is a request body that is no readable multipart form.
type malformedForm struct {
	name        string
	contentType func(boundary string) string
	body        func(h *harness, boundary string) string
}

// malformedForms returns bodies that cannot be read as a multipart form.
func malformedForms() []malformedForm {
	return []malformedForm{
		{
			name:        "not multipart",
			contentType: func(string) string { return "application/json" },
			body: func(h *harness, _ string) string {
				return `{"file":"` + h.Faker.Word() + `"}`
			},
		},
		{
			name:        "missing boundary",
			contentType: func(string) string { return "multipart/form-data" },
			body: func(h *harness, _ string) string {
				return h.Faker.LoremIpsumSentence(4)
			},
		},
		{
			name:        "unterminated part",
			contentType: func(boundary string) string { return "multipart/form-data; boundary=" + boundary },
			body: func(h *harness, boundary string) string {
				return "--" + boundary + "\r\nContent-Disposition: form-data; name=\"comment\"\r\n\r\n" + h.Faker.Word()
			},
		},
	}
}

// endlessUpload returns a multipart body and its content type with a file in field
// that is larger than every body limit.
func endlessUpload(h *harness, field string) (io.Reader, string) {
	boundary := h.Faker.LetterN(16)
	header := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"" + field + "\"; filename=\"LSTSTYLE.TXT\"\r\n" +
		"Content-Type: text/plain\r\n\r\n"
	content := io.LimitReader(endlessReader{}, 1<<30)

	return io.MultiReader(strings.NewReader(header), content), "multipart/form-data; boundary=" + boundary
}

// endlessReader yields an endless stream of letters.
type endlessReader struct{}

// Read fills p with letters.
func (endlessReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}

func TestMeetReloadHandler(t *testing.T) {
	t.Parallel()

	t.Run("replaces the meet", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.loadMeet(newMeetFixture(h.Faker, 1))
		meet := newMeetFixture(h.Faker, 2)
		admin := h.asAdmin()
		admin.POST("/api/meet/files", apitest.Multipart{
			Files: meet.files(t),
		}).AssertOK()

		response := admin.POST("/api/meet/reload", nil)

		response.AssertNoContent()
		h.Client.GET("/api/meet").
			AssertJSONCount("events", 1).
			AssertJSONCount("startEntries", 2*meetLaneCount).
			AssertJSONPath("events.0.number", meet.event.Number).
			AssertJSONPath("events.0.name", meet.displayName())
	})

	t.Run("rejects an invalid meet file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 1)
		h.loadMeet(meet)
		admin := h.asAdmin()
		// A start list whose heat column holds text cannot be read
		broken := "event;round;heat;lane;relais;idBib\r\n" +
			strconv.Itoa(meet.event.Number) + ";0;" + h.Faker.LetterN(4) + ";1;0;1;\r\n"
		admin.PUT(meetFilePath(domain.MeetFileStartEntries), apitest.Multipart{
			Files: []apitest.File{
				{
					Field:   "file",
					Name:    "LSTSTART.TXT",
					Content: windows1252(t, broken),
				},
			},
		}).AssertNoContent()

		response := admin.POST("/api/meet/reload", nil)

		response.AssertStatus(http.StatusBadRequest).AssertJSONPath("code", "bad_request")
		var body api.ErrorResponse
		response.JSON(&body)
		assert.Contains(t, body.Message, "LSTSTART.TXT, line 2, column heat", "the message names the file and the position")
		h.Client.GET("/api/meet").AssertJSONCount("startEntries", meetLaneCount).AssertJSONPath("events.0.number", meet.event.Number)
	})

	t.Run("is rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		for range 10 {
			admin.POST("/api/meet/reload", nil).AssertNoContent()
		}

		response := admin.POST("/api/meet/reload", nil)

		response.AssertTooManyRequests().AssertJSONPath("code", "rate_limited")
		assert.NotEmpty(t, response.Header.Get("Retry-After"))
	})
}

// meetFilePath returns the API path of a single meet file.
func meetFilePath(file domain.MeetFile) string {
	return "/api/meet/files/" + string(file)
}

// meetHeatPath returns the path of a heat of an event, counted from 1.
func meetHeatPath(number, heat int) string {
	return meetEventPath(number) + "/heats/" + strconv.Itoa(heat)
}

// meetEventPath returns the path of an event.
func meetEventPath(number int) string {
	return "/api/meet/events/" + strconv.Itoa(number)
}
