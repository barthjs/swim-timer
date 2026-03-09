package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/rs/zerolog"
)

// daemonize starts the application in the background.
func daemonize(logger zerolog.Logger) {
	if os.Getenv("SWIM_TIMER_DAEMON") == "1" {
		return
	}

	executable, err := os.Executable()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to get executable path")
	}

	var args []string
	for _, arg := range os.Args[1:] {
		if arg != "--daemon" && arg != "-d" {
			args = append(args, arg)
		}
	}

	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(), "SWIM_TIMER_DAEMON=1")

	if err := cmd.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start daemon process")
	}

	fmt.Printf("Swim-Timer daemon started with PID %d\n", cmd.Process.Pid)
	os.Exit(0)
}

// printVersion prints the version information.
func printVersion() {
	// Extract VCS information from the binary itself.
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Printf("Swim-Timer %s\n", Version)
		return
	}

	var revision string
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && len(s.Value) > 7 {
			revision = s.Value[:7]
		}
	}

	fmt.Printf("Swim-Timer %s\n", Version)
	if revision != "" {
		fmt.Printf("Commit:     %s\n", revision)
	}
	fmt.Printf("Go Version: %s\n", bi.GoVersion)
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// openURL opens the specified URL in the default browser of the user.
func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		if isWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}
	if len(args) > 1 {
		// args[0] is used for 'start' command argument to prevent issues with URLs starting with a quote
		args = append(args[:1], append([]string{""}, args[1:]...)...)
	}
	return exec.Command(cmd, args...).Start()
}

// isWSL checks if the Go program is running inside Windows Subsystem for Linux
func isWSL() bool {
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}
