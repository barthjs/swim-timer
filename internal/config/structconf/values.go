package structconf

import (
	"fmt"
	"slices"
	"strings"
)

// CheckValues reports every setting of the struct cfg points to whose value is not one of the values of its tag.
func CheckValues(cfg any) error {
	allowed := map[string][]string{}
	for _, leaf := range Leaves(cfg) {
		if len(leaf.Values) > 0 {
			allowed[leaf.Key] = leaf.Values
		}
	}

	var p Problems
	eachLeafValue(cfg, func(key string, value any) {
		values, ok := allowed[key]
		if !ok {
			return
		}
		if text := fmt.Sprint(value); !slices.Contains(values, text) {
			p.Addf("%s must be one of %s, got %q", key, strings.Join(values, ", "), text)
		}
	})
	return p.Err()
}
