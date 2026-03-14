package data

// Category represents a swimming category (from LSTCAT.TXT)
type Category struct {
	Name         string `csv:"Category"`
	Abbreviation string `csv:"AbrevCat"`
}
