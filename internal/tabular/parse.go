package tabular

import (
	"strconv"
	"strings"
	"time"
)

// ParseString returns the text as it is.
func ParseString(raw string) (string, error) {
	return raw, nil
}

// ParseInt parses a whole number, failing with the tag "number".
func ParseInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &FieldError{
			Tag: "number",
		}
	}
	return value, nil
}

// ParseDecimal parses a number with a decimal point or a decimal comma, failing with the tag "number".
func ParseDecimal(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.Replace(raw, ",", ".", 1), 64)
	if err != nil {
		return 0, &FieldError{
			Tag: "number",
		}
	}
	return value, nil
}

// ParseBool parses "true" or "false", failing with the tag "boolean",
// which a column offers as choices for a mapping to translate.
func ParseBool(raw string) (bool, error) {
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, &FieldError{
			Tag: "boolean",
		}
	}
}

// FormatBool returns the text of a value ParseBool reads.
func FormatBool(value bool) string {
	return strconv.FormatBool(value)
}

// ParseDate returns a parser for dates in the first of the layouts that fits, failing with the tag "date"
// and the first layout as its parameter.
func ParseDate(layouts ...string) func(raw string) (time.Time, error) {
	return func(raw string) (time.Time, error) {
		for _, layout := range layouts {
			if value, err := time.Parse(layout, raw); err == nil {
				return value, nil
			}
		}
		param := ""
		if len(layouts) > 0 {
			param = layouts[0]
		}
		return time.Time{}, &FieldError{
			Tag:   "date",
			Param: param,
		}
	}
}
