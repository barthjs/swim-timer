package api_test

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
)

func TestPresetAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/presets",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/presets/{name}",
			arrange: func(h *harness) (string, any) {
				preset := newPresetSaveRequest(h.Faker)
				h.asAdmin().PUT(presetPath(preset.Name), preset).AssertOK()
				return presetPath(preset.Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusOK,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/presets/{name}",
			arrange: func(h *harness) (string, any) {
				preset := newPresetSaveRequest(h.Faker)
				return presetPath(preset.Name), preset
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/presets/{name}",
			arrange: func(h *harness) (string, any) {
				preset := newPresetSaveRequest(h.Faker)
				h.asAdmin().PUT(presetPath(preset.Name), preset).AssertOK()
				return presetPath(preset.Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
	})
}

func TestPresetListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the presets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		names := make([]string, 0, 3)
		for range 3 {
			preset := newPresetSaveRequest(h.Faker)
			admin.PUT(presetPath(preset.Name), preset).AssertOK()
			names = append(names, preset.Name)
		}

		listed := h.Client.GET("/api/presets").AssertOK().Decode[[]string]()

		slices.Sort(names)
		slices.Sort(listed)
		assert.Equal(t, names, listed)
	})

	t.Run("returns an empty list without presets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/presets")

		response.AssertOK().AssertJSON([]string{})
	})
}

func TestPresetGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		preset := newPresetSaveRequest(h.Faker)
		h.asAdmin().PUT(presetPath(preset.Name), preset).AssertOK()

		response := h.Client.GET(presetPath(preset.Name))

		response.AssertOK().AssertJSON(preset)
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET(presetPath(presetName(h.Faker)))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET(presetPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestPresetSaveHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		preset := newPresetSaveRequest(h.Faker)

		response := h.asAdmin().PUT(presetPath(preset.Name), preset)

		response.AssertOK().AssertJSON(preset)
		exists, err := h.Storage.Exists(t.Context(), "presets/"+preset.Name+".json")
		require.NoError(t, err)
		assert.True(t, exists, "the preset is stored as a file")
		h.Client.GET(presetPath(preset.Name)).AssertJSON(preset)
	})

	t.Run("replaces the preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		preset := newPresetSaveRequest(h.Faker)
		admin.PUT(presetPath(preset.Name), preset).AssertOK()
		replaced := newPresetSaveRequest(h.Faker, func(p *api.PresetSaveRequest) { p.Name = preset.Name })

		admin.PUT(presetPath(preset.Name), replaced).AssertOK()

		h.Client.GET(presetPath(preset.Name)).AssertJSON(replaced)
		h.Client.GET("/api/presets").AssertJSONCount("", 1)
	})

	t.Run("stores the longest name of wide characters", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		// Characters of 4 bytes each give the longest file name the name limit allows
		name := make([]rune, 60)
		for i := range name {
			name[i] = rune(h.Faker.IntRange(0x1F600, 0x1F64F))
		}
		preset := newPresetSaveRequest(h.Faker, func(p *api.PresetSaveRequest) { p.Name = string(name) })

		h.asAdmin().PUT(presetPath(preset.Name), preset).AssertOK()

		h.Client.GET(presetPath(preset.Name)).AssertOK().AssertJSON(preset)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*api.PresetSaveRequest, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "missing name",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.Name = "" },
				field:  "name",
				tag:    "required",
			},
			{
				name:   "missing text size",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.TextSize = 0 },
				field:  "textSize",
				tag:    "required",
			},
			{
				name:   "gap larger than a lane",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.LaneGap = 101 },
				field:  "laneGap",
				tag:    "max",
			},
			{
				name:   "text larger than the lane",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.TextSize = 101 },
				field:  "textSize",
				tag:    "max",
			},
			{
				name:   "unknown text anchor",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.TextAnchor = "middle" },
				field:  "textAnchor",
				tag:    "oneof",
			},
			{
				name:   "empty comment",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { empty := ""; r.Comment = &empty },
				field:  "comment",
				tag:    "min",
			},
			{
				name:   "missing lanes",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.Lanes = nil },
				field:  "lanes",
				tag:    "required",
			},
			{
				name: "coordinate outside the canvas",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					lane := r.Lanes["1"]
					lane.TL.X = 1.5
					r.Lanes["1"] = lane
				},
				field: "lanes[1].tl.x",
				tag:   "max",
			},
			{
				name:   "name too long",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.Name = strings.Repeat(r.Name[:1], 61) },
				field:  "name",
				tag:    "max",
			},
			{
				name: "lane key that is no number",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					r.Lanes["left"] = r.Lanes["1"]
				},
				field: "lanes[left]",
				tag:   "lane",
			},
			{
				name: "lane key zero",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					r.Lanes["0"] = r.Lanes["1"]
				},
				field: "lanes[0]",
				tag:   "lane",
			},
			{
				name: "negative lane key",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					r.Lanes["-1"] = r.Lanes["1"]
				},
				field: "lanes[-1]",
				tag:   "lane",
			},
			{
				name: "fractional lane key",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					r.Lanes["1.5"] = r.Lanes["1"]
				},
				field: "lanes[1.5]",
				tag:   "lane",
			},
			{
				name: "lane key with a leading zero",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) {
					r.Lanes["01"] = r.Lanes["1"]
				},
				field: "lanes[01]",
				tag:   "lane",
			},
			{
				name:   "missing frame width",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.FrameWidth = 0 },
				field:  "frameWidth",
				tag:    "required",
			},
			{
				name:   "blank name",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.Name = " " },
				field:  "name",
				tag:    "invalid",
			},
			{
				name:   "name outside the folder",
				change: func(r *api.PresetSaveRequest, _ *gofakeit.Faker) { r.Name = ".." },
				field:  "name",
				tag:    "invalid",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				request := newPresetSaveRequest(h.Faker)
				name := request.Name
				tt.change(&request, h.Faker)
				// The path carries the changed name, so the name check of the store sees it.
				// An empty name has no route and keeps the original one.
				if request.Name != "" {
					name = request.Name
				}

				response := h.asAdmin().PUT(presetPath(name), request)

				assertFieldError(t, response, tt.field, tt.tag)
				h.Client.GET("/api/presets").AssertJSONCount("", 0)
			})
		}
	})

	t.Run("rejects a name other than the path", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		preset := newPresetSaveRequest(h.Faker)
		pathName := presetName(h.Faker)

		response := h.asAdmin().PUT(presetPath(pathName), preset)

		assertFieldError(t, response, "name", "invalid")
		h.Client.GET(presetPath(pathName)).AssertNotFound()
		h.Client.GET(presetPath(preset.Name)).AssertNotFound()
	})

	t.Run("stores nothing for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		preset := newPresetSaveRequest(h.Faker)

		h.Client.PUT(presetPath(preset.Name), preset).AssertUnauthorized()

		h.Client.GET(presetPath(preset.Name)).AssertNotFound()
	})
}

func TestPresetDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		preset := newPresetSaveRequest(h.Faker)
		admin.PUT(presetPath(preset.Name), preset).AssertOK()

		admin.DELETE(presetPath(preset.Name)).AssertNoContent()

		h.Client.GET(presetPath(preset.Name)).AssertNotFound()
		exists, err := h.Storage.Exists(t.Context(), "presets/"+preset.Name+".json")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns not found for an unknown preset", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(presetPath(presetName(h.Faker)))

		response.AssertNotFound()
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		preset := newPresetSaveRequest(h.Faker)
		admin.PUT(presetPath(preset.Name), preset).AssertOK()

		response := admin.DELETE(presetPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
		h.Client.GET("/api/presets").AssertJSON([]string{preset.Name})
	})

	t.Run("keeps the preset for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		preset := newPresetSaveRequest(h.Faker)
		h.asAdmin().PUT(presetPath(preset.Name), preset).AssertOK()

		h.Client.DELETE(presetPath(preset.Name)).AssertUnauthorized()

		h.Client.GET(presetPath(preset.Name)).AssertOK()
	})
}

// presetPath returns the API path of a preset.
func presetPath(name string) string {
	return "/api/presets/" + url.PathEscape(name)
}
