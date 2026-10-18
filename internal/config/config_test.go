package config

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/appdirs"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
)

// rejection is a change to a valid configuration that Validate must reject with an error naming wantErr.
type rejection struct {
	name    string
	change  func(*Config)
	wantErr string
}

// assertRejects applies each change to the default configuration and checks that Validate rejects it.
func assertRejects(t *testing.T, tests []rejection) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig(t)
			tt.change(&cfg)

			err := Validate(&cfg)

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("documents every setting", func(t *testing.T) {
		t.Parallel()

		for _, l := range structconf.Leaves(&Config{}) {
			t.Run(l.Key, func(t *testing.T) {
				t.Parallel()

				assert.NotEmpty(t, l.Usage, "every setting is documented")
				assert.NotEmpty(t, l.Path, "every setting has a JSON path, so its source can be shown")
				// Values derived at load time, such as the parsed log level, are not settings
				assert.NotEqual(t, reflect.TypeFor[zerolog.Level](), l.Type)
				assert.NotEqual(t, reflect.TypeFor[[]*net.IPNet](), l.Type)
			})
		}
	})

	t.Run("parses every default", func(t *testing.T) {
		t.Parallel()

		// Registering the flags parses every default, so an unparseable default fails here
		err := structconf.RegisterFlags(pflag.NewFlagSet("test", pflag.ContinueOnError), &Config{})

		require.NoError(t, err)
	})

	t.Run("loads the template to the defaults", func(t *testing.T) {
		t.Parallel()

		var sb strings.Builder
		require.NoError(t, structconf.Template(&sb, &Config{}))
		path := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(path, []byte(sb.String()), 0o644))
		defaults, err := Defaults()
		require.NoError(t, err)

		var loaded Config
		err = structconf.LoadFile(path, &loaded)

		require.NoError(t, err)
		assert.Equal(t, defaults, loaded)
	})

	// The .env.* templates in the repository root must list every setting with its environment variable,
	// so a new field cannot be forgotten there.
	t.Run("lists every setting in the env files", func(t *testing.T) {
		t.Parallel()

		files, err := filepath.Glob(filepath.Join("..", "..", ".env.*"))
		require.NoError(t, err)
		require.NotEmpty(t, files, "the repository root has .env.* templates")

		for _, file := range files {
			t.Run(filepath.Base(file), func(t *testing.T) {
				t.Parallel()

				defined := envFileKeys(t, file)

				for _, l := range structconf.Leaves(&Config{}) {
					assert.Contains(t, defined, l.Env, "%s is missing in %s", l.Env, filepath.Base(file))
				}
			})
		}
	})
}

// envFileKeys returns the variable names assigned in an env file, ignoring comments and blank lines.
func envFileKeys(t *testing.T, path string) map[string]bool {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	keys := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if name, _, ok := strings.Cut(line, "="); ok {
			keys[strings.TrimSpace(name)] = true
		}
	}
	require.NoError(t, scanner.Err())
	return keys
}

func TestLoad(t *testing.T) {
	t.Run("returns the defaults without sources", func(t *testing.T) {
		cfg := loadWithoutSources(t)

		assert.Equal(t, "Swim-Timer", cfg.App.Name)
		assert.Equal(t, EnvProduction, cfg.App.Env)
		assert.Equal(t, zerolog.WarnLevel, cfg.Log.Level)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Empty(t, cfg.Server.Host, "all interfaces")
		assert.Len(t, cfg.Server.TrustedProxies, 2, "only the loopback ranges")
		assert.Empty(t, cfg.Server.CORSAllowedOrigins, "CORS is off")
		assert.Equal(t, media.DriverLocal, cfg.Storage.Driver)
		assert.Empty(t, cfg.Storage.Path, "the data directory")
		assert.Equal(t, 9600, cfg.Timing.BaudRate)
		assert.Equal(t, 50, cfg.Timing.PoolLength)
		assert.Zero(t, cfg.Timing.SimulateEvent)
		assert.Empty(t, cfg.Timing.ReplayPath)
		assert.True(t, cfg.Timing.Recording)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "config.yaml"))
		t.Setenv("SERVER_PORT", "70000")

		_, err := Load(nil)

		assert.ErrorContains(t, err, "server.port")
	})
}

func TestDefaults(t *testing.T) {
	t.Run("sets the app name", func(t *testing.T) {
		t.Parallel()

		cfg, err := Defaults()

		require.NoError(t, err)
		assert.Equal(t, AppTitle, cfg.App.Name)
	})

	t.Run("matches the configuration loaded without sources", func(t *testing.T) {
		loaded := loadWithoutSources(t)

		cfg := testConfig(t)

		assert.Equal(t, loaded, cfg)
	})
}

// loadWithoutSources loads the configuration with neither flags nor a config file. CONFIG_FILE points to a file
// that does not exist, so Load neither reads a config file of the machine running the tests
// nor fails for the missing file.
func loadWithoutSources(t *testing.T) Config {
	t.Helper()

	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "config.yaml"))
	cfg, err := Load(nil)
	require.NoError(t, err)
	return cfg
}

// testConfig returns the validated configuration built from the struct-tag defaults alone.
func testConfig(t *testing.T) Config {
	t.Helper()

	cfg, err := Defaults()
	require.NoError(t, err)
	require.NoError(t, Validate(&cfg))
	return cfg
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("reports every invalid setting at once", func(t *testing.T) {
		t.Parallel()

		cfg, err := Defaults()
		require.NoError(t, err)
		cfg.Server.Port = 0
		cfg.Log.Channel = "nowhere"
		cfg.Theme.Primary = "blue"

		err = Validate(&cfg)

		require.Error(t, err)
		assert.ErrorContains(t, err, "server.port")
		assert.ErrorContains(t, err, "log.channel")
		assert.ErrorContains(t, err, "theme.primary")
	})
}

func TestResolveDataPath(t *testing.T) {
	t.Parallel()

	t.Run("keeps an explicit path", func(t *testing.T) {
		t.Parallel()

		path := apitest.Faker(t).Word()

		got, err := ResolveDataPath(path, "logs")

		require.NoError(t, err)
		assert.Equal(t, path, got)
	})
}

func TestDataPaths(t *testing.T) {
	t.Run("returns only an explicit path", func(t *testing.T) {
		t.Parallel()

		paths := DataPaths("custom", "logs")

		assert.Equal(t, []string{"custom"}, paths, "an explicit path is the only candidate")
	})

	t.Run("returns the candidates of the data directory without creating them", func(t *testing.T) {
		// The data directory lies below the user directories, which only the environment selects
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("LocalAppData", home)

		paths := DataPaths("", "logs")

		dirs := appdirs.DataDirs(AppName)
		require.NotEmpty(t, dirs)
		require.Len(t, paths, len(dirs))
		for i, dir := range dirs {
			assert.Equal(t, filepath.Join(dir, "logs"), paths[i])
		}
		entries, err := os.ReadDir(home)
		require.NoError(t, err)
		assert.Empty(t, entries, "no candidate is created")
	})
}

func TestCleanPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "keeps an empty path empty",
			path: "",
			want: "",
		},
		{
			name: "keeps a blank path empty",
			path: "  ",
			want: "",
		},
		{
			name: "trims and cleans the path",
			path: " logs/../data/ ",
			want: "data",
		},
		{
			name: "keeps a clean absolute path",
			path: "/var/lib/app",
			want: "/var/lib/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := cleanPath(tt.path)

			assert.Equal(t, tt.want, got)
		})
	}
}
