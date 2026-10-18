package i18n

import (
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// dateLayouts holds the date and date-time layouts of a language. Languages without an entry use ISO 8601,
// which every reader understands. The layouts avoid month names where the language needs its own,
// since time.Format only writes English names.
var dateLayouts = map[string]struct{ date, dateTime string }{
	"de": {
		date:     "02.01.2006",
		dateTime: "02.01.2006, 15:04",
	},
	"en": {
		date:     "Jan 2, 2006",
		dateTime: "Jan 2, 2006, 3:04 PM",
	},
}

const (
	isoDate     = "2006-01-02"
	isoDateTime = "2006-01-02 15:04"
)

// FormatNumber writes a number with the digit grouping and decimal separator of the locale,
// such as "1.234,5" in German and "1,234.5" in English. Integers and floats are accepted.
func FormatNumber(locale string, value any) string {
	return message.NewPrinter(tag(locale)).Sprint(number.Decimal(value))
}

// FormatDate writes the date of t the way the locale writes dates, in the location of t.
func FormatDate(locale string, t time.Time) string {
	if layouts, ok := dateLayouts[base(locale)]; ok {
		return t.Format(layouts.date)
	}
	return t.Format(isoDate)
}

// FormatDateTime writes the date and time of t the way the locale writes them, in the location of t.
func FormatDateTime(locale string, t time.Time) string {
	if layouts, ok := dateLayouts[base(locale)]; ok {
		return t.Format(layouts.dateTime)
	}
	return t.Format(isoDateTime)
}

// base returns the language of a locale without region or script, such as "de" for "de-AT".
func base(locale string) string {
	b, _ := tag(locale).Base()
	return b.String()
}

// tag parses a locale, falling back to the default locale for an unknown or invalid one.
func tag(locale string) language.Tag {
	parsed, err := language.Parse(locale)
	if err != nil {
		return language.Make(DefaultLocale)
	}
	return parsed
}

// LocaleFuncs returns template functions that format for one locale: number, date, and datetime.
// The catalogs register them for the locale of each catalog, so a message writes "{{ number .Count }}".
func LocaleFuncs(locale string) map[string]any {
	return map[string]any{
		"number":   func(value any) string { return FormatNumber(locale, value) },
		"date":     func(t time.Time) string { return FormatDate(locale, t) },
		"datetime": func(t time.Time) string { return FormatDateTime(locale, t) },
	}
}
