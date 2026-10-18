package domain

// Distance represents the distance of an event (from LSTLONG.TXT).
// The French column names come from the timing software: "Longueur" is the length.
type Distance struct {
	ID     int    `csv:"idLength"`  // key referenced by Event.DistanceID
	Name   string `csv:"Longueur"`  // display name, e.g. "4x100 m"
	Length int    `csv:"Mlongueur"` // total distance in meters, all relay legs together, e.g. 400
	Relay  int    `csv:"Relais"`    // number of swimmers per team, 1 for an individual event
}
