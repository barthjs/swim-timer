package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

// testBuild is the build information of the command tree in the tests.
var testBuild = cmdline.Build{
	Title:   "Swim-Timer",
	Version: "v1.0.0",
	Commit:  "abc1234",
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
