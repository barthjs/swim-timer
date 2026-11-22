package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

func TestNewRoot(t *testing.T) {
	t.Parallel()

	t.Run("returns the version", func(t *testing.T) {
		t.Parallel()

		code, stdout, _ := run(t, "version")

		require.Equal(t, 0, code)
		assert.Contains(t, stdout, "Swim-Timer v1.0.0")
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
