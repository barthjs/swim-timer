package data

// Person represents a person in a swimming race (from LSTCONC.TXT)
type Person struct {
	ID        int    `csv:"id"`
	BIB       int    `csv:"bib"`
	LastName  string `csv:"lastname"`
	FirstName string `csv:"firstname"`
	BirthYear string `csv:"birthyear"`
	AbNat     string `csv:"abNat"` // Clubname
	AbCat     string `csv:"abCat"`
}

func (p Person) GetClub() string {
	return p.AbNat
}
