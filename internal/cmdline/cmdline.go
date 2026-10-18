// Package cmdline builds command-line interfaces on cobra.
package cmdline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// skipSetupAnnotation marks a command that runs without the application setup.
const skipSetupAnnotation = "cmdline.skipSetup"

// init makes cobra run every PersistentPreRunE from the root down to the command, not only the closest one.
// It is a global switch of cobra, so it is set once instead of per command tree.
func init() {
	cobra.EnableTraverseRunHooks = true
}

// NewRoot creates the root command. It is named after the running binary, so help, usage,
// and shell completion stay correct when the binary is renamed. Errors are printed by Execute instead of cobra,
// and a failing command does not print its usage, since most failures are not usage errors.
// The root command also answers --version.
func NewRoot(build Build, short string) *cobra.Command {
	root := &cobra.Command{
		Use:           BinaryName(),
		Short:         short,
		Version:       build.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetVersionTemplate(VersionText(build))
	return root
}

// BinaryName returns the file name of the running binary without directory and Windows extension.
func BinaryName() string {
	return strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
}

// SkipSetup marks a command, and with it its subcommands, to run without the setup registered with PersistentSetup.
func SkipSetup(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[skipSetupAnnotation] = "true"
	return cmd
}

// PersistentSetup runs setup once before every command below root that needs it.
func PersistentSetup(root *cobra.Command, setup func(cmd *cobra.Command) error) {
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if !needsSetup(cmd) {
			return nil
		}
		return setup(cmd)
	}
}

// needsSetup reports whether the setup runs for cmd. Help and shell completion never need it.
func needsSetup(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[skipSetupAnnotation] == "true" {
			return false
		}
		switch c.Name() {
		case "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return false
		}
	}
	return true
}

// ExitError ends a command with a specific exit code. With a nil Err,
// nothing is printed, for commands that already reported what went wrong.
type ExitError struct {
	Code int
	Err  error
}

// Error returns the message of the wrapped error, or an empty string without one.
func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return e.Err.Error()
}

// Unwrap returns the wrapped error.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// Exit returns an ExitError with the code and an optional error to print.
func Exit(code int, err error) error {
	return &ExitError{
		Code: code,
		Err:  err,
	}
}

// cleanupKey is the context key of the cleanup functions of a run.
type cleanupKey struct{}

// cleanups collects the functions that run after a command.
type cleanups struct {
	fns []func()
}

// Defer registers fn to run after the command finished, also when it failed.
// Functions run in reverse order, like defer. Outside Execute, for example in a test
// that calls root.Execute directly, fn runs when the command context ends, if it ever does.
func Defer(cmd *cobra.Command, fn func()) {
	if list, ok := cmd.Context().Value(cleanupKey{}).(*cleanups); ok {
		list.fns = append(list.fns, fn)
		return
	}
	context.AfterFunc(cmd.Context(), fn)
}

// Execute runs the root command with the given arguments, os.Args[1:] when nil. The context ends on SIGINT or SIGTERM,
// so long-running commands can stop gracefully. It prints a failure to stderr and returns the exit code:
// 0 on success, the code of an ExitError, and 1 for any other error.
func Execute(root *cobra.Command, args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	list := &cleanups{}
	ctx = context.WithValue(ctx, cleanupKey{}, list)
	defer func() {
		for _, v := range slices.Backward(list.fns) {
			v()
		}
	}()

	if args != nil {
		root.SetArgs(args)
	}

	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}

	if exitErr, ok := errors.AsType[*ExitError](err); ok {
		if exitErr.Err != nil {
			_, _ = fmt.Fprintln(root.ErrOrStderr(), "Error:", exitErr.Err)
		}
		return exitErr.Code
	}

	_, _ = fmt.Fprintln(root.ErrOrStderr(), "Error:", err)
	return 1
}

// ManCommand generates a man page for every command of root into a directory, for packaging.
func ManCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "man [directory]",
		Short:   "Generate man pages",
		Example: "  " + root.Name() + " gen man ./man",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{
				Title:   strings.ToUpper(root.Name()),
				Section: "1",
			}
			return doc.GenManTree(root, header, dir)
		},
	}
	return SkipSetup(cmd)
}
