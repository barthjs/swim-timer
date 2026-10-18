package api_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
)

func TestAuthAccess(t *testing.T) {
	t.Parallel()
	f := apitest.Faker(t)

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/auth/me",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/auth/login",
			body:   newAuthLoginRequest(f),
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusConflict,
			},
			codes: map[*clientKind]string{
				clientGuest: "invalid_credentials",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/auth/logout",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusNoContent,
				clientAdmin: http.StatusNoContent,
			},
		},
	})
}

func TestAuthStatusGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("reports a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/api/auth/me").AssertOK().AssertJSON(map[string]bool{"authenticated": false})
	})
}

func TestAuthLoginHandler(t *testing.T) {
	t.Parallel()

	t.Run("starts a session", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: h.Config.App.Secret,
		}).AssertNoContent()

		h.Client.GET("/api/auth/me").AssertOK().AssertJSON(map[string]bool{"authenticated": true})
	})

	t.Run("accepts an empty password without a secret", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) { cfg.App.Secret = "" }))

		h.Client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: "",
		}).AssertNoContent()

		h.Client.GET("/api/auth/me").AssertOK().AssertJSON(map[string]bool{"authenticated": true})
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.POST("/api/auth/login", map[string]string{h.Faker.LetterN(8): h.Faker.Word()}).
			AssertBadRequest().
			AssertJSONPath("code", "bad_request")
	})

	t.Run("rejects wrong credentials", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertUnauthorized().AssertJSONPath("code", "invalid_credentials")

		h.Client.GET("/api/auth/me").AssertOK().AssertJSON(map[string]bool{"authenticated": false})
	})

	t.Run("rejects a signed-in session", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: h.Config.App.Secret,
		}).
			AssertStatus(http.StatusConflict).
			AssertJSONPath("code", "conflict")
	})

	t.Run("is rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		wrong := newAuthLoginRequest(h.Faker)

		for range 5 {
			h.Client.POST("/api/auth/login", wrong).AssertUnauthorized()
		}

		response := h.Client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: h.Config.App.Secret,
		}).
			AssertTooManyRequests().
			AssertJSONPath("code", "rate_limited")
		assert.NotEmpty(t, response.Header.Get("Retry-After"), "a rate limited response names when to retry")
	})

	t.Run("keeps parallel attempts within the limit", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		wrong := newAuthLoginRequest(h.Faker)

		counts := parallelStatuses(20, func(int) int { return h.Client.POST("/api/auth/login", wrong).Status })

		assert.Equal(t, 5, counts[http.StatusUnauthorized], "only the allowed attempts reach the password check")
		assert.Equal(t, 15, counts[http.StatusTooManyRequests], "every further attempt is rejected")
	})

	t.Run("resets the failed attempts after a success", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		wrong := newAuthLoginRequest(h.Faker)

		for range 4 {
			h.Client.POST("/api/auth/login", wrong).AssertUnauthorized()
		}
		h.Client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: h.Config.App.Secret,
		}).AssertNoContent()
		h.Client.POST("/api/auth/logout", nil).AssertNoContent()

		// Without the reset the last of these attempts would exceed the limit
		for range 4 {
			h.Client.POST("/api/auth/login", wrong).AssertUnauthorized()
		}
		h.Client.POST("/api/auth/login", api.AuthLoginRequest{
			Password: h.Config.App.Secret,
		}).AssertNoContent()
	})
}

func TestAuthLogoutHandler(t *testing.T) {
	t.Parallel()

	t.Run("ends the session", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		client.POST("/api/auth/logout", nil).AssertNoContent()

		client.GET("/api/auth/me").AssertOK().AssertJSON(map[string]bool{"authenticated": false})
		client.GET("/api/config").AssertUnauthorized()
	})
}

// parallelStatuses sends n requests at the same time through send and counts the returned status codes.
func parallelStatuses(n int, send func(i int) int) map[int]int {
	statuses := make(chan int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			statuses <- send(i)
		})
	}
	wg.Wait()
	close(statuses)

	counts := make(map[int]int)
	for status := range statuses {
		counts[status]++
	}
	return counts
}
