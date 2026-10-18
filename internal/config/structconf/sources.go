package structconf

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// SourceKind names where the value of a setting comes from.
type SourceKind string

const (
	SourceFlag    SourceKind = "flag"
	SourceEnv     SourceKind = "env"
	SourceFile    SourceKind = "file"
	SourceDefault SourceKind = "default"
)

// Source is where the value of a setting comes from.
type Source struct {
	Kind SourceKind
	// Name is the flag, the environment variable, or the file path the value was read from, empty for the default.
	Name string
}

// String returns the source as shown to a person, such as "env LOG_LEVEL" or "default".
func (s Source) String() string {
	if s.Name == "" {
		return string(s.Kind)
	}
	return string(s.Kind) + " " + s.Name
}

// Sources reports for every setting of the struct cfg points to where Load takes its value from,
// keyed by the dotted key. It applies the same precedence as Load: a flag passed on the command line,
// then the environment variable or its _FILE variant, then the config file, then the default.
func Sources(appName string, flags *pflag.FlagSet, cfg any) (map[string]Source, error) {
	v := viper.New()
	if err := readFile(v, appName, flags); err != nil {
		return nil, err
	}
	file := v.ConfigFileUsed()

	sources := map[string]Source{}
	for _, l := range Leaves(cfg) {
		sources[l.Key] = sourceOf(l, flags, v, file)
	}
	return sources, nil
}

// PathSources reports the same sources as Sources, keyed by the JSON path of each setting, such as "log.maxAgeDays",
// the way a configuration leaves the process as JSON. Settings without a JSON path are left out.
func PathSources(appName string, flags *pflag.FlagSet, cfg any) (map[string]Source, error) {
	sources, err := Sources(appName, flags, cfg)
	if err != nil {
		return nil, err
	}

	byPath := map[string]Source{}
	for _, l := range Leaves(cfg) {
		if l.Path == "" {
			continue
		}
		byPath[l.Path] = sources[l.Key]
	}
	return byPath, nil
}

// sourceOf returns the source of one setting. An empty environment variable counts as unset, like in Load.
func sourceOf(l Leaf, flags *pflag.FlagSet, v *viper.Viper, file string) Source {
	if flags != nil {
		if flag := flags.Lookup(l.Flag); flag != nil && flag.Changed {
			return Source{
				Kind: SourceFlag,
				Name: "--" + l.Flag,
			}
		}
	}

	if os.Getenv(l.Env) != "" {
		return Source{
			Kind: SourceEnv,
			Name: l.Env,
		}
	}
	if os.Getenv(l.Env+fileSuffix) != "" {
		return Source{
			Kind: SourceEnv,
			Name: l.Env + fileSuffix,
		}
	}

	if file != "" && v.InConfig(l.Key) {
		return Source{
			Kind: SourceFile,
			Name: file,
		}
	}

	return Source{
		Kind: SourceDefault,
	}
}
