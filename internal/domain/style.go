package domain

// Style represents a swimming style (from LSTSTYLE.TXT).
type Style struct {
	ID           int    `csv:"idStyle"`    // key referenced by Event.StyleID
	Name         string `csv:"Style"`      // display name, e.g. "Freestyle"
	Abbreviation string `csv:"StyleAbrév"` // short name, e.g. "FR"
}
