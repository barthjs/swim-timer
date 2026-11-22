package structconf

import (
	"strings"
	"time"
)

// ToMap returns a configuration as nested maps keyed like the config file, with durations as text,
// and takes a redacted copy when a person may see the output.
func ToMap(cfg any) map[string]any {
	result := map[string]any{}
	eachLeafValue(cfg, func(key string, value any) {
		if duration, ok := value.(time.Duration); ok {
			value = duration.String()
		}

		node := result
		parts := strings.Split(key, ".")
		for _, part := range parts[:len(parts)-1] {
			child, ok := node[part].(map[string]any)
			if !ok {
				child = map[string]any{}
				node[part] = child
			}
			node = child
		}
		node[parts[len(parts)-1]] = value
	})
	return result
}
