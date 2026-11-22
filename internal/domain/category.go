package domain

// Category represents a swimming category (from LSTCAT.TXT).
type Category struct {
	Name         string // display name, e.g. "Women"
	Abbreviation string // key referenced by Event.AbCat and Competitor.AbCat, e.g. "D"
}
