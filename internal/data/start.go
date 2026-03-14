package data

// StartEntry represents a single entry in the start list (from LSTSTART.TXT)
type StartEntry struct {
	Event  int     `csv:"event"`
	Round  int     `csv:"round"`
	Heat   int     `csv:"heat"`
	Lane   int     `csv:"lane"`
	Relais int     `csv:"relais"`
	IDBIB  int     `csv:"idBib"`
	Person *Person `csv:"-"`
}
