package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/colormix"
)

func TestManifestAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/site.webmanifest",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestManifestGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the manifest with the name and the dark background of the default theme", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, func(h *harness) {
			h.Config.App.Name = h.Faker.AppName()
			h.Config.Theme.Neutral = ""
		})

		response := h.Client.GET("/site.webmanifest").AssertOK().AssertHeader("Cache-Control", "no-cache")
		manifest := response.Decode[api.ManifestResponse]()

		assert.Equal(t, h.Config.App.Name, manifest.Name)
		assert.Equal(t, h.Config.App.Name, manifest.ShortName)
		assert.Equal(t, "/", manifest.StartURL)
		assert.Equal(t, "standalone", manifest.Display)
		assert.NotEmpty(t, manifest.Icons)
		assert.Equal(t, "#0f172a", manifest.ThemeColor)
		assert.Equal(t, "#004b71", manifest.BackgroundColor, "the splash screen shows the background of the app icon")
	})

	t.Run("colors the bars with the dark background of the neutral color", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, func(h *harness) {
			h.Config.Theme.Neutral = h.Faker.HexColor()
		})
		want, err := colormix.WithBlack(h.Config.Theme.Neutral, 60)
		require.NoError(t, err)

		manifest := h.Client.GET("/site.webmanifest").AssertOK().Decode[api.ManifestResponse]()

		assert.Equal(t, want, manifest.ThemeColor)
	})
}
