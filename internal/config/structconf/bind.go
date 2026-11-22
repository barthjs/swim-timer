package structconf

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Leaf describes one setting, a scalar field of the configuration struct, and the names it is set by.
type Leaf struct {
	// Key is the dotted config-file key.
	Key string
	// Path is the dotted path of the JSON names, empty if the field or one of its sections has none.
	Path string
	// Flag is the command-line flag name without dashes in front.
	Flag string
	Env  string
	Type reflect.Type
	// Default is the raw default tag, empty for the zero value.
	Default string
	// Usage is the description from the usage tag.
	Usage string
	// Values are the allowed values from the values tag, empty if any value is allowed.
	Values []string
	// Secret reports whether the value must never be shown, see Redacted.
	Secret bool
}

// Description returns the usage text followed by the allowed values in parentheses,
// so flags, documentation, and templates describe a setting with the same words.
func (l Leaf) Description() string {
	if len(l.Values) == 0 {
		return l.Usage
	}
	return l.Usage + " (" + strings.Join(l.Values, ", ") + ")"
}

// Leaves returns every setting of the struct cfg points to, in field order.
func Leaves(cfg any) []Leaf {
	var leaves []Leaf
	eachLeaf(cfg, func(l Leaf) {
		leaves = append(leaves, l)
	})
	return leaves
}

var (
	// flagNameReplacer turns a dotted config key (log.max_age_days) into the flag name convention (log-max-age-days):
	// both dots and underscores become dashes.
	flagNameReplacer = strings.NewReplacer(".", "-", "_", "-")
)

// eachLeafValue walks a populated configuration and calls fn with each leaf's dotted key and current value,
// so the effective configuration can be serialized back to a file.
func eachLeafValue(cfg any, fn func(key string, value any)) {
	walkValue(reflect.ValueOf(cfg).Elem(), "", fn)
}

// walkValue calls fn with the key and value of every leaf field of the struct value v.
func walkValue(v reflect.Value, prefix string, fn func(key string, value any)) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)

		name, _, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		if name == "" || name == "-" {
			continue
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}

		fieldValue := v.Field(i)
		if fieldValue.Kind() == reflect.Struct {
			walkValue(fieldValue, key, fn)
			continue
		}

		fn(key, fieldValue.Interface())
	}
}

// RegisterFlags adds a flag with the default and description of its struct tags for every setting of cfg,
// and must run before the flag set is parsed.
func RegisterFlags(flags *pflag.FlagSet, cfg any) error {
	var outErr error
	eachLeaf(cfg, func(l Leaf) {
		if outErr != nil {
			return
		}
		value, err := parseDefault(l.Type, l.Default)
		if err != nil {
			outErr = fmt.Errorf("config %s: invalid default %q: %w", l.Key, l.Default, err)
			return
		}

		switch v := value.(type) {
		case time.Duration:
			flags.Duration(l.Flag, v, l.Description())
		case string:
			flags.String(l.Flag, v, l.Description())
		case bool:
			flags.Bool(l.Flag, v, l.Description())
		case int:
			flags.Int(l.Flag, v, l.Description())
		case float64:
			flags.Float64(l.Flag, v, l.Description())
		case []string:
			flags.StringSlice(l.Flag, v, l.Description())
		default:
			outErr = fmt.Errorf("config %s: unsupported field type %s", l.Key, l.Type)
		}
	})
	return outErr
}

// bindLeaves binds the default, the environment, and the flags, which may be nil, to every config key of cfg on v.
func bindLeaves(v *viper.Viper, cfg any, flags *pflag.FlagSet) error {
	var outErr error
	eachLeaf(cfg, func(l Leaf) {
		if outErr != nil {
			return
		}

		value, err := parseDefault(l.Type, l.Default)
		if err != nil {
			outErr = fmt.Errorf("config %s: invalid default %q: %w", l.Key, l.Default, err)
			return
		}
		// The default also makes the key known to viper, so Unmarshal resolves it from the environment
		// or a config file even when nothing else sets it.
		v.SetDefault(l.Key, value)

		// Bind the exact environment variable for this key. This is deliberately explicit rather than AutomaticEnv,
		// which would let a system variable that collides with a parent key (for example, MAIL) shadow a whole subtree.
		_ = v.BindEnv(l.Key, l.Env)

		if err := bindFileValue(v, l, flags); err != nil {
			outErr = err
			return
		}

		// A command that registers no flags gets the environment and the defaults only
		if flags == nil {
			return
		}
		if err := v.BindPFlag(l.Key, flags.Lookup(l.Flag)); err != nil {
			outErr = fmt.Errorf("config %s: binding flag %q: %w", l.Key, l.Flag, err)
		}
	})
	return outErr
}

// fileSuffix marks the environment variable that names a file holding a setting's value,
// the convention Docker and Kubernetes use for secrets.
const fileSuffix = "_FILE"

// bindFileValue sets the value of l from the trimmed content of the file its _FILE variable names,
// at the precedence of the environment.
func bindFileValue(v *viper.Viper, l Leaf, flags *pflag.FlagSet) error {
	fileEnv := l.Env + fileSuffix
	path := os.Getenv(fileEnv)
	if path == "" {
		return nil
	}

	// Setting both is unclear. An empty variable counts as unset, as in viper and Sources,
	// so a compose default that expands to nothing does not conflict with the _FILE variant.
	if os.Getenv(l.Env) != "" {
		return fmt.Errorf("config %s: set either %s or %s, not both", l.Key, l.Env, fileEnv)
	}

	if flags != nil {
		if flag := flags.Lookup(l.Flag); flag != nil && flag.Changed {
			return nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config %s: reading the file named by %s: %w", l.Key, fileEnv, err)
	}

	// Set has the highest precedence in viper, above flags, so it is only used when no flag was passed
	v.Set(l.Key, strings.TrimSpace(string(data)))
	return nil
}

// setDefaults registers the struct-tag default of every leaf of cfg with v.
// Unlike bindLeaves, it binds neither environment variables nor flags.
func setDefaults(v *viper.Viper, cfg any) error {
	var outErr error
	eachLeaf(cfg, func(l Leaf) {
		if outErr != nil {
			return
		}

		value, err := parseDefault(l.Type, l.Default)
		if err != nil {
			outErr = fmt.Errorf("config %s: invalid default %q: %w", l.Key, l.Default, err)
			return
		}
		v.SetDefault(l.Key, value)
	})
	return outErr
}

// eachLeaf calls fn for every setting of the struct behind cfg, including those of nested sections,
// and skips the fields tagged `mapstructure:"-"`, which hold values derived at load time.
func eachLeaf(cfg any, fn func(Leaf)) {
	walkStruct(reflect.TypeOf(cfg).Elem(), "", "", fn)
}

// walkStruct calls fn for every leaf field of the struct type t, recursing into nested structs.
// An empty pathPrefix below the top level means a section without a JSON name, so its fields get no path.
func walkStruct(t reflect.Type, prefix, pathPrefix string, fn func(Leaf)) {
	for field := range t.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		if name == "" || name == "-" {
			continue
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		path := jsonPath(field, prefix, pathPrefix)

		if field.Type.Kind() == reflect.Struct {
			walkStruct(field.Type, key, path, fn)
			continue
		}

		fn(Leaf{
			Key:     key,
			Path:    path,
			Flag:    flagNameReplacer.Replace(key),
			Env:     envName(key),
			Type:    field.Type,
			Default: field.Tag.Get("default"),
			Usage:   field.Tag.Get("usage"),
			Values:  splitValues(field.Tag.Get("values")),
			Secret:  field.Tag.Get(secretTag) == "true",
		})
	}
}

// jsonPath returns the dotted JSON path of field below a section with the given key prefix and path prefix,
// empty if the field or its section has no JSON name.
func jsonPath(field reflect.StructField, prefix, pathPrefix string) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return ""
	}
	if prefix == "" {
		return name
	}
	if pathPrefix == "" {
		return ""
	}
	return pathPrefix + "." + name
}

// envName turns a dotted config key (log.max_age_days) into its environment variable name (LOG_MAX_AGE_DAYS).
func envName(key string) string {
	return strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
}

// splitValues splits a comma-separated values tag into its trimmed entries, nil for an empty tag.
func splitValues(tag string) []string {
	if tag == "" {
		return nil
	}

	values := strings.Split(tag, ",")
	for i, value := range values {
		values[i] = strings.TrimSpace(value)
	}
	return values
}

// parseDefault converts a struct-tag default string into a value of the field's type.
// An empty tag yields that type's zero value.
func parseDefault(t reflect.Type, s string) (any, error) {
	if t == reflect.TypeFor[time.Duration]() {
		if s == "" {
			return time.Duration(0), nil
		}
		return time.ParseDuration(s)
	}

	switch t.Kind() {
	case reflect.String:
		return s, nil
	case reflect.Bool:
		if s == "" {
			return false, nil
		}
		return strconv.ParseBool(s)
	case reflect.Int:
		if s == "" {
			return 0, nil
		}
		return strconv.Atoi(s)
	case reflect.Float64:
		if s == "" {
			return 0.0, nil
		}
		return strconv.ParseFloat(s, 64)
	case reflect.Slice:
		if t.Elem().Kind() != reflect.String {
			return nil, fmt.Errorf("only []string slices are supported")
		}
		if s == "" {
			return []string{}, nil
		}
		return strings.Split(s, ","), nil
	default:
		return nil, fmt.Errorf("unsupported type %s", t)
	}
}
