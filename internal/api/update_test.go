package api_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
)

func TestUpdateAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/update/status",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/update",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/update",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusUnprocessableEntity,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
	})
}

func TestUpdateStatusGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the idle state", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().GET("/api/update/status").AssertOK().AssertJSONPath("state", "idle")
	})
}

func TestUpdateCheckHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the newer release", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withLatestRelease("v1.1.0"))

		h.asAdmin().GET("/api/update").AssertOK().
			AssertJSONPath("current", harnessVersion).
			AssertJSONPath("latest", "v1.1.0").
			AssertJSONPath("newer", true).
			AssertJSONPath("unsupported", "development_build")
	})

	t.Run("reports the running version as the latest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().GET("/api/update").AssertOK().AssertJSONPath("latest", harnessVersion).AssertJSONPath("newer", false)
	})
}

func TestUpdateStartHandler(t *testing.T) {
	t.Parallel()

	t.Run("installs the release in the background", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			result func(h *harness) error
			state  string
		}{
			{
				name:   "applies the installed release",
				result: func(*harness) error { return nil },
				state:  "applied",
			},
			{
				name:   "reports a failed installation",
				result: func(h *harness) error { return errors.New(h.Faker.LoremIpsumSentence(4)) },
				state:  "failed",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				updater := newInstallableUpdater(t)
				h := newHarness(t, withUpdater(updater))
				admin := h.asAdmin()

				admin.POST("/api/update", nil).AssertStatus(http.StatusAccepted)
				admin.GET("/api/update/status").AssertOK().AssertJSONPath("state", "running")
				admin.POST("/api/update", nil).AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")

				updater.result <- tt.result(h)

				require.Eventually(
					t,
					func() bool {
						return admin.GET("/api/update/status").Decode[api.UpdateStatusResponse]().State != "running"
					},
					eventuallyTimeout,
					eventuallyTick,
					"the update ends once the release is installed",
				)
				admin.GET("/api/update/status").AssertOK().AssertJSONPath("state", tt.state).AssertJSONPath("version", updater.latest)
			})
		}
	})

	t.Run("rejects an unsupported installation", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withLatestRelease("v1.1.0"))
		admin := h.asAdmin()

		assertFieldError(t, admin.POST("/api/update", nil), "update", "development_build")

		admin.GET("/api/update/status").AssertStatus(http.StatusOK).AssertJSONPath("state", "idle")
	})
}
