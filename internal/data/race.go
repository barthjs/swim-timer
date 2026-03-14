package data

// Race represents a scheduled race (from LSTRACE.TXT)
type Race struct {
	Event      int       `csv:"event"`
	Round      int       `csv:"round"`
	NBHeat     int       `csv:"nbHeat"`
	DistanceID int       `csv:"idLen"`
	Distance   *Distance `csv:"-"`
	StyleID    int       `csv:"idStyle"`
	Style      *Style    `csv:"-"`
	AbCat      string    `csv:"abCat"`
	Category   *Category `csv:"-"`
	Date       string    `csv:"date"`
	Time       string    `csv:"time"`
}
