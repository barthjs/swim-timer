package api_test

import (
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/web"
)

func TestSystemAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/healthz",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/readyz",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/meta",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/ping",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestHealth_Live(t *testing.T) {
	t.Parallel()

	t.Run("reports the liveness", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/healthz").AssertOK().AssertJSONPath("status", "ok").AssertJSONPath("version", harnessVersion)
	})
}

func TestHealth_Ready(t *testing.T) {
	t.Parallel()

	t.Run("reports the dependencies", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/readyz").AssertOK().AssertJSONPath("status", "ok").AssertJSONPath("checks.timing", "ok")
	})

	t.Run("reports degraded without the device", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDisconnectedDevice())

		// The application keeps serving without the device, so it reports itself degraded, not unavailable
		h.Client.GET("/readyz").AssertOK().AssertJSONPath("status", "degraded").AssertJSONPath("checks.timing", "unreachable")
		h.Client.GET("/healthz").AssertOK().AssertJSONPath("status", "ok")
	})
}

func TestAPI_Routes(t *testing.T) {
	t.Parallel()

	t.Run("serves the docs in production", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.App.Env = config.EnvProduction
		}))

		h.Client.GET("/docs").AssertStatus(http.StatusMovedPermanently).AssertHeader("Location", "/docs/")
		h.Client.GET("/docs/").AssertOK()
		h.Client.GET("/docs/api/").AssertOK()

		spec := h.Client.GET("/docs/api.json").AssertOK()
		assert.Contains(t, string(spec.Body), `"swagger"`)
		assert.NotContains(t, string(spec.Body), "internal_", "the schemas carry their type names without the package path")
	})

	t.Run("redirects a docs path without a page to the start page", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/docs/"+h.Faker.LetterN(12)+"/").AssertStatus(http.StatusFound).AssertHeader("Location", "/docs/")
		h.Client.GET("/docs/_astro/").AssertStatus(http.StatusFound).AssertHeader("Location", "/docs/")
	})

	t.Run("returns not found for an unknown route", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.GET("/api/"+h.Faker.LetterN(12)).AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("rejects a wrong method", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.DELETE("/api/ping").AssertStatus(http.StatusMethodNotAllowed).AssertJSONPath("code", "method_not_allowed")
	})
}

func TestMetaGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the public settings", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, func(h *harness) {
			h.Config.App.Name = h.Faker.AppName()
			h.Config.App.Language = config.LanguageDE
			h.Config.Theme.Primary = h.Faker.HexColor()
		})

		response := h.Client.GET("/api/meta").AssertOK()
		meta := response.Decode[api.MetaResponse]()

		assert.Equal(t, h.Config.App.Name, meta.Name)
		assert.Equal(t, harnessVersion, meta.Version)
		assert.Equal(t, string(config.LanguageDE), meta.Language)
		assert.Equal(t, h.Config.Theme.Primary, meta.Theme.Primary)
		assert.NotContains(t, string(response.Body), h.Config.App.Secret, "the public settings never carry the secret")
	})
}

func TestPingGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("answers pong", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/ping").AssertOK().AssertHeader("Content-Type", "text/plain")
		assert.Equal(t, "pong", string(response.Body))
	})
}

func TestAPI_StaticRoutes(t *testing.T) {
	t.Parallel()

	t.Run("serves the pages", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		index := h.Client.GET("/").AssertOK()
		assert.Contains(t, string(index.Body), "<html")

		// Unknown paths outside the API belong to the client-side router and get the index page too
		h.Client.GET("/"+h.Faker.LetterN(10)).AssertOK().AssertHeader("Content-Type", index.Header.Get("Content-Type"))
	})

	t.Run("serves the public files", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		entries, err := fs.ReadDir(web.Files, "dist")
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			want, err := web.Files.ReadFile("dist/" + entry.Name())
			require.NoError(t, err)

			response := h.Client.GET("/" + entry.Name()).AssertOK()

			assert.Equal(t, string(want), string(response.Body), entry.Name())
		}
	})

	t.Run("caches the assets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		assets, err := fs.ReadDir(web.Files, "dist/assets")
		require.NoError(t, err)
		require.NotEmpty(t, assets, "the web interface is built into the binary")

		h.Client.GET("/assets/"+assets[0].Name()).AssertOK().AssertHeader("Cache-Control", "public, max-age=31536000, immutable")
	})

	t.Run("serves the overlay app", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		index := h.Client.GET("/").AssertOK()
		overlay := h.Client.GET("/overlays/" + h.Faker.LetterN(6)).AssertOK()

		assert.Contains(t, string(overlay.Body), "<html")
		assert.NotEqual(t, string(index.Body), string(overlay.Body), "the overlays are a separate app")
	})
}
