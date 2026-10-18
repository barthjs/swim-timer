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

func TestFindFile(t *testing.T) {
	t.Run("returns the file from the environment without a flag", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", "/from/env.yaml")

		got := FindFile(testAppName, nil)

		assert.Equal(t, "/from/env.yaml", got)
	})

	t.Run("prefers the flag over the environment", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", "/from/env.yaml")
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String(FileFlag, "", "")
		require.NoError(t, flags.Parse([]string{"--config=/from/flag.yaml"}))

		got := FindFile(testAppName, flags)

		assert.Equal(t, "/from/flag.yaml", got)
	})

	t.Run("searches the working directory with yaml first", func(t *testing.T) {
		t.Setenv("CONFIG_FILE", "")
		dir := t.TempDir()
		t.Chdir(dir)

		assert.Empty(t, FindFile(testAppName, nil), "no file exists yet")

		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644))
		assert.Equal(t, filepath.Join(".", "config.json"), FindFile(testAppName, nil))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(""), 0o644))
		assert.Equal(t, filepath.Join(".", "config.yaml"), FindFile(testAppName, nil))
	})
}

func TestSearchPaths(t *testing.T) {
	t.Parallel()

	t.Run("starts in the working directory", func(t *testing.T) {
		t.Parallel()

		paths := SearchPaths("example-app")

		require.NotEmpty(t, paths)
		assert.Equal(t, []string{"config.yaml", "config.yml", "config.json", "config.toml"}, paths[:4])
	})
}

func TestWrite(t *testing.T) {
	t.Run("stores the configuration so it loads back", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "config.yaml")
		t.Setenv("CONFIG_FILE", path)
		want, err := loadTagConfig(t, nil)
		require.NoError(t, err)
		want.Demo.Name = "written"
		want.Demo.Timeout = 2 * time.Minute
		want.Demo.Tags = []string{"x", "y"}
		want.Demo.Server.Port = 9443

		got, err := Write(testAppName, path, &want)

		require.NoError(t, err)
		assert.Equal(t, path, got)
		loaded, err := loadTagConfig(t, nil)
		require.NoError(t, err)
		assert.Equal(t, want, loaded)
		matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), "config-*"))
		require.NoError(t, err)
		assert.Empty(t, matches, "no temporary file is left behind")
	})
}
