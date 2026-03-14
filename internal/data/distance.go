package data

// Distance represents a race distance (from LSTLONG.TXT)
type Distance struct {
	ID     int    `csv:"idLength"`
	Name   string `csv:"Longueur"`
	Length int    `csv:"Mlongueur"`
	Relay  int    `csv:"Relais"`
}
