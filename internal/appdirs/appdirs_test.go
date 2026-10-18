package appdirs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/appdirs"
)

const appName = "example-app"

var roots = appdirs.Roots{
	User:   filepath.Join("home", "config"),
	System: filepath.Join("sys", "config"),
}

// TestConfigRoots does not run in parallel, since its subtests set the environment.
func TestConfigRoots(t *testing.T) {
	t.Run("returns the user and system roots", func(t *testing.T) {
		userRoot := setUserConfigRoot(t)

		got := appdirs.ConfigRoots()

		assert.Equal(t, userRoot, got.User)
		assert.NotEmpty(t, got.System)
	})

	t.Run("leaves the user root empty without a home", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("only Linux derives the user config directory from these variables")
		}
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")

		got := appdirs.ConfigRoots()

		assert.Empty(t, got.User)
	})
}

// TestDataRoots does not run in parallel, since its subtests set the environment.
func TestDataRoots(t *testing.T) {
	t.Run("returns the user and system roots", func(t *testing.T) {
		userRoot := t.TempDir()
		switch runtime.GOOS {
		case "darwin":
			t.Setenv("HOME", userRoot)
			userRoot = filepath.Join(userRoot, "Library", "Application Support")
		case "windows":
			t.Setenv("LocalAppData", userRoot)
		default:
			t.Setenv("XDG_DATA_HOME", userRoot)
		}

		got := appdirs.DataRoots()

		assert.Equal(t, userRoot, got.User)
		assert.NotEmpty(t, got.System)
	})

	tests := []struct {
		name        string
		xdgDataHome string
	}{
		{
			name:        "falls back to the home directory without XDG_DATA_HOME",
			xdgDataHome: "",
		},
		{
			name:        "ignores a relative XDG_DATA_HOME",
			xdgDataHome: "relative/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
				t.Skip("only Unix systems besides macOS follow the XDG Base Directory Specification")
			}
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_DATA_HOME", tt.xdgDataHome)

			got := appdirs.DataRoots()

			assert.Equal(t, filepath.Join(home, ".local", "share"), got.User)
		})
	}

	t.Run("leaves the user root empty without a home", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("only Linux derives the user data directory from these variables")
		}
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "")

		got := appdirs.DataRoots()

		assert.Empty(t, got.User)
	})
}

// TestConfigSearchDirs does not run in parallel, since it sets the environment.
func TestConfigSearchDirs(t *testing.T) {
	setUserConfigRoot(t)

	got := appdirs.ConfigSearchDirs(appName, "x")

	assert.Equal(t, appdirs.ConfigRoots().SearchDirs(appName, "x"), got)
}

// TestConfigWriteDirs does not run in parallel, since it sets the environment.
func TestConfigWriteDirs(t *testing.T) {
	setUserConfigRoot(t)

	got := appdirs.ConfigWriteDirs(appName, "x")

	assert.Equal(t, appdirs.ConfigRoots().WriteDirs(appName, "x"), got)
}

// setUserConfigRoot points the user config root of the current system to a temporary directory and returns the root.
func setUserConfigRoot(t *testing.T) string {
	t.Helper()

	userRoot := t.TempDir()
	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", userRoot)
		return filepath.Join(userRoot, "Library", "Application Support")
	case "windows":
		t.Setenv("AppData", userRoot)
	default:
		t.Setenv("XDG_CONFIG_HOME", userRoot)
	}

	return userRoot
}

// TestLogDirs does not run in parallel, since its subtests set the environment.
func TestLogDirs(t *testing.T) {
	t.Run("returns the system and user directories", func(t *testing.T) {
		home := t.TempDir()
		var want []string
		switch runtime.GOOS {
		case "darwin":
			t.Setenv("HOME", home)
			want = []string{
				filepath.Join("/Library/Logs", appName),
				filepath.Join(home, "Library", "Logs", appName),
			}
		case "windows":
			system := t.TempDir()
			t.Setenv("ProgramData", system)
			t.Setenv("LocalAppData", home)
			want = []string{
				filepath.Join(system, appName, "logs"),
				filepath.Join(home, appName, "logs"),
			}
		default:
			t.Setenv("XDG_STATE_HOME", home)
			want = []string{
				filepath.Join("/var/log", appName),
				filepath.Join(home, appName, "logs"),
			}
		}

		got := appdirs.LogDirs(appName)

		assert.Equal(t, want, got)
	})

	tests := []struct {
		name         string
		xdgStateHome string
	}{
		{
			name:         "falls back to the home directory without XDG_STATE_HOME",
			xdgStateHome: "",
		},
		{
			name:         "ignores a relative XDG_STATE_HOME",
			xdgStateHome: "relative/state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
				t.Skip("only Unix systems besides macOS follow the XDG Base Directory Specification")
			}
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_STATE_HOME", tt.xdgStateHome)

			got := appdirs.LogDirs(appName)

			assert.Equal(t, []string{
				filepath.Join("/var/log", appName),
				filepath.Join(home, ".local", "state", appName, "logs"),
			}, got)
		})
	}

	t.Run("keeps only the system directory without a home", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("only Linux derives the user log directory from these variables")
		}
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", "")

		got := appdirs.LogDirs(appName)

		assert.Equal(t, []string{
			filepath.Join("/var/log", appName),
		}, got)
	})
}

// TestLogDir does not run in parallel, since its subtests set the environment.
func TestLogDir(t *testing.T) {
	t.Run("falls back to the user directory", func(t *testing.T) {
		if runtime.GOOS != "linux" || os.Geteuid() == 0 {
			t.Skip("needs a system-wide log directory the current user cannot write")
		}
		t.Setenv("XDG_STATE_HOME", t.TempDir())

		got, err := appdirs.LogDir(appName)

		require.NoError(t, err)
		assert.Equal(t, appdirs.LogDirs(appName)[1], got)
		assert.DirExists(t, got, "the directory is created")
	})
}

func TestRoots_SearchDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		roots  appdirs.Roots
		subdir string
		want   []string
	}{
		{
			name:  "returns the working, user, then system directory",
			roots: roots,
			want:  []string{".", filepath.Join("home", "config", appName), filepath.Join("sys", "config", appName)},
		},
		{
			name:   "appends the subdirectory",
			roots:  roots,
			subdir: "certs",
			want: []string{
				"certs",
				filepath.Join("home", "config", appName, "certs"),
				filepath.Join("sys", "config", appName, "certs"),
			},
		},
		{
			name: "omits a missing user root",
			roots: appdirs.Roots{
				System: roots.System,
			},
			want: []string{".", filepath.Join("sys", "config", appName)},
		},
		{
			name:  "returns only the working directory without any root",
			roots: appdirs.Roots{},
			want:  []string{"."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.roots.SearchDirs(appName, tt.subdir)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoots_WriteDirs(t *testing.T) {
	t.Parallel()

	got := roots.WriteDirs(appName, "certs")

	assert.Equal(t,
		[]string{
			filepath.Join("sys", "config", appName, "certs"),
			filepath.Join("home", "config", appName, "certs"),
			"certs",
		},
		got,
		"system first, working directory last",
	)
}

func TestRoots_DataDir(t *testing.T) {
	t.Parallel()

	t.Run("prefers the system directory", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User:   t.TempDir(),
			System: t.TempDir(),
		}

		got, err := r.DataDir(appName)

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(r.System, appName), got)
		assert.DirExists(t, got, "the directory is created")
	})

	t.Run("falls back to the user directory", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User:   t.TempDir(),
			System: unwritableRoot(t),
		}

		got, err := r.DataDir(appName)

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(r.User, appName), got)
	})

	t.Run("skips an empty root", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User: t.TempDir(),
		}

		got, err := r.DataDir(appName)

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(r.User, appName), got)
	})

	t.Run("fails without a writable directory", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User:   unwritableRoot(t),
			System: unwritableRoot(t),
		}

		_, err := r.DataDir(appName)

		assert.ErrorContains(t, err, "no writable directory")
	})

	t.Run("never uses the working directory", func(t *testing.T) {
		t.Parallel()

		_, err := appdirs.Roots{}.DataDir(appName)

		assert.Error(t, err)
	})
}

func TestRoots_DataDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns the system and user directories without creating them", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User:   filepath.Join(t.TempDir(), "user"),
			System: filepath.Join(t.TempDir(), "system"),
		}

		got := r.DataDirs(appName)

		assert.Equal(t, []string{filepath.Join(r.System, appName), filepath.Join(r.User, appName)}, got)
		for _, dir := range got {
			assert.NoDirExists(t, dir, "a candidate is not created")
		}
	})

	t.Run("omits an empty root", func(t *testing.T) {
		t.Parallel()

		r := appdirs.Roots{
			User: filepath.Join(t.TempDir(), "user"),
		}

		got := r.DataDirs(appName)

		assert.Equal(t, []string{filepath.Join(r.User, appName)}, got)
	})
}

func TestFirstWritableDir(t *testing.T) {
	t.Parallel()

	t.Run("returns the first writable directory", func(t *testing.T) {
		t.Parallel()

		first := filepath.Join(t.TempDir(), "first")
		second := t.TempDir()

		got, err := appdirs.FirstWritableDir(first, second)

		require.NoError(t, err)
		assert.Equal(t, first, got)
		assert.DirExists(t, got, "the directory is created")
	})

	t.Run("skips an unwritable directory", func(t *testing.T) {
		t.Parallel()

		second := t.TempDir()

		got, err := appdirs.FirstWritableDir(unwritableRoot(t), second)

		require.NoError(t, err)
		assert.Equal(t, second, got)
	})

	t.Run("fails without a writable directory", func(t *testing.T) {
		t.Parallel()

		_, err := appdirs.FirstWritableDir(unwritableRoot(t))

		assert.ErrorContains(t, err, "no writable directory")
	})

	t.Run("fails without any directory", func(t *testing.T) {
		t.Parallel()

		_, err := appdirs.FirstWritableDir()

		assert.Error(t, err)
	})
}

func TestDirWritable(t *testing.T) {
	t.Parallel()

	t.Run("creates a missing directory", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "a", "b")

		writable := appdirs.DirWritable(dir)

		assert.True(t, writable)
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("leaves no probe file behind", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		writable := appdirs.DirWritable(dir)

		require.True(t, writable)
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("rejects a path below a file", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(unwritableRoot(t), "dir")

		writable := appdirs.DirWritable(dir)

		assert.False(t, writable)
	})

	t.Run("rejects a read-only directory", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("directory permissions are not enforced here")
		}
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		writable := appdirs.DirWritable(dir)

		assert.False(t, writable)
	})
}

// unwritableRoot returns a root below a regular file, which can never hold a directory and is never writable.
func unwritableRoot(t *testing.T) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	return filepath.Join(file, "root")
}
