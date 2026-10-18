package structconf

import (
	"strings"
	"time"
)

// ToMap returns a populated configuration as nested maps keyed like the config file. Durations are written as text,
// such as "30s", the way they are set. Pass a redacted copy (see Redacted) when the output may be shown to a person.
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
