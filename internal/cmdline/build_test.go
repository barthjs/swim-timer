package cmdline_test

import (
	"encoding/json/v2"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/cmdline"
)

func TestResolveCommit(t *testing.T) {
	t.Parallel()

	t.Run("returns nothing without vcs information", func(t *testing.T) {
		t.Parallel()

		commit := cmdline.ResolveCommit()

		assert.Empty(t, commit)
	})
}

func TestResolveArch(t *testing.T) {
	t.Parallel()

	t.Run("returns the architecture of the binary", func(t *testing.T) {
		t.Parallel()

		arch := cmdline.ResolveArch()

		if runtime.GOARCH == "arm" {
			assert.Regexp(t, `^armv\d+$`, arch)
			return
		}
		assert.Equal(t, runtime.GOARCH, arch)
	})
}

func TestVersionText(t *testing.T) {
	t.Parallel()

	t.Run("returns the build information", func(t *testing.T) {
		t.Parallel()

		text := cmdline.VersionText(testBuild)

		assert.Contains(t, text, "Demo\tv1.2.3\n")
		assert.Contains(t, text, "Commit:\t\tabc1234\n")
		assert.Contains(t, text, "Go Version:\t"+runtime.Version()+"\n")
		assert.Contains(t, text, "OS/Arch:\t"+runtime.GOOS+"/"+runtime.GOARCH+"\n")
	})

	t.Run("leaves out a missing commit", func(t *testing.T) {
		t.Parallel()

		build := testBuild
		build.Commit = ""

		// Test binaries carry no VCS information, so the commit line is left out
		text := cmdline.VersionText(build)

		assert.NotContains(t, text, "Commit:")
	})
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	t.Run("prints the version as text", func(t *testing.T) {
		t.Parallel()

		root, stdout, _ := newTestRoot(t)
		root.AddCommand(cmdline.VersionCommand(testBuild))

		code := cmdline.Execute(root, []string{"version"})

		require.Equal(t, 0, code)
		assert.Equal(t, cmdline.VersionText(testBuild), stdout.String())
	})

	t.Run("prints the version as json", func(t *testing.T) {
		t.Parallel()

		root, stdout, _ := newTestRoot(t)
		root.AddCommand(cmdline.VersionCommand(testBuild))

		code := cmdline.Execute(root, []string{"version", "--output", "json"})

		require.Equal(t, 0, code)
		var info map[string]string
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &info))
		assert.Equal(t, "Demo", info["name"])
		assert.Equal(t, "v1.2.3", info["version"])
		assert.Equal(t, "abc1234", info["commit"])
		assert.Equal(t, cmdline.ResolveArch(), info["arch"])
	})

	t.Run("rejects an unknown format", func(t *testing.T) {
		t.Parallel()

		root, _, stderr := newTestRoot(t)
		root.AddCommand(cmdline.VersionCommand(testBuild))

		code := cmdline.Execute(root, []string{"version", "-o", "xml"})

		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `unknown output format "xml"`)
	})
}
