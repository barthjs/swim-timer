package structconf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAppName names the config directories the tests search, which do not exist on the test machine.
const testAppName = "structconf-test"

func TestLoad(t *testing.T) {
	t.Run("applies the default, the file, the environment, and the flags in order", func(t *testing.T) {
		path := writeFile(t, "config.yaml", "demo:\n  name: from file\n  server:\n    port: 9000\n    log_level: info\n")
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("DEMO_SERVER_PORT", "9100")
		t.Setenv("DEMO_SERVER_LOG_LEVEL", "error")
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("config", "", "")
		require.NoError(t, RegisterFlags(flags, &tagConfig{}))
		require.NoError(t, flags.Parse([]string{"--demo-server-log-level=debug"}))

		cfg, err := loadTagConfig(t, flags)

		require.NoError(t, err)
		assert.True(t, cfg.Demo.Debug, "the default applies without another value")
		assert.Equal(t, "from file", cfg.Demo.Name, "the file overrides the default")
		assert.Equal(t, 9100, cfg.Demo.Server.Port, "the environment overrides the file")
		assert.Equal(t, "debug", cfg.Demo.Server.LogLevel, "a flag overrides the environment")
	})

	t.Run("prefers the file from the flag over the environment", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", writeFile(t, "env.yaml", "demo:\n  name: from env file\n"))
		flagPath := writeFile(t, "flag.yaml", "demo:\n  name: from flag file\n")
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("config", "", "")
		require.NoError(t, RegisterFlags(flags, &tagConfig{}))
		require.NoError(t, flags.Parse([]string{"--config=" + flagPath}))

		cfg, err := loadTagConfig(t, flags)

		require.NoError(t, err)
		assert.Equal(t, "from flag file", cfg.Demo.Name)
	})

	t.Run("reads every file format", func(t *testing.T) {
		tests := map[string]string{
			"config.yaml": "demo:\n  name: yaml\n",
			"config.json": `{"demo": {"name": "json"}}`,
			"config.toml": "[demo]\nname = \"toml\"\n",
		}

		for name, content := range tests {
			t.Run(name, func(t *testing.T) {
				t.Setenv("CONFIG_FILE", writeFile(t, name, content))

				cfg, err := loadTagConfig(t, nil)

				require.NoError(t, err)
				assert.Equal(t, filepath.Ext(name)[1:], cfg.Demo.Name)
			})
		}
	})

	t.Run("keeps the defaults for a missing explicit file", func(t *testing.T) {
		// Docker images set CONFIG_FILE to a path that may not exist yet on a fresh volume
		t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))

		cfg, err := loadTagConfig(t, nil)

		require.NoError(t, err)
		assert.Equal(t, "demo", cfg.Demo.Name)
	})

	t.Run("searches the working directory", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", "")
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("demo:\n  name: from cwd\n"), 0o644))
		t.Chdir(dir)

		cfg, err := loadTagConfig(t, nil)

		require.NoError(t, err)
		assert.Equal(t, "from cwd", cfg.Demo.Name)
	})

	t.Run("rejects an invalid file", func(t *testing.T) {
		path := writeFile(t, "config.yaml", "demo: [not: valid")
		t.Setenv("CONFIG_FILE", path)

		_, err := loadTagConfig(t, nil)

		assert.ErrorContains(t, err, path)
	})
}

// loadTagConfig loads a tagConfig through all sources.
func loadTagConfig(t *testing.T, flags *pflag.FlagSet) (tagConfig, error) {
	t.Helper()

	var cfg tagConfig
	err := Load(testAppName, flags, &cfg)
	return cfg, err
}

func TestDefaults(t *testing.T) {
	t.Run("returns the defaults without the environment or the file", func(t *testing.T) {
		t.Setenv("DEMO_NAME", "from-env")
		t.Setenv("CONFIG_FILE", writeFile(t, "config.yaml", "demo:\n  name: from file\n"))
		var cfg tagConfig

		require.NoError(t, Defaults(&cfg))

		assert.Equal(t, "demo", cfg.Demo.Name)
		assert.Equal(t, 8080, cfg.Demo.Server.Port)
		assert.Equal(t, 90*time.Second, cfg.Demo.Timeout)
		assert.Empty(t, cfg.Demo.Server.NoValue)
	})
}

func TestLoadFile(t *testing.T) {
	t.Run("reads the file without the environment", func(t *testing.T) {
		path := writeFile(t, "config.yaml", "demo:\n  name: from file\n")
		t.Setenv("DEMO_NAME", "from env")
		t.Setenv("DEMO_SERVER_PORT", "9100")
		var cfg tagConfig

		require.NoError(t, LoadFile(path, &cfg))

		assert.Equal(t, "from file", cfg.Demo.Name)
		assert.Equal(t, 8080, cfg.Demo.Server.Port, "the default, not the environment")
	})

	tests := []struct {
		name string
		path string
	}{
		{
			name: "rejects a missing file",
			path: filepath.Join(t.TempDir(), "config.yaml"),
		},
		{
			name: "rejects an invalid file",
			path: writeFile(t, "config.yaml", "demo: [unclosed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cfg tagConfig

			err := LoadFile(tt.path, &cfg)

			assert.ErrorContains(t, err, tt.path)
		})
	}
}

// writeFile writes content to a file in a fresh temporary directory and returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
