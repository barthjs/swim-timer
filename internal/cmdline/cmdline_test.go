package cmdline_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

var testBuild = cmdline.Build{
	Title:   "Demo",
	Version: "v1.2.3",
	Commit:  "abc1234",
}

func TestNewRoot(t *testing.T) {
	t.Parallel()

	t.Run("is named after the binary", func(t *testing.T) {
		t.Parallel()

		root, _, _ := newTestRoot(t)

		assert.Equal(t, strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"), root.Name())
	})

	t.Run("prints the version for the version flag", func(t *testing.T) {
		t.Parallel()

		root, stdout, _ := newTestRoot(t)
		root.Run = func(*cobra.Command, []string) {}

		code := cmdline.Execute(root, []string{"--version"})

		assert.Equal(t, 0, code)
		assert.Equal(t, cmdline.VersionText(testBuild), stdout.String())
	})
}

func TestPersistentSetup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantSetup bool
	}{
		{
			name:      "runs the setup for a regular command",
			args:      []string{"run"},
			wantSetup: true,
		},
		{
			name:      "runs the setup for a nested command",
			args:      []string{"group", "child"},
			wantSetup: true,
		},
		{
			name:      "skips the setup for a skipped command",
			args:      []string{"plain"},
			wantSetup: false,
		},
		{
			name:      "skips the setup for the child of a skipped command",
			args:      []string{"skipped", "child"},
			wantSetup: false,
		},
		{
			name:      "skips the setup for help",
			args:      []string{"help"},
			wantSetup: false,
		},
		{
			name:      "skips the setup for completion",
			args:      []string{"completion", "bash"},
			wantSetup: false,
		},
		{
			name:      "skips the setup for version",
			args:      []string{"version"},
			wantSetup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _, _ := newTestRoot(t)
			setupRan := false
			cmdline.PersistentSetup(root, func(*cobra.Command) error {
				setupRan = true
				return nil
			})

			noop := func(*cobra.Command, []string) {}
			group := &cobra.Command{
				Use: "group",
			}
			group.AddCommand(&cobra.Command{
				Use: "child",
				Run: noop,
			})
			skipped := cmdline.SkipSetup(&cobra.Command{
				Use: "skipped",
			})
			skipped.AddCommand(&cobra.Command{
				Use: "child",
				Run: noop,
			})
			root.AddCommand(
				&cobra.Command{
					Use: "run",
					Run: noop,
				},
				group,
				cmdline.SkipSetup(&cobra.Command{
					Use: "plain",
					Run: noop,
				}),
				skipped,
				cmdline.VersionCommand(testBuild),
			)

			code := cmdline.Execute(root, tt.args)

			assert.Equal(t, 0, code)
			assert.Equal(t, tt.wantSetup, setupRan)
		})
	}

	t.Run("runs the setup before the hook of a command", func(t *testing.T) {
		t.Parallel()

		root, _, _ := newTestRoot(t)
		var calls []string
		cmdline.PersistentSetup(root, func(*cobra.Command) error {
			calls = append(calls, "setup")
			return nil
		})
		root.AddCommand(&cobra.Command{
			Use: "run",
			PersistentPreRunE: func(*cobra.Command, []string) error {
				calls = append(calls, "hook")
				return nil
			},
			Run: func(*cobra.Command, []string) { calls = append(calls, "run") },
		})

		code := cmdline.Execute(root, []string{"run"})

		require.Equal(t, 0, code)
		assert.Equal(t, []string{"setup", "hook", "run"}, calls)
	})

	t.Run("stops the command when the setup fails", func(t *testing.T) {
		t.Parallel()

		root, _, stderr := newTestRoot(t)
		cmdline.PersistentSetup(root, func(*cobra.Command) error { return errors.New("no config") })
		ran := false
		root.AddCommand(&cobra.Command{
			Use: "run",
			Run: func(*cobra.Command, []string) { ran = true },
		})

		code := cmdline.Execute(root, []string{"run"})

		assert.Equal(t, 1, code)
		assert.False(t, ran)
		assert.Equal(t, "Error: no config\n", stderr.String())
	})
}

func TestExit(t *testing.T) {
	t.Parallel()

	t.Run("returns an error that wraps the cause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("cause")

		err := cmdline.Exit(4, cause)

		assert.EqualError(t, err, "cause")
		assert.ErrorIs(t, err, cause)
	})

	t.Run("names the exit status without a cause", func(t *testing.T) {
		t.Parallel()

		err := cmdline.Exit(4, nil)

		assert.EqualError(t, err, "exit status 4")
	})
}

func TestDefer(t *testing.T) {
	t.Parallel()

	t.Run("runs in reverse order after a failure", func(t *testing.T) {
		t.Parallel()

		root, _, _ := newTestRoot(t)
		var order []string
		root.AddCommand(&cobra.Command{
			Use: "run",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cmdline.Defer(cmd, func() { order = append(order, "first") })
				cmdline.Defer(cmd, func() { order = append(order, "second") })
				return errors.New("fail")
			},
		})

		code := cmdline.Execute(root, []string{"run"})

		assert.Equal(t, 1, code)
		assert.Equal(t, []string{"second", "first"}, order)
	})

	t.Run("runs when the context ends outside execute", func(t *testing.T) {
		t.Parallel()

		root, _, _ := newTestRoot(t)
		ran := make(chan struct{})
		root.AddCommand(&cobra.Command{
			Use: "run",
			Run: func(cmd *cobra.Command, _ []string) {
				cmdline.Defer(cmd, func() { close(ran) })
			},
		})
		root.SetArgs([]string{"run"})
		ctx, cancel := contextWithCancel(t)

		err := root.ExecuteContext(ctx)

		require.NoError(t, err)
		cancel()
		<-ran
	})
}

// contextWithCancel returns a cancelable context that is canceled when the test ends.
func contextWithCancel(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	return ctx, cancel
}

func TestExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		run        func(*cobra.Command, []string) error
		wantCode   int
		wantStderr string
	}{
		{
			name:     "returns zero on success",
			run:      func(*cobra.Command, []string) error { return nil },
			wantCode: 0,
		},
		{
			name:       "returns one and prints an error",
			run:        func(*cobra.Command, []string) error { return errors.New("broken") },
			wantCode:   1,
			wantStderr: "Error: broken\n",
		},
		{
			name:       "returns the exit code and prints its message",
			run:        func(*cobra.Command, []string) error { return cmdline.Exit(3, errors.New("partial")) },
			wantCode:   3,
			wantStderr: "Error: partial\n",
		},
		{
			name:     "returns a silent exit code",
			run:      func(*cobra.Command, []string) error { return cmdline.Exit(2, nil) },
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root, _, stderr := newTestRoot(t)
			root.AddCommand(&cobra.Command{
				Use:  "run",
				RunE: tt.run,
			})

			code := cmdline.Execute(root, []string{"run"})

			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}

	t.Run("rejects an unknown command", func(t *testing.T) {
		t.Parallel()

		root, _, stderr := newTestRoot(t)
		root.AddCommand(&cobra.Command{
			Use: "run",
			Run: func(*cobra.Command, []string) {},
		})

		code := cmdline.Execute(root, []string{"nope"})

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `unknown command "nope"`)
	})
}

func TestManCommand(t *testing.T) {
	t.Parallel()

	t.Run("writes the man pages", func(t *testing.T) {
		t.Parallel()

		root, _, _ := newTestRoot(t)
		gen := &cobra.Command{
			Use: "gen",
		}
		gen.AddCommand(cmdline.ManCommand(root))
		root.AddCommand(gen)
		dir := filepath.Join(t.TempDir(), "man")

		code := cmdline.Execute(root, []string{"gen", "man", dir})

		require.Equal(t, 0, code)
		pages, err := filepath.Glob(filepath.Join(dir, "*.1"))
		require.NoError(t, err)
		assert.Contains(t, pages, filepath.Join(dir, root.Name()+"-gen-man.1"))
	})
}

// newTestRoot creates a root command whose output is captured.
func newTestRoot(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	root := cmdline.NewRoot(testBuild, "A demo")
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	return root, &stdout, &stderr
}
