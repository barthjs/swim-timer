package api_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/server"
)

// loginAddressLimit is the number of failed logins one address may make before it is limited.
const loginAddressLimit = 5

func TestRequestID(t *testing.T) {
	t.Parallel()

	t.Run("creates and returns an id", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/ping").AssertOK()

		assert.Len(t, response.Header.Get("X-Request-ID"), 32)
	})

	t.Run("takes the id from the request", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		id := h.Faker.UUID()

		h.Client.WithHeader("X-Request-ID", id).GET("/api/missing").AssertNotFound().AssertHeader("X-Request-ID", id).AssertJSONPath("requestId", id)
	})

	t.Run("includes the id in validation errors", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		id := h.Faker.UUID()

		h.asAdmin().WithHeader("X-Request-ID", id).PUT(presetPath(h.Faker.LetterN(8)), api.PresetSaveRequest{}).
			AssertUnprocessable().
			AssertJSONPath("requestId", id)
	})
}

func TestRealIP(t *testing.T) {
	t.Parallel()

	t.Run("trusts forwarding headers of trusted proxies", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			// The range contains the peer address of the test requests
			cfg.Server.RawTrustedProxies = "192.0.2.0/24"
		}))
		first := h.Client.WithHeader("X-Forwarded-For", h.Faker.IPv4Address())
		second := h.Client.WithHeader("X-Real-IP", h.Faker.IPv6Address())

		for range loginAddressLimit {
			first.POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertUnauthorized()
		}

		first.POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertTooManyRequests()
		second.POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertUnauthorized()
	})

	t.Run("ignores forwarding headers of untrusted peers", func(t *testing.T) {
		t.Parallel()
		// The peer address of the test requests is not a trusted proxy by default
		h := newHarness(t)

		for range loginAddressLimit {
			h.Client.WithHeader("X-Forwarded-For", h.Faker.IPv4Address()).POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertUnauthorized()
		}

		h.Client.WithHeader("X-Forwarded-For", h.Faker.IPv4Address()).POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertTooManyRequests()
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	t.Run("sets the security headers on a page", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/").AssertOK()

		assertSecurityHeaders(t, response)
	})

	t.Run("sets the security headers on an api response", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/ping").AssertOK()

		assertSecurityHeaders(t, response)
	})

	t.Run("sets the security headers on an overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/overlays/" + h.Faker.LetterN(6)).AssertOK()

		assertSecurityHeaders(t, response)
	})

	t.Run("sets the security headers on a rejected websocket upgrade", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		// The upgrader writes the response of an accepted connection itself, only a rejected one passes the middleware
		response := h.Client.GET("/api/ws").AssertBadRequest()

		assertSecurityHeaders(t, response)
	})

	t.Run("serves the api docs without inline scripts", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/docs/").AssertOK()

		assertSecurityHeaders(t, response)
		assert.NotContains(t, string(response.Body), "<script>", "the policy blocks inline scripts")
	})
}

// assertSecurityHeaders checks that a response carries the security headers and the policies of the application.
// httptest addresses every request to example.com, whose websocket origins the policy lists.
func assertSecurityHeaders(t *testing.T, response *apitest.Response) {
	t.Helper()

	response.
		AssertHeader("Cross-Origin-Opener-Policy", "same-origin").
		AssertHeader("Cross-Origin-Resource-Policy", "same-origin").
		AssertHeader("Referrer-Policy", "strict-origin-when-cross-origin").
		AssertHeader("X-Content-Type-Options", "nosniff").
		AssertHeader("X-Frame-Options", "DENY")

	policy := response.Header.Get("Content-Security-Policy")
	assert.Contains(t, policy, "default-src 'self'")
	assert.Contains(t, policy, "script-src 'self';")
	assert.Contains(t, policy, "connect-src 'self' ws://example.com wss://example.com;")
	assert.Contains(t, policy, "frame-ancestors 'none'")
	assert.Contains(t, response.Header.Get("Permissions-Policy"), "camera=()")
}

func TestCORS(t *testing.T) {
	t.Parallel()

	t.Run("allows configured origins", func(t *testing.T) {
		t.Parallel()
		origin := "https://" + apitest.Faker(t).DomainName()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Server.CORSAllowedOrigins = []string{origin}
			cfg.Server.CORSAllowCredentials = true
		}))

		h.Client.WithHeader("Origin", origin).GET("/api/ping").
			AssertOK().
			AssertHeader("Access-Control-Allow-Origin", origin).
			AssertHeader("Access-Control-Allow-Credentials", "true")

		h.Client.WithHeader("Origin", origin).WithHeader("Access-Control-Request-Method", http.MethodPost).
			Do(http.MethodOptions, "/api/auth/login", nil).
			AssertNoContent().
			AssertHeader("Access-Control-Allow-Origin", origin)
	})

	t.Run("ignores other origins", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.WithHeader("Origin", "https://"+h.Faker.DomainName()).GET("/api/ping").AssertOK()

		assert.Empty(t, response.Header.Get("Access-Control-Allow-Origin"))
	})
}

func TestBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("rejects a large json body", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		body := `{"password":"` + strings.Repeat("a", server.DefaultMaxBodySize) + `"}`
		h.Client.POST("/api/auth/login", strings.NewReader(body)).
			AssertStatus(http.StatusRequestEntityTooLarge).
			AssertJSONPath("code", "payload_too_large")
	})

	t.Run("allows more for uploads", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		meet := newMeetFixture(h.Faker, 2)

		// The unknown file is reported on its own and only makes the request exceed the default limit
		files := append(meet.files(t), apitest.File{
			Field:   "files",
			Name:    "unknown.txt",
			Content: []byte(strings.Repeat("a", server.DefaultMaxBodySize)),
		})
		h.asAdmin().POST("/api/meet/files", apitest.Multipart{
			Files: files,
		}).AssertOK()
	})
}

func TestCrossOrigin(t *testing.T) {
	t.Parallel()

	t.Run("accepts same origin changes", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		client.WithHeader("Sec-Fetch-Site", "same-origin").WithHeader("Origin", "http://example.com").POST("/api/auth/logout", nil).AssertNoContent()
	})

	t.Run("accepts configured origins", func(t *testing.T) {
		t.Parallel()
		origin := "https://" + apitest.Faker(t).DomainName()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Server.CORSAllowedOrigins = []string{origin}
		}))
		client := h.asAdmin()

		client.WithHeader("Sec-Fetch-Site", "cross-site").WithHeader("Origin", origin).POST("/api/auth/logout", nil).AssertNoContent()
	})

	t.Run("rejects cross site changes", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		client.WithHeader("Sec-Fetch-Site", "cross-site").WithHeader("Origin", "https://"+h.Faker.DomainName()).
			POST("/api/auth/logout", nil).
			AssertForbidden().
			AssertJSONPath("code", "forbidden")

		client.GET("/api/auth/me").AssertJSONPath("authenticated", true)
	})
}

func TestAPI_RateLimit(t *testing.T) {
	t.Parallel()

	t.Run("reports the headers", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.POST("/api/meet/reload", nil).AssertNoContent()

		assert.Equal(t, "10", response.Header.Get("RateLimit-Limit"))
		assert.Equal(t, "9", response.Header.Get("RateLimit-Remaining"))
		reset, err := strconv.Atoi(response.Header.Get("RateLimit-Reset"))
		require.NoError(t, err)
		assert.InDelta(t, time.Minute.Seconds(), reset, 2)
	})

	t.Run("counts the request itself", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.POST("/api/auth/login", newAuthLoginRequest(h.Faker)).AssertUnauthorized()

		assert.Equal(t, "5", response.Header.Get("RateLimit-Limit"))
		assert.Equal(t, "4", response.Header.Get("RateLimit-Remaining"), "the attempt itself is counted")
	})
}
