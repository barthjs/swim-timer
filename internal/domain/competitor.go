package domain

// Competitor represents a competitor of the meet (from LSTCONC.TXT).
type Competitor struct {
	ID        int    `csv:"id"`  // key of the competitor, referenced by StartEntry.CompetitorID
	BIB       int    `csv:"bib"` // start number, not used for the lookup since it may be empty
	LastName  string `csv:"lastname"`
	FirstName string `csv:"firstname"`
	BirthYear string `csv:"birthyear"` // year of birth, kept as text since it may be empty
	AbNat     string `csv:"abNat"`     // club, or the nation at international meets
	AbCat     string `csv:"abCat"`     // category abbreviation, see Category.Abbreviation
}

// Club returns the club name of the competitor.
func (c Competitor) Club() string {
	return c.AbNat
}
