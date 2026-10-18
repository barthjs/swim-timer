// Package appdirs resolves the per-OS locations of an application's configuration, data, and logs.
// Configuration is searched in the working directory, the user's config directory, and the system-wide config directory.
// Data and logs are kept in the system-wide or the user's directory.
package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Roots are the base directories the application directories are derived from.
// An empty root is skipped, for example when the OS exposes no user config directory.
type Roots struct {
	// User is the current user's root, such as ~/.config (config) or ~/.local/share (data) on Linux.
	User string
	// System is the system-wide root, such as /etc (config) or /var/lib (data) on Linux.
	System string
}

// ConfigRoots returns the config roots of the current OS and user.
func ConfigRoots() Roots {
	user, err := os.UserConfigDir()
	if err != nil {
		user = ""
	}
	return Roots{
		User:   user,
		System: systemConfigRoot(),
	}
}

// DataRoots returns the data roots of the current OS and user.
func DataRoots() Roots {
	return Roots{
		User:   userDataRoot(),
		System: systemDataRoot(),
	}
}

// ConfigSearchDirs returns the config directories to search for an existing file, see Roots.SearchDirs.
func ConfigSearchDirs(appName, subdir string) []string {
	return ConfigRoots().SearchDirs(appName, subdir)
}

// ConfigWriteDirs returns the config directories to write a new file into, see Roots.WriteDirs.
func ConfigWriteDirs(appName, subdir string) []string {
	return ConfigRoots().WriteDirs(appName, subdir)
}

// DataDir returns the directory the application keeps its persistent data in,
// below the data roots of the current OS and user (see Roots.DataDir).
func DataDir(appName string) (string, error) {
	return DataRoots().DataDir(appName)
}

// DataDirs returns the candidates of DataDir in the order it tries them, without creating any (see Roots.DataDirs).
func DataDirs(appName string) []string {
	return DataRoots().DataDirs(appName)
}

// LogDirs returns the log directories of appName on the current OS and user, the system-wide one first.
// A location the OS does not expose is omitted.
func LogDirs(appName string) []string {
	return joinDirs("", systemLogDir(appName), userLogDir(appName))
}

// LogDir returns the first writable of the application's system and user log directory, creating it if missing.
func LogDir(appName string) (string, error) {
	return FirstWritableDir(LogDirs(appName)...)
}

// SearchDirs returns the directories to search for an existing file, in the order they should be tried:
// the working directory, then the user root, then the system root.
// appName names the application's subdirectory below the roots.
// A non-empty subdir is appended to each directory, and an empty root is omitted.
func (r Roots) SearchDirs(appName, subdir string) []string {
	return joinDirs(subdir, ".", r.appDir(r.User, appName), r.appDir(r.System, appName))
}

// WriteDirs returns the same directories in the order a fresh installation picks one to write a new file into:
// the system root, the user root, then the working directory.
func (r Roots) WriteDirs(appName, subdir string) []string {
	return joinDirs(subdir, r.appDir(r.System, appName), r.appDir(r.User, appName), ".")
}

// DataDir returns the first writable of the application's system and user directory, creating it if missing.
// The working directory is no candidate, so the location does not depend on how the process was started.
func (r Roots) DataDir(appName string) (string, error) {
	return FirstWritableDir(r.DataDirs(appName)...)
}

// DataDirs returns the application's system and user directory in the order DataDir tries them,
// without creating them. An empty root is omitted.
func (r Roots) DataDirs(appName string) []string {
	return joinDirs("", r.appDir(r.System, appName), r.appDir(r.User, appName))
}

// appDir joins a root with the application's directory name, or returns empty for an empty root.
func (r Roots) appDir(root, appName string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, appName)
}

// FirstWritableDir returns the first of dirs that is writable (see DirWritable), creating it if missing.
func FirstWritableDir(dirs ...string) (string, error) {
	for _, dir := range dirs {
		if DirWritable(dir) {
			return dir, nil
		}
	}

	return "", fmt.Errorf("no writable directory found (tried %v)", dirs)
}

// DirWritable reports whether a dir exists (creating it if missing) and accepts a new file.
func DirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}

	probe, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return false
	}

	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)

	return true
}

// systemConfigRoot is the system-wide config root of the current OS.
func systemConfigRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support"
	case "windows":
		return os.Getenv("ProgramData")
	default:
		return "/etc"
	}
}

// userDataRoot is the current user's data root, or empty if the OS exposes none. On Linux and other Unix systems,
// it follows the XDG Base Directory Specification, which ignores a relative XDG_DATA_HOME.
func userDataRoot() string {
	switch runtime.GOOS {
	case "darwin":
		dir, err := os.UserConfigDir()
		if err != nil {
			return ""
		}
		return dir
	case "windows":
		return os.Getenv("LocalAppData")
	}

	if dir := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share")
}

// systemDataRoot is the system-wide data root of the current OS.
func systemDataRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support"
	case "windows":
		return os.Getenv("ProgramData")
	default:
		return "/var/lib"
	}
}

// systemLogDir is the system-wide log directory of appName on the current OS, or empty if the OS exposes none.
func systemLogDir(appName string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join("/Library/Logs", appName)
	case "windows":
		root := os.Getenv("ProgramData")
		if root == "" {
			return ""
		}
		return filepath.Join(root, appName, "logs")
	default:
		return filepath.Join("/var/log", appName)
	}
}

// userLogDir is the current user's log directory of appName, or empty if the OS exposes none.
// On Linux and other Unix systems, it follows the XDG Base Directory Specification for state data,
// which ignores a relative XDG_STATE_HOME.
func userLogDir(appName string) string {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "Library", "Logs", appName)
	case "windows":
		root := os.Getenv("LocalAppData")
		if root == "" {
			return ""
		}
		return filepath.Join(root, appName, "logs")
	}

	if dir := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, appName, "logs")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", appName, "logs")
}

// joinDirs appends subdir to each non-empty entry of dirs, in order.
func joinDirs(subdir string, dirs ...string) []string {
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if subdir != "" {
			dir = filepath.Join(dir, subdir)
		}
		result = append(result, dir)
	}
	return result
}
