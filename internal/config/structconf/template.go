package structconf

import (
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Template writes a YAML config file with every setting of the struct cfg points to, nested by section.
// Each setting carries its description as a comment and its default as the value,
// so the file documents itself and loads to the same values as the defaults. Secrets are left empty.
func Template(w io.Writer, cfg any) error {
	var sb strings.Builder
	var previous []string

	for _, l := range Leaves(cfg) {
		parts := strings.Split(l.Key, ".")
		sections := parts[:len(parts)-1]

		common := commonPrefix(previous, sections)
		if len(previous) > 0 && common == 0 {
			_, _ = sb.WriteString("\n")
		}
		for depth := common; depth < len(sections); depth++ {
			_, _ = fmt.Fprintf(&sb, "%s%s:\n", indent(depth), sections[depth])
		}
		previous = sections

		value, err := templateValue(l)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(&sb, "%s# %s\n", indent(len(sections)), l.Description())
		_, _ = fmt.Fprintf(&sb, "%s%s: %s\n", indent(len(sections)), parts[len(parts)-1], value)
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// templateValue returns the default of a setting as a YAML scalar or flow sequence.
func templateValue(l Leaf) (string, error) {
	if l.Secret {
		return `""`, nil
	}

	value, err := parseDefault(l.Type, l.Default)
	if err != nil {
		return "", fmt.Errorf("config %s: invalid default %q: %w", l.Key, l.Default, err)
	}
	// A duration is written as text, such as 30s, the way it is set
	if stringer, ok := value.(fmt.Stringer); ok {
		value = stringer.String()
	}

	var sb strings.Builder
	encoder := yaml.NewEncoder(&sb)
	if list, ok := value.([]string); ok {
		node := &yaml.Node{}
		if err := node.Encode(list); err != nil {
			return "", err
		}
		node.Style = yaml.FlowStyle
		value = node
	}
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// commonPrefix returns how many leading sections a and b share.
func commonPrefix(a, b []string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// indent returns the indentation of a nesting depth.
func indent(depth int) string {
	return strings.Repeat("  ", depth)
}
