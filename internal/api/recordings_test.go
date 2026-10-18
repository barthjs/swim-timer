package api_test

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/media"
)

func TestRecordingAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/recordings",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/recordings",
			arrange: func(h *harness) (string, any) {
				return "/api/recordings", newRecordingUpload(h.t, h.Faker)
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusCreated,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/recordings/{name}",
			arrange: func(h *harness) (string, any) {
				return recordingPath(h.storeRecording().Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/recordings/{name}",
			arrange: func(h *harness) (string, any) {
				return recordingPath(h.storeRecording().Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/recordings/{name}/csv",
			arrange: func(h *harness) (string, any) {
				return recordingPath(h.storeRecording().Name) + "/csv", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestRecordingListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the recordings and marks the one being written", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		current := h.currentRecording()
		created := h.storeRecording()

		response := h.asAdmin().GET("/api/recordings")

		response.AssertOK()
		got := response.Decode[[]api.RecordingResponse]()
		assert.ElementsMatch(t, []api.RecordingResponse{
			current,
			created,
		}, got)
		assert.True(t, current.Current)
		assert.False(t, created.Current)
	})

	t.Run("returns no recordings with the recording off", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Timing.Recording = false
		}))

		response := h.asAdmin().GET("/api/recordings")

		response.AssertOK().AssertJSONCount("", 0)
	})
}

func TestRecordingCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the recording", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		upload := newRecordingUpload(t, h.Faker)

		created := client.POST("/api/recordings", upload).AssertCreated().Decode[api.RecordingResponse]()

		assert.Equal(t, upload.Files[0].Name, created.Name)
		assert.Equal(t, "recordings/"+created.Name, created.Path)
		stored, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		require.NoError(t, err)
		assert.Equal(t, upload.Files[0].Content, stored)
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
				name: "binary file",
				change: func(m *apitest.Multipart, _ *gofakeit.Faker) {
					m.Files[0].Content = []byte{0x00, 0x01, 0x02, 0xFF}
				},
				field: "file",
				tag:   "unsupported_type",
			},
			{
				name: "file that is no recording",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files[0].Content = []byte(f.Sentence(6))
				},
				field: "file",
				tag:   "invalid",
			},
			{
				name: "missing file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files = nil
					m.Fields = map[string]string{"name": f.Word()}
				},
				field: "file",
				tag:   "required",
			},
			{
				name: "too large file",
				change: func(m *apitest.Multipart, _ *gofakeit.Faker) {
					m.Files[0].Content = bytes.Repeat([]byte(" "), 101<<20)
				},
				field: "file",
				tag:   "too_large",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t, withConfig(func(cfg *config.Config) {
					cfg.Timing.Recording = false
				}))
				client := h.asAdmin()
				body := newRecordingUpload(t, h.Faker)
				tt.change(&body, h.Faker)

				response := client.POST("/api/recordings", body)

				assertFieldError(t, response, tt.field, tt.tag)
				client.GET("/api/recordings").AssertOK().AssertJSONCount("", 0)
			})
		}
	})

	t.Run("keeps the recording being written", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		current := h.currentRecording()
		upload := newRecordingUpload(t, h.Faker)
		upload.Files[0].Name = current.Name

		response := h.asAdmin().POST("/api/recordings", upload)

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
	})
}

func TestRecordingDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the recording", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Timing.Recording = false
		}))
		client := h.asAdmin()
		created := h.storeRecording()

		response := client.DELETE(recordingPath(created.Name))

		response.AssertNoContent()
		client.GET("/api/recordings").AssertOK().AssertJSONCount("", 0)
		_, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		assert.ErrorIs(t, err, media.ErrNotFound)
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(recordingPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for an unknown recording", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(recordingPath(h.Faker.LetterN(8) + ".jsonl"))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("keeps the replayed recording", func(t *testing.T) {
		t.Parallel()
		name := apitest.Faker(t).LetterN(8) + ".jsonl"
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Timing.ReplayPath = "recordings/" + name
		}))
		created := h.storeRecordingNamed(name)

		response := h.asAdmin().DELETE(recordingPath(created.Name))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
		_, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		assert.NoError(t, err)
	})

	t.Run("keeps the recording being written", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		current := h.currentRecording()

		response := h.asAdmin().DELETE(recordingPath(current.Name))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
	})
}

func TestRecordingDownloadHandler(t *testing.T) {
	t.Parallel()

	t.Run("downloads the recording as it is stored", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		created := h.storeRecording()
		stored, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		require.NoError(t, err)

		response := h.asAdmin().GET(recordingPath(created.Name))

		response.AssertOK().
			AssertHeader("Content-Type", "application/jsonl").
			AssertHeader("Content-Disposition", "attachment; filename="+created.Name)
		assert.Equal(t, stored, response.Body)
	})

	t.Run("returns not found for an unknown recording", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(recordingPath(h.Faker.LetterN(8) + ".jsonl"))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestRecordingExportHandler(t *testing.T) {
	t.Parallel()

	t.Run("downloads the recording as a table", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		created := h.storeRecording()

		response := h.asAdmin().GET(recordingPath(created.Name) + "/csv")

		response.AssertOK().
			AssertHeader("Content-Type", "text/csv").
			AssertHeader("Content-Disposition", "attachment; filename="+strings.TrimSuffix(created.Name, ".jsonl")+".csv")
		rows, err := csv.NewReader(bytes.NewReader(response.Body)).ReadAll()
		require.NoError(t, err)
		require.Len(t, rows, 3, "the header, the session, and the frame")
		assert.Equal(t, "frame", rows[2][2])
	})

	t.Run("returns not found for an unknown recording", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(recordingPath(h.Faker.LetterN(8)+".jsonl") + "/csv")

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("reports a stored recording that cannot be read", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		name := h.Faker.LetterN(8) + ".jsonl"
		content := []byte(h.Faker.Sentence(6))
		require.NoError(t, h.Storage.Put(t.Context(), "recordings/"+name, bytes.NewReader(content), int64(len(content))))

		response := h.asAdmin().GET(recordingPath(name) + "/csv")

		response.AssertStatus(http.StatusInternalServerError)
		assert.Empty(t, response.Header.Get("Content-Disposition"), "no download is offered")
	})
}

// recordingPath returns the API path of a recording.
func recordingPath(name string) string {
	return "/api/recordings/" + url.PathEscape(name)
}
