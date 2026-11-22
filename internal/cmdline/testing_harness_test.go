package cmdline_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

var testBuild = cmdline.Build{
	Title:   "Demo",
	Version: "v1.2.3",
	Commit:  "abc1234",
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
