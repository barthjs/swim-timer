package api

import (
	"cmp"
	"net/http"

	"github.com/barthjs/swim-timer/internal/colormix"
)

// defaultDarkBackground is the page background of the dark mode with the default neutral color of the theme.
const defaultDarkBackground = "#0f172a"

// darkBackgroundBlackPercent is the share of black in the shade of the neutral color that the dark mode uses
// as page background, as the web interface mixes it.
const darkBackgroundBlackPercent = 60

// manifestGetHandler returns the web app manifest with the name and the colors of the configuration.
// Browsers paint the bars of an installed app in the theme color, which stays dark.
func (a *API) manifestGetHandler(w http.ResponseWriter, r *http.Request) {
	// The bar at the bottom keeps the color of the installation and cannot follow the color mode of the page
	themeColor := defaultDarkBackground
	if a.options.Theme.Neutral != "" {
		mixed, err := colormix.WithBlack(a.options.Theme.Neutral, darkBackgroundBlackPercent)
		if err != nil {
			a.serverErrorResponse(w, r, err)
			return
		}
		themeColor = mixed
	}

	a.writeJSON(w, r, http.StatusOK, ManifestResponse{
		ID:              "/",
		Name:            a.options.Name,
		ShortName:       a.options.Name,
		StartURL:        "/",
		Scope:           "/",
		Display:         "standalone",
		BackgroundColor: cmp.Or(a.options.SplashColor, themeColor),
		ThemeColor:      themeColor,
		Icons: []ManifestIcon{
			{
				Src:   "/icons/icon-192.png",
				Sizes: "192x192",
				Type:  "image/png",
			},
			{
				Src:   "/icons/icon-512.png",
				Sizes: "512x512",
				Type:  "image/png",
			},
			{
				Src:     "/icons/icon-512.png",
				Sizes:   "512x512",
				Type:    "image/png",
				Purpose: "maskable",
			},
		},
	}, http.Header{
		"Cache-Control": {"no-cache"},
	})
}

// ManifestResponse is the web app manifest, whose field names the standard sets in snake case.
type ManifestResponse struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	ShortName       string         `json:"short_name"`
	StartURL        string         `json:"start_url"`
	Scope           string         `json:"scope"`
	Display         string         `json:"display"`
	BackgroundColor string         `json:"background_color"`
	ThemeColor      string         `json:"theme_color"`
	Icons           []ManifestIcon `json:"icons"`
}

type ManifestIcon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose,omitzero"`
}
