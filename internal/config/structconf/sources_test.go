package structconf

import (
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSource_String(t *testing.T) {
	t.Parallel()

	t.Run("names the kind and the origin", func(t *testing.T) {
		t.Parallel()

		got := Source{
			Kind: SourceEnv,
			Name: "LOG_LEVEL",
		}.String()

		assert.Equal(t, "env LOG_LEVEL", got)
	})

	t.Run("names only the kind without an origin", func(t *testing.T) {
		t.Parallel()

		got := Source{
			Kind: SourceDefault,
		}.String()

		assert.Equal(t, "default", got)
	})
}

func TestSources(t *testing.T) {
	t.Run("returns the source of every setting", func(t *testing.T) {
		path := writeFile(t, "config.yaml", "demo:\n  name: from file\n  debug: false\n  server:\n    port: 9000\n")
		secret := writeFile(t, "ratio", "0.7\n")
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("DEMO_SERVER_PORT", "9100")
		t.Setenv("DEMO_RATIO_FILE", secret)
		t.Setenv("DEMO_TAGS", "")
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		require.NoError(t, RegisterFlags(flags, &tagConfig{}))
		require.NoError(t, flags.Parse([]string{"--demo-debug=true"}))

		sources, err := Sources(testAppName, flags, &tagConfig{})

		require.NoError(t, err)
		want := map[string]Source{
			"demo.debug": {
				Kind: SourceFlag,
				Name: "--demo-debug",
			},
			"demo.server.port": {
				Kind: SourceEnv,
				Name: "DEMO_SERVER_PORT",
			},
			"demo.ratio": {
				Kind: SourceEnv,
				Name: "DEMO_RATIO_FILE",
			},
			"demo.name": {
				Kind: SourceFile,
				Name: path,
			},
			"demo.tags": {
				Kind: SourceDefault,
			},
			"demo.server.log_level": {
				Kind: SourceDefault,
			},
		}
		for key, source := range want {
			assert.Equal(t, source, sources[key], key)
		}
		assert.Len(t, sources, len(Leaves(&tagConfig{})), "every setting has a source")
	})

	t.Run("returns the default without a file", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))

		sources, err := Sources(testAppName, nil, &tagConfig{})

		require.NoError(t, err)
		assert.Equal(t, Source{
			Kind: SourceDefault,
		}, sources["demo.name"])
	})
}

func TestPathSources(t *testing.T) {
	t.Run("returns the sources keyed by the JSON path", func(t *testing.T) {
		path := writeFile(t, "config.yaml", "demo:\n  name: from file\n")
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("DEMO_SERVER_LOG_LEVEL", "error")

		sources, err := PathSources(testAppName, nil, &tagConfig{})

		require.NoError(t, err)
		assert.Equal(t, Source{
			Kind: SourceEnv,
			Name: "DEMO_SERVER_LOG_LEVEL",
		}, sources["demo.server.logLevel"], "a source is keyed by the JSON path")
		assert.Equal(t, Source{
			Kind: SourceFile,
			Name: path,
		}, sources["demo.name"])
		assert.Equal(t, Source{
			Kind: SourceDefault,
		}, sources["demo.server.port"])
		assert.NotContains(t, sources, "demo.server.no_value", "a setting without a JSON name is left out")
		assert.Len(t, sources, len(Leaves(&tagConfig{}))-1)
	})
}
