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
	// Key is the dotted config-file key, such as log.max_age_days.
	Key string
	// Path is the dotted path of the JSON names, such as log.maxAgeDays, the way the setting leaves the process as JSON.
	// It is empty if the field or one of its sections has no JSON name.
	Path string
	// Flag is the command-line flag name without dashes in front, such as log-max-age-days.
	Flag string
	// Env is the environment variable name, such as LOG_MAX_AGE_DAYS.
	Env string
	// Type is the Go type of the field.
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

// Description returns the usage text followed by the allowed values, such as "Log level (debug, info)",
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
	// flagNameReplacer turns a dotted config key (log.max_age_days) into the flag name convention
	// (log-max-age-days): both dots and underscores become dashes.
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

// RegisterFlags adds a command-line flag for every configuration field of cfg to flags,
// using the field's default and description from its struct tags. It must be called before
// the flag set is parsed. Binding these flags to the config keys happens later in Load.
// The caller registers flags that are not settings, such as --config.
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

// bindLeaves registers the struct-tag default and the environment binding for every config key of cfg on v.
// Registering the default also makes the key known to viper, so Unmarshal resolves it from the environment
// or a config file even when nothing else sets it. When flags are non-nil, each key is additionally bound
// to its already-registered flag, so a flag passed on the command line overrides the environment, the config file,
// and the default. A command that registers no flags passes nil and gets the environment and the defaults only.
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
		v.SetDefault(l.Key, value)

		// Bind the exact environment variable for this key. This is deliberately explicit rather than AutomaticEnv,
		// which would let a system variable that collides with a parent key (for example, MAIL) shadow a whole subtree.
		_ = v.BindEnv(l.Key, l.Env)

		if err := bindFileValue(v, l, flags); err != nil {
			outErr = err
			return
		}

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
// the convention Docker and Kubernetes use for secrets, for example APP_SECRET_FILE.
const fileSuffix = "_FILE"

// bindFileValue sets the value of l from the file its _FILE variable names, at the precedence of the environment.
// It overrides a config file and the default, while a flag passed on the command line still wins.
// Surrounding whitespace, such as the trailing newline of a secret file, is removed.
// Setting both the variable and its _FILE variant is an error, since it is unclear which one is meant.
// An empty variable counts as unset, as in viper and Sources, so a compose default such as ${APP_SECRET:-}
// does not conflict with the _FILE variant.
func bindFileValue(v *viper.Viper, l Leaf, flags *pflag.FlagSet) error {
	fileEnv := l.Env + fileSuffix
	path := os.Getenv(fileEnv)
	if path == "" {
		return nil
	}

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

// eachLeaf walks the struct behind cfg and calls fn for every scalar field, building the dotted key
// from the mapstructure tags and the path from the JSON tags. Nested structs are recursed into.
// Fields tagged `mapstructure:"-"` hold values derived at load time and are skipped.
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
