package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

// testBuild is the build information of the command tree in the tests.
var testBuild = cmdline.Build{
	Title:   "Swim-Timer",
	Version: "v1.0.0",
	Commit:  "abc1234",
}

func TestNewRoot(t *testing.T) {
	t.Parallel()

	t.Run("returns the version", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "version")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "Swim-Timer\tv1.0.0")
	})

	t.Run("groups the commands in the help", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "--help")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "Commands:\n")
		assert.Contains(t, stdout, "Additional Commands:\n")
		assert.Contains(t, stdout, "healthcheck")
	})

	t.Run("rejects an unknown command", func(t *testing.T) {
		t.Parallel()

		code, _, stderr := run(t, "unknown")

		require.Equal(t, 1, code)
		assert.Contains(t, stderr, `unknown command "unknown"`)
	})
}

// run executes the command line against a fresh command tree and returns the exit code and output.
// A config file that does not exist keeps the configuration of the machine running the tests out.
func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	root := newRoot(testBuild)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	args = append(args, "--config", filepath.Join(t.TempDir(), "config.yaml"))
	code := cmdline.Execute(root, args)
	return code, stdout.String(), stderr.String()
}

func TestRegisterValueCompletions(t *testing.T) {
	t.Parallel()

	t.Run("completes the values of a setting", func(t *testing.T) {
		t.Parallel()

		root := newRoot(testBuild)
		var stdout bytes.Buffer
		root.SetOut(&stdout)

		code := cmdline.Execute(root, []string{cobra.ShellCompRequestCmd, "--log-level", ""})

		require.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "debug\n")
		assert.Contains(t, stdout.String(), "error\n")
	})
}
