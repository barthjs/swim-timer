package i18n

import (
	"maps"

	"golang.org/x/text/feature/plural"
)

// CountKey is the data key TN adds the count under, so a message writes "{{.Count}}".
const CountKey = "Count"

// pluralForm returns the CLDR plural category of a whole number in the locale, such as "one" or "other".
func pluralForm(locale string, count int) string {
	n := count
	if n < 0 {
		n = -n
	}

	switch plural.Cardinal.MatchPlural(tag(locale), n, 0, 0, 0, 0) {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	default:
		return "other"
	}
}

// withCount returns the data with the count added under CountKey. A map gets a copy with the count
// unless it already holds the key, nil becomes a map with only the count, and any other value is returned
// as it is, so a struct must carry the count itself.
func withCount(data any, count int) any {
	switch value := data.(type) {
	case nil:
		return map[string]any{CountKey: count}
	case map[string]any:
		if _, ok := value[CountKey]; ok {
			return value
		}
		copied := make(map[string]any, len(value)+1)
		maps.Copy(copied, value)
		copied[CountKey] = count
		return copied
	default:
		return data
	}
}
