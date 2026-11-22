//go:build !windows

package cmdline

// releaseOwnConsole does nothing, since only Windows opens a console for a binary started outside a terminal.
func releaseOwnConsole() {}
