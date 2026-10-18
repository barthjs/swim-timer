package structconf

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tagConfig exercises every field type and tag combination the package supports.
type tagConfig struct {
	Demo tagDemo `mapstructure:"demo" json:"demo"`
}

// tagDemo is a sample configuration section that covers the supported field types and tags.
type tagDemo struct {
	Name     string        `mapstructure:"name" json:"name" default:"demo" usage:"Name of the demo"`
	Debug    bool          `mapstructure:"debug" json:"debug" default:"true" usage:"Debug mode"`
	Ratio    float64       `mapstructure:"ratio" json:"ratio" default:"0.5" usage:"A ratio"`
	Timeout  time.Duration `mapstructure:"timeout" json:"timeout" default:"1m30s" usage:"A timeout"`
	Tags     []string      `mapstructure:"tags" json:"tags" default:"a,b" usage:"Some tags"`
	Server   tagServer     `mapstructure:"server" json:"server"`
	Derived  int           `mapstructure:"-"`
	Untagged string
}

// tagServer is a nested sample configuration section.
type tagServer struct {
	Port     int    `mapstructure:"port" json:"port" default:"8080" usage:"Port"`
	LogLevel string `mapstructure:"log_level" json:"logLevel" default:"warn" values:"debug, info,warn,error" usage:"Log level"`
	NoValue  string `mapstructure:"no_value" usage:"Without a default"`
}

func TestLeaf_Description(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		leaf Leaf
		want string
	}{
		{
			name: "returns the usage without values",
			leaf: Leaf{
				Usage: "Port",
			},
			want: "Port",
		},
		{
			name: "lists the allowed values",
			leaf: Leaf{
				Usage:  "Log level",
				Values: []string{"debug", "warn"},
			},
			want: "Log level (debug, warn)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.leaf.Description()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLeaves(t *testing.T) {
	t.Parallel()

	t.Run("returns the leaves of nested sections", func(t *testing.T) {
		t.Parallel()

		leaves := tagLeaves()

		keys := make([]string, 0, len(leaves))
		for key := range leaves {
			keys = append(keys, key)
		}
		assert.ElementsMatch(t, []string{
			"demo.name", "demo.debug", "demo.ratio", "demo.timeout", "demo.tags",
			"demo.server.port", "demo.server.log_level", "demo.server.no_value",
		}, keys, "nested structs are recursed into, fields tagged - or without a tag are skipped")
	})

	t.Run("returns the names, the default, and the type of each leaf", func(t *testing.T) {
		t.Parallel()

		leaves := tagLeaves()

		tests := []struct {
			key      string
			path     string
			envName  string
			flagName string
			def      string
			usage    string
			typ      reflect.Type
		}{
			{
				key:      "demo.name",
				path:     "demo.name",
				envName:  "DEMO_NAME",
				flagName: "demo-name",
				def:      "demo",
				usage:    "Name of the demo",
				typ:      reflect.TypeFor[string](),
			},
			{
				key:      "demo.server.log_level",
				path:     "demo.server.logLevel",
				envName:  "DEMO_SERVER_LOG_LEVEL",
				flagName: "demo-server-log-level",
				def:      "warn",
				usage:    "Log level",
				typ:      reflect.TypeFor[string](),
			},
			{
				key:      "demo.timeout",
				path:     "demo.timeout",
				envName:  "DEMO_TIMEOUT",
				flagName: "demo-timeout",
				def:      "1m30s",
				usage:    "A timeout",
				typ:      reflect.TypeFor[time.Duration](),
			},
			{
				key:      "demo.server.no_value",
				path:     "",
				envName:  "DEMO_SERVER_NO_VALUE",
				flagName: "demo-server-no-value",
				def:      "",
				usage:    "Without a default",
				typ:      reflect.TypeFor[string](),
			},
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				t.Parallel()

				l, ok := leaves[tt.key]

				require.True(t, ok)
				assert.Equal(t, tt.path, l.Path, "a field without a JSON name has no path")
				assert.Equal(t, tt.envName, l.Env)
				assert.Equal(t, tt.flagName, l.Flag)
				assert.Equal(t, tt.def, l.Default)
				assert.Equal(t, tt.usage, l.Usage)
				assert.Equal(t, tt.typ, l.Type)
			})
		}
	})

	t.Run("reads the values and secrets", func(t *testing.T) {
		t.Parallel()

		type withSecret struct {
			Token string `mapstructure:"token" secret:"true" usage:"Token"`
		}

		leaves := tagLeaves()

		assert.Equal(t, []string{"debug", "info", "warn", "error"}, leaves["demo.server.log_level"].Values, "entries are trimmed")
		assert.Nil(t, leaves["demo.server.port"].Values)
		assert.False(t, leaves["demo.name"].Secret)
		assert.True(t, Leaves(&withSecret{})[0].Secret)
	})

	t.Run("leaves the path empty below a section without a JSON name", func(t *testing.T) {
		t.Parallel()

		type section struct {
			Port int `mapstructure:"port" json:"port"`
		}
		type withoutJSONSection struct {
			Section section `mapstructure:"section"`
		}

		leaves := Leaves(&withoutJSONSection{})

		require.Len(t, leaves, 1)
		assert.Empty(t, leaves[0].Path, "a field below a section without a JSON name has no path")
	})
}

// tagLeaves returns the leaves of the sample configuration by key.
func tagLeaves() map[string]Leaf {
	leaves := map[string]Leaf{}
	for _, l := range Leaves(&tagConfig{}) {
		leaves[l.Key] = l
	}
	return leaves
}

func TestEachLeafValue(t *testing.T) {
	t.Parallel()

	t.Run("returns the values without derived fields", func(t *testing.T) {
		t.Parallel()

		cfg := tagConfig{
			Demo: tagDemo{
				Name: "x",
				Server: tagServer{
					Port: 9,
				},
				Derived: 5,
			},
		}
		values := map[string]any{}

		eachLeafValue(&cfg, func(key string, value any) {
			values[key] = value
		})

		assert.Equal(t, "x", values["demo.name"])
		assert.Equal(t, 9, values["demo.server.port"])
		assert.NotContains(t, values, "demo.derived", "derived values are not written back")
		assert.Len(t, values, 8)
	})
}

func TestRegisterFlags(t *testing.T) {
	t.Parallel()

	t.Run("registers a flag with the default of each leaf", func(t *testing.T) {
		t.Parallel()

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		require.NoError(t, RegisterFlags(flags, &tagConfig{}))

		tests := map[string]string{
			"demo-name":             "demo",
			"demo-debug":            "true",
			"demo-ratio":            "0.5",
			"demo-timeout":          "1m30s",
			"demo-tags":             "[a,b]",
			"demo-server-port":      "8080",
			"demo-server-log-level": "warn",
			"demo-server-no-value":  "",
		}
		for name, def := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				flag := flags.Lookup(name)

				require.NotNil(t, flag)
				assert.Equal(t, def, flag.DefValue)
				assert.NotEmpty(t, flag.Usage)
			})
		}
		assert.Equal(t, "Log level (debug, info, warn, error)", flags.Lookup("demo-server-log-level").Usage, "the usage lists the allowed values")
	})

	t.Run("rejects an invalid default", func(t *testing.T) {
		t.Parallel()

		type invalidDefault struct {
			Port int `mapstructure:"port" default:"eighty"`
		}
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		err := RegisterFlags(flags, &invalidDefault{})

		assert.ErrorContains(t, err, "port")
	})

	t.Run("rejects an unsupported type", func(t *testing.T) {
		t.Parallel()

		type unsupportedType struct {
			Size int64 `mapstructure:"size" default:"1"`
		}
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		err := RegisterFlags(flags, &unsupportedType{})

		assert.ErrorContains(t, err, "size")
	})
}

func TestBindLeaves(t *testing.T) {
	t.Run("applies the default, the environment, and the flags in order", func(t *testing.T) {
		t.Setenv("DEMO_SERVER_PORT", "9100")
		t.Setenv("DEMO_SERVER_LOG_LEVEL", "error")
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		require.NoError(t, RegisterFlags(flags, &tagConfig{}))
		require.NoError(t, flags.Parse([]string{"--demo-server-log-level=debug"}))
		v := viper.New()

		require.NoError(t, bindLeaves(v, &tagConfig{}, flags))

		var cfg tagConfig
		require.NoError(t, v.Unmarshal(&cfg))
		assert.Equal(t, "demo", cfg.Demo.Name, "the default applies without another value")
		assert.Equal(t, 90*time.Second, cfg.Demo.Timeout)
		assert.Equal(t, []string{"a", "b"}, cfg.Demo.Tags)
		assert.Equal(t, 9100, cfg.Demo.Server.Port, "the environment overrides the default")
		assert.Equal(t, "debug", cfg.Demo.Server.LogLevel, "a flag overrides the environment")
	})

	t.Run("reads the environment without flags", func(t *testing.T) {
		t.Setenv("DEMO_NAME", "from env")
		v := viper.New()

		require.NoError(t, bindLeaves(v, &tagConfig{}, nil))

		var cfg tagConfig
		require.NoError(t, v.Unmarshal(&cfg))
		assert.Equal(t, "from env", cfg.Demo.Name)
		assert.Equal(t, 8080, cfg.Demo.Server.Port)
	})

	t.Run("reads file variables", func(t *testing.T) {
		dir := t.TempDir()
		nameFile := filepath.Join(dir, "name")
		require.NoError(t, os.WriteFile(nameFile, []byte("  from-file\n"), 0o600))
		portFile := filepath.Join(dir, "port")
		require.NoError(t, os.WriteFile(portFile, []byte("9090\n"), 0o600))
		t.Setenv("DEMO_NAME_FILE", nameFile)
		t.Setenv("DEMO_SERVER_PORT_FILE", portFile)
		v := viper.New()
		var cfg tagConfig

		require.NoError(t, bindLeaves(v, &cfg, nil))

		require.NoError(t, v.Unmarshal(&cfg))
		assert.Equal(t, "from-file", cfg.Demo.Name, "surrounding whitespace is removed")
		assert.Equal(t, 9090, cfg.Demo.Server.Port, "the value is converted like an environment variable")
	})

	t.Run("prefers a flag over a file variable", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "name")
		require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o600))
		t.Setenv("DEMO_NAME_FILE", file)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		require.NoError(t, RegisterFlags(flags, &tagConfig{}))
		require.NoError(t, flags.Parse([]string{"--demo-name=from-flag"}))
		v := viper.New()
		var cfg tagConfig

		require.NoError(t, bindLeaves(v, &cfg, flags))

		require.NoError(t, v.Unmarshal(&cfg))
		assert.Equal(t, "from-flag", cfg.Demo.Name)
	})

	t.Run("keeps a file variable next to an empty variable", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "name")
		require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o600))
		t.Setenv("DEMO_NAME", "")
		t.Setenv("DEMO_NAME_FILE", file)
		v := viper.New()
		var cfg tagConfig

		require.NoError(t, bindLeaves(v, &cfg, nil), "an empty variable counts as unset")

		require.NoError(t, v.Unmarshal(&cfg))
		assert.Equal(t, "from-file", cfg.Demo.Name)
	})

	t.Run("rejects a variable set both directly and as a file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "name")
		require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o600))
		t.Setenv("DEMO_NAME", "from-env")
		t.Setenv("DEMO_NAME_FILE", file)
		var cfg tagConfig

		err := bindLeaves(viper.New(), &cfg, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "DEMO_NAME_FILE")
	})

	t.Run("rejects a missing file", func(t *testing.T) {
		t.Setenv("DEMO_NAME_FILE", filepath.Join(t.TempDir(), "missing"))
		var cfg tagConfig

		err := bindLeaves(viper.New(), &cfg, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "demo.name")
	})
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()

	t.Run("rejects an invalid default", func(t *testing.T) {
		t.Parallel()

		var cfg struct {
			Port int `mapstructure:"port" default:"not-a-number" usage:"Port"`
		}

		err := setDefaults(viper.New(), &cfg)

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid default")
	})
}

func TestParseDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		typ     reflect.Type
		in      string
		want    any
		wantErr bool
	}{
		{
			name: "parses a string",
			typ:  reflect.TypeFor[string](),
			in:   "text",
			want: "text",
		},
		{
			name: "keeps an empty string",
			typ:  reflect.TypeFor[string](),
			in:   "",
			want: "",
		},
		{
			name: "parses a bool",
			typ:  reflect.TypeOf(false),
			in:   "true",
			want: true,
		},
		{
			name: "returns false for an empty bool",
			typ:  reflect.TypeOf(false),
			in:   "",
			want: false,
		},
		{
			name: "parses an int",
			typ:  reflect.TypeFor[int](),
			in:   "42",
			want: 42,
		},
		{
			name: "returns zero for an empty int",
			typ:  reflect.TypeFor[int](),
			in:   "",
			want: 0,
		},
		{
			name: "parses a float",
			typ:  reflect.TypeFor[float64](),
			in:   "0.25",
			want: 0.25,
		},
		{
			name: "returns zero for an empty float",
			typ:  reflect.TypeFor[float64](),
			in:   "",
			want: 0.0,
		},
		{
			name: "parses a duration",
			typ:  reflect.TypeFor[time.Duration](),
			in:   "2s",
			want: 2 * time.Second,
		},
		{
			name: "returns zero for an empty duration",
			typ:  reflect.TypeFor[time.Duration](),
			in:   "",
			want: time.Duration(0),
		},
		{
			name: "splits a string slice",
			typ:  reflect.TypeFor[[]string](),
			in:   "a,b",
			want: []string{"a", "b"},
		},
		{
			name: "returns an empty string slice",
			typ:  reflect.TypeFor[[]string](),
			in:   "",
			want: []string{},
		},
		{
			name:    "rejects an invalid bool",
			typ:     reflect.TypeOf(false),
			in:      "maybe",
			wantErr: true,
		},
		{
			name:    "rejects an invalid int",
			typ:     reflect.TypeFor[int](),
			in:      "x",
			wantErr: true,
		},
		{
			name:    "rejects an invalid duration",
			typ:     reflect.TypeFor[time.Duration](),
			in:      "soon",
			wantErr: true,
		},
		{
			name:    "rejects an unsupported slice",
			typ:     reflect.TypeFor[[]int](),
			in:      "1",
			wantErr: true,
		},
		{
			name:    "rejects an unsupported type",
			typ:     reflect.TypeFor[int64](),
			in:      "1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDefault(tt.typ, tt.in)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
