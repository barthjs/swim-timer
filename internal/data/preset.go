package data

type TextAnchor string

const (
	TextAnchorLeft  TextAnchor = "left"
	TextAnchorRight TextAnchor = "right"
)

// Point represents a coordinate in a preset.
type Point struct {
	X int `json:"x" validate:"min=1,max=10000"`
	Y int `json:"y" validate:"min=1,max=10000"`
}

// LanePreset represents a lane configuration in a preset.
type LanePreset struct {
	TL     Point `json:"tl" validate:"required"`    // Top Left
	TR     Point `json:"tr" validate:"required"`    // Top Right
	BR     Point `json:"br" validate:"required"`    // Bottom Right
	BL     Point `json:"bl" validate:"required"`    // Bottom Left
	Locked bool  `json:"locked" validate:"boolean"` // Locks the movement in the Editor
}

// Preset is used for parsing and validating preset save requests.
type Preset struct {
	Name          string                `json:"name" validate:"required,min=1,max=255"`
	Comment       *string               `json:"comment" validate:"omitempty,min=1,max=1000"`
	VirtualWidth  int                   `json:"virtualWidth" validate:"required,min=1,max=100000"`
	VirtualHeight int                   `json:"virtualHeight" validate:"required,min=1,max=100000"`
	ReverseLanes  bool                  `json:"reverseLanes" validate:"boolean"`                 // true maps visual top lane to the highest race lane number
	TextAnchor    TextAnchor            `json:"textAnchor" validate:"required,oneof=left right"` // layout direction based on camera position
	Lanes         map[string]LanePreset `json:"lanes" validate:"required,dive"`
}
