package domain

// Competitor represents a competitor of the meet (from LSTCONC.TXT).
type Competitor struct {
	ID        int // key of the competitor, referenced by StartEntry.CompetitorID
	BIB       int // start number, not used for the lookup since it may be empty
	LastName  string
	FirstName string
	BirthYear string // year of birth, kept as text since it may be empty
	AbNat     string // club, or the nation at international meets
	AbCat     string // category abbreviation, see Category.Abbreviation
}

// Club returns the club name of the competitor.
func (c Competitor) Club() string {
	return c.AbNat
}
