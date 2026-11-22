package cmdline

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
)

// releaseOwnConsole closes the console window that Windows opens for a binary started outside a terminal.
// A console that the binary shares with a terminal stays attached.
func releaseOwnConsole() {
	// Windows creates a console of its own only for a process started outside one,
	// so this process is then the only one attached to it
	var pids [2]uint32
	count, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	if count != 1 {
		return
	}

	// Without a console, a failure has nowhere to go, and the window merely stays open
	_, _, _ = procFreeConsole.Call()
}
