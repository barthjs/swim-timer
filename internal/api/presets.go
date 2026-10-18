package api

import (
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/barthjs/swim-timer/internal/domain"
)

// presetListHandler returns the names of all presets.
//
//	@Summary		List presets
//	@Description	Returns the names of all AR overlay presets.
//	@Tags			presets
//	@Produce		json
//	@Success		200	{array}		string
//	@Failure		500	{object}	ErrorResponse
//	@Router			/presets [get]
func (a *API) presetListHandler(w http.ResponseWriter, r *http.Request) {
	names, err := a.presetService.List(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, names, nil)
}

// presetGetHandler returns a single preset.
//
//	@Summary		Get preset
//	@Description	Returns the AR overlay preset with the given name.
//	@Tags			presets
//	@Produce		json
//	@Param			name	path		string	true	"Preset name"
//	@Success		200		{object}	PresetResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/presets/{name} [get]
func (a *API) presetGetHandler(w http.ResponseWriter, r *http.Request) {
	preset, err := a.presetService.Get(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toPresetResponse(*preset), nil)
}

// PresetResponse is a stored preset.
type PresetResponse struct {
	Name         string                        `json:"name"`
	Comment      *string                       `json:"comment"`
	FrameWidth   int                           `json:"frameWidth"`
	FrameHeight  int                           `json:"frameHeight"`
	TextSize     int                           `json:"textSize"`
	LaneGap      int                           `json:"laneGap"`
	ReverseLanes bool                          `json:"reverseLanes"`
	TextAnchor   string                        `json:"textAnchor" enums:"left,right"`
	Lanes        map[string]PresetLaneResponse `json:"lanes"`
}

// PresetLaneResponse holds the corners and the text position of a lane.
type PresetLaneResponse struct {
	TL     PresetPointResponse `json:"tl"`
	TR     PresetPointResponse `json:"tr"`
	BR     PresetPointResponse `json:"br"`
	BL     PresetPointResponse `json:"bl"`
	Locked bool                `json:"locked"`
}

// PresetPointResponse is a position in the output image as a fraction of its width and height, from 0 to 1.
type PresetPointResponse struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PresetSaveRequest holds the preset to create or replace. The name has to match the name in the path.
// It names the preset file, so 60 characters of at most 4 bytes each plus the extension
// stay within the 255 bytes a file name may have.
type PresetSaveRequest struct {
	Name         string                       `json:"name" validate:"required,min=1,max=60"`
	Comment      *string                      `json:"comment" validate:"omitempty,min=1,max=1000"`
	FrameWidth   int                          `json:"frameWidth" validate:"required,min=1,max=100000"`
	FrameHeight  int                          `json:"frameHeight" validate:"required,min=1,max=100000"`
	TextSize     int                          `json:"textSize" validate:"required,min=10,max=100"`
	LaneGap      int                          `json:"laneGap" validate:"min=0,max=100"`
	ReverseLanes bool                         `json:"reverseLanes"`
	TextAnchor   domain.TextAnchor            `json:"textAnchor" validate:"required,oneof=left right"`
	Lanes        map[string]PresetLaneRequest `json:"lanes" validate:"required,min=1,dive,keys,lane,endkeys"`
}

// PresetLaneRequest holds the corners and the text position of a lane.
type PresetLaneRequest struct {
	TL     PresetPointRequest `json:"tl" validate:"required"`
	TR     PresetPointRequest `json:"tr" validate:"required"`
	BR     PresetPointRequest `json:"br" validate:"required"`
	BL     PresetPointRequest `json:"bl" validate:"required"`
	Locked bool               `json:"locked"`
}

// PresetPointRequest is a position in the output image as a fraction of its width and height, from 0 to 1.
type PresetPointRequest struct {
	X float64 `json:"x" validate:"min=0,max=1"`
	Y float64 `json:"y" validate:"min=0,max=1"`
}

// presetSaveHandler creates or replaces a preset.
//
//	@Summary		Save preset
//	@Description	Creates or replaces the AR overlay preset with the given name, which has to match the name in the body (Admin only).
//	@Tags			presets
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string				true	"Preset name"
//	@Param			request	body		PresetSaveRequest	true	"Preset"
//	@Success		200		{object}	PresetResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/presets/{name} [put]
func (a *API) presetSaveHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[PresetSaveRequest](w, r)
	if !ok {
		return
	}

	preset := req.toPreset()
	if err := a.presetService.Save(r.Context(), chi.URLParam(r, "name"), preset); err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toPresetResponse(preset), nil)
}

// toPresetResponse maps a preset to its response.
func toPresetResponse(p domain.Preset) PresetResponse {
	point := func(pt domain.Point) PresetPointResponse {
		return PresetPointResponse{
			X: pt.X,
			Y: pt.Y,
		}
	}

	lanes := make(map[string]PresetLaneResponse, len(p.Lanes))
	for key, lane := range p.Lanes {
		lanes[key] = PresetLaneResponse{
			TL:     point(lane.TL),
			TR:     point(lane.TR),
			BR:     point(lane.BR),
			BL:     point(lane.BL),
			Locked: lane.Locked,
		}
	}

	return PresetResponse{
		Name:         p.Name,
		Comment:      p.Comment,
		FrameWidth:   p.FrameWidth,
		FrameHeight:  p.FrameHeight,
		TextSize:     p.TextSize,
		LaneGap:      p.LaneGap,
		ReverseLanes: p.ReverseLanes,
		TextAnchor:   string(p.TextAnchor),
		Lanes:        lanes,
	}
}

// toPreset maps the request to a preset.
func (r PresetSaveRequest) toPreset() domain.Preset {
	lanes := make(map[string]domain.LanePreset, len(r.Lanes))
	for key, lane := range r.Lanes {
		lanes[key] = domain.LanePreset{
			TL: domain.Point{
				X: lane.TL.X,
				Y: lane.TL.Y,
			},
			TR: domain.Point{
				X: lane.TR.X,
				Y: lane.TR.Y,
			},
			BR: domain.Point{
				X: lane.BR.X,
				Y: lane.BR.Y,
			},
			BL: domain.Point{
				X: lane.BL.X,
				Y: lane.BL.Y,
			},
			Locked: lane.Locked,
		}
	}

	return domain.Preset{
		Name:         r.Name,
		Comment:      r.Comment,
		FrameWidth:   r.FrameWidth,
		FrameHeight:  r.FrameHeight,
		TextSize:     r.TextSize,
		LaneGap:      r.LaneGap,
		ReverseLanes: r.ReverseLanes,
		TextAnchor:   r.TextAnchor,
		Lanes:        lanes,
	}
}

// laneNumberPattern matches a lane number as a key of a preset: a positive integer without leading zeros.
var laneNumberPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

// validateLaneNumber reports whether a string is a lane number, see laneNumberPattern.
func validateLaneNumber(fl validator.FieldLevel) bool {
	return laneNumberPattern.MatchString(fl.Field().String())
}

// presetDeleteHandler deletes a preset.
//
//	@Summary		Delete preset
//	@Description	Deletes the AR overlay preset with the given name (Admin only).
//	@Tags			presets
//	@Param			name	path	string	true	"Preset name"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/presets/{name} [delete]
func (a *API) presetDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.presetService.Delete(r.Context(), chi.URLParam(r, "name")); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
