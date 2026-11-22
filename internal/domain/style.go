package domain

// Style represents a swimming style (from LSTSTYLE.TXT).
type Style struct {
	ID           int    // key referenced by Event.StyleID
	Name         string // display name, e.g. "Freestyle"
	Abbreviation string // short name, e.g. "FR"
}
