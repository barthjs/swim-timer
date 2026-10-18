package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/cmdline"
)

func TestNewConfigInitCommand(t *testing.T) {
	t.Parallel()

	t.Run("prints the template", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "init")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "server:\n")
		assert.Contains(t, stdout, "# Listen port")
	})

	t.Run("stores the defaults", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "nested", "config.yaml")

		code, _, _ := run(t, "config", "init", path)

		require.Equal(t, 0, code)
		code, stdout, _ := run(t, "config", "validate", path)
		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "is valid")
	})

	t.Run("replaces the file with force", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "config.yaml")
		code, _, _ := run(t, "config", "init", path)
		require.Equal(t, 0, code)

		code, _, _ = run(t, "config", "init", path, "--force")

		assert.Equal(t, 0, code)
	})

	t.Run("rejects an existing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "config.yaml")
		code, _, _ := run(t, "config", "init", path)
		require.Equal(t, 0, code)

		code, _, stderr := run(t, "config", "init", path)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, "--force")
	})

	t.Run("rejects a path below a file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(writeConfigFile(t, ""), "config.yaml")

		code, _, stderr := run(t, "config", "init", path)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, path)
	})

	t.Run("rejects a directory it cannot write to", func(t *testing.T) {
		t.Parallel()

		if os.Geteuid() == 0 {
			t.Skip("root ignores the permissions of the directory")
		}
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		path := filepath.Join(dir, "nested", "config.yaml")

		code, _, stderr := run(t, "config", "init", path)

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, filepath.Join(dir, "nested"))
		assert.NoFileExists(t, path)
	})

	t.Run("rejects a directory as the file with force", func(t *testing.T) {
		t.Parallel()

		path := t.TempDir()

		code, _, stderr := run(t, "config", "init", path, "--force")

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, path)
		assert.DirExists(t, path)
	})
}

func TestNewConfigPathCommand(t *testing.T) {
	t.Parallel()

	t.Run("returns the config file and the search path", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "path", "-o", "json")

		require.Equal(t, 0, code)
		var paths configPaths
		require.NoError(t, json.Unmarshal([]byte(stdout), &paths))
		assert.Equal(t, "config.yaml", filepath.Base(paths.File), "the --config of the test run")
		assert.Contains(t, paths.Search, "config.yaml")
	})

	t.Run("returns the config file and the search path as text", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "path")

		require.Equal(t, 0, code)
		assert.Regexp(t, `Config file:\s+\S+config\.yaml\n`, stdout)
		assert.Contains(t, stdout, "Search order:\n  config.yaml\n")
	})

	t.Run("reports an output that cannot be written", func(t *testing.T) {
		t.Parallel()

		root := newRoot(testBuild)
		root.SetOut(failingOutput{})
		var stderr bytes.Buffer
		root.SetErr(&stderr)

		code := cmdline.Execute(root, []string{"config", "path", "--config", filepath.Join(t.TempDir(), "config.yaml")})

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), errOutputFailed.Error())
	})
}

// errOutputFailed is the error of every write to failingOutput.
var errOutputFailed = errors.New("output failed")

// failingOutput is a command output that fails every write.
type failingOutput struct{}

// Write fails without writing.
func (failingOutput) Write([]byte) (int, error) {
	return 0, errOutputFailed
}

func TestNewConfigShowCommand(t *testing.T) {
	t.Parallel()

	t.Run("returns the configuration as yaml", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "show")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "server:\n")
		assert.Contains(t, stdout, "  port: 8080\n")
	})

	t.Run("returns the configuration as a table", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "show", "-o", "table")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "server.port")
		assert.Contains(t, stdout, "8080")
	})

	t.Run("returns the origin of every setting", func(t *testing.T) {
		t.Parallel()

		port := strconv.Itoa(apitest.Faker(t).IntRange(1024, 65535))

		code, stdout, _ := run(t, "config", "show", "--origin", "-o", "json", "--server-port", port)

		require.Equal(t, 0, code)
		var settings []setting
		require.NoError(t, json.Unmarshal([]byte(stdout), &settings))
		sources := map[string]string{}
		for _, s := range settings {
			sources[s.Key] = s.Source
		}
		assert.Equal(t, "flag --server-port", sources["server.port"])
		assert.Equal(t, "default", sources["timing.baud_rate"])
	})

	t.Run("masks the secrets", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		secret := f.LetterN(32)
		port := f.IntRange(1024, 65535)

		code, stdout, _ := run(t, "config", "show", "-o", "json", "--app-secret", secret, "--server-port", strconv.Itoa(port))

		require.Equal(t, 0, code)
		var shown struct {
			App struct {
				Secret string `json:"secret"`
			} `json:"app"`
			Server struct {
				Port int `json:"port"`
			} `json:"server"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &shown))
		assert.Equal(t, "[redacted]", shown.App.Secret)
		assert.Equal(t, port, shown.Server.Port)
		assert.NotContains(t, stdout, secret)
	})
}

func TestNewConfigValidateCommand(t *testing.T) {
	t.Parallel()

	t.Run("accepts the configuration", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "config", "validate")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "The configuration is valid")
	})

	t.Run("names the config file", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, "server:\n  port: 9001\n")
		root := newRoot(testBuild)
		var stdout bytes.Buffer
		root.SetOut(&stdout)

		code := cmdline.Execute(root, []string{"config", "validate", "--config", path})

		require.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "config file "+path)
	})

	t.Run("accepts a file", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, "server:\n  port: 9001\n")

		code, stdout, _ := run(t, "config", "validate", path)

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, path+" is valid")
	})

	t.Run("keeps the flags out of a file check", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, "server:\n  port: 9001\n")

		// The flags apply to the configuration of the application, not to a file checked on its own
		code, _, _ := run(t, "config", "validate", path, "--server-port", "0")

		assert.Equal(t, 0, code)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		code, _, stderr := run(t, "config", "validate", "--server-port", "0", "--log-channel", "syslog")

		require.Equal(t, 1, code)
		assert.Contains(t, stderr, "the configuration is invalid")
		assert.Contains(t, stderr, "server.port")
		assert.Contains(t, stderr, "log.channel")
	})

	t.Run("rejects an invalid file", func(t *testing.T) {
		t.Parallel()

		path := writeConfigFile(t, "server:\n  port: 0\n")

		code, _, stderr := run(t, "config", "validate", path)

		require.Equal(t, 1, code)
		assert.Contains(t, stderr, "server.port")
	})

	t.Run("rejects a missing file", func(t *testing.T) {
		t.Parallel()

		code, _, stderr := run(t, "config", "validate", filepath.Join(t.TempDir(), "missing.yaml"))

		assert.Equal(t, 1, code)
		assert.NotEmpty(t, stderr)
	})
}

// writeConfigFile writes content to a config file in a temporary directory and returns its path.
func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
