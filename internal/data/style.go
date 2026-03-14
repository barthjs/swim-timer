package data

// Style represents a swimming style (from LSTSTYLE.TXT)
type Style struct {
	ID           int    `csv:"idStyle"`
	Name         string `csv:"Style"`
	Abbreviation string `csv:"StyleAbrev"`
}
