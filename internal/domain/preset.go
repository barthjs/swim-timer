package domain

// TextAnchor aligns the text of a lane.
type TextAnchor string

const (
	TextAnchorLeft  TextAnchor = "left"
	TextAnchorRight TextAnchor = "right"
)

// Point represents a position in the output image of the AR overlay as a fraction of its width and height,
// from 0 at the top left to 1 at the bottom right, so a preset fits every resolution of its aspect ratio.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// LanePreset represents a lane configuration in a preset.
type LanePreset struct {
	TL     Point `json:"tl"`     // Top Left
	TR     Point `json:"tr"`     // Top Right
	BR     Point `json:"br"`     // Bottom Right
	BL     Point `json:"bl"`     // Bottom Left
	Locked bool  `json:"locked"` // Locks the movement in the Editor
}

// Preset describes how the AR overlay maps the lane texts onto the camera image.
// It carries JSON tags because the preset store persists it as JSON.
type Preset struct {
	Name    string  `json:"name"`
	Comment *string `json:"comment"`
	// FrameWidth and FrameHeight are the reference resolution of the output image.
	// They set the aspect ratio the editor draws the preset in and the pixels it shows,
	// while the lanes are stored as fractions.
	FrameWidth  int `json:"frameWidth"`
	FrameHeight int `json:"frameHeight"`
	// TextSize is the height of the text in percent of the height of a lane.
	TextSize int `json:"textSize"`
	// LaneGap is the gap between two lanes in percent of the height of a lane, which the editor keeps
	// when it distributes the lanes between the first and the last lane.
	LaneGap int `json:"laneGap"`
	// ReverseLanes maps the visual top lane to the highest lane number of the heat.
	ReverseLanes bool `json:"reverseLanes"`
	// TextAnchor is the layout direction based on the camera position.
	TextAnchor TextAnchor            `json:"textAnchor"`
	Lanes      map[string]LanePreset `json:"lanes"`
}
