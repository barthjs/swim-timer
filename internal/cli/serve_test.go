package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/server"
)

// TestSupervisedEnv keeps the variable in step with the one the Windows installation script sets.
func TestSupervisedEnv(t *testing.T) {
	t.Parallel()

	t.Run("matches the variable of the windows installation", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "SWIM_TIMER_SUPERVISED", supervisedEnv)
	})
}

func TestRestartAfterUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// stop ends the serve context or the signal context in the order of the events of the test
		stop func(signal context.CancelFunc, serve context.CancelCauseFunc)
		want bool
	}{
		{
			name: "restarts after an installed update",
			stop: func(_ context.CancelFunc, serve context.CancelCauseFunc) {
				serve(errUpdated)
			},
			want: true,
		},
		{
			name: "stops on a quit before the update",
			stop: func(_ context.CancelFunc, serve context.CancelCauseFunc) {
				serve(nil)
				serve(errUpdated)
			},
		},
		{
			name: "stops on a signal before the update",
			stop: func(signal context.CancelFunc, serve context.CancelCauseFunc) {
				signal()
				serve(errUpdated)
			},
		},
		{
			name: "stops on a signal after the update",
			stop: func(signal context.CancelFunc, serve context.CancelCauseFunc) {
				serve(errUpdated)
				signal()
			},
		},
		{
			name: "stops on a quit without an update",
			stop: func(_ context.CancelFunc, serve context.CancelCauseFunc) {
				serve(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signals, signal := context.WithCancel(t.Context())
			t.Cleanup(signal)
			serve, cancel := context.WithCancelCause(signals)
			t.Cleanup(func() { cancel(nil) })

			tt.stop(signal, cancel)

			got := restartAfterUpdate(signals, serve)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDesktopSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
		env  map[string]string
		want bool
	}{
		{
			name: "finds a desktop on windows",
			goos: "windows",
			want: true,
		},
		{
			name: "finds a desktop on macos",
			goos: "darwin",
			want: true,
		},
		{
			name: "finds an x11 display on linux",
			goos: "linux",
			env: map[string]string{
				"DISPLAY": ":0",
			},
			want: true,
		},
		{
			name: "finds a wayland display on linux",
			goos: "linux",
			env: map[string]string{
				"WAYLAND_DISPLAY": "wayland-0",
			},
			want: true,
		},
		{
			name: "finds a desktop in a systemd unit of the desktop session",
			goos: "linux",
			env: map[string]string{
				"WAYLAND_DISPLAY": "wayland-0",
				"INVOCATION_ID":   apitest.Faker(t).UUID(),
			},
			want: true,
		},
		{
			name: "finds no desktop on linux without a display",
			goos: "linux",
			want: false,
		},
		{
			name: "finds no desktop in a service on linux",
			goos: "linux",
			env: map[string]string{
				"DISPLAY":     ":0",
				supervisedEnv: "1",
			},
			want: false,
		},
		{
			name: "finds no desktop in a service on windows",
			goos: "windows",
			env: map[string]string{
				supervisedEnv: "1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string { return tt.env[key] }

			got := desktopSession(tt.goos, getenv)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenOnStart(t *testing.T) {
	t.Parallel()

	t.Run("opens the browser in production", func(t *testing.T) {
		t.Parallel()

		got := openOnStart(config.EnvProduction)

		assert.True(t, got)
	})

	t.Run("keeps the browser closed in development", func(t *testing.T) {
		t.Parallel()

		got := openOnStart(config.EnvDevelopment)

		assert.False(t, got)
	})
}

func TestServerOptions(t *testing.T) {
	t.Parallel()

	t.Run("maps the server settings", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		cfg := config.ServerConfig{
			Host:        f.IPv4Address(),
			Port:        f.IntRange(1024, 65535),
			SSLEnabled:  true,
			SSLCertFile: f.LetterN(8) + ".pem",
			SSLKeyFile:  f.LetterN(8) + ".pem",
		}

		got := serverOptions(cfg)

		assert.Equal(t, server.Options{
			Host:        cfg.Host,
			Port:        cfg.Port,
			SSLEnabled:  true,
			SSLCertFile: cfg.SSLCertFile,
			SSLKeyFile:  cfg.SSLKeyFile,
			AppName:     config.AppName,
		}, got)
	})
}

func TestBrowserCommand(t *testing.T) {
	t.Parallel()

	url := apitest.Faker(t).URL()
	isWSL := func() bool { return true }
	isNotWSL := func() bool { return false }

	tests := []struct {
		name     string
		goos     string
		wsl      func() bool
		wantName string
		wantArgs []string
	}{
		{
			name:     "passes an empty window title before the url on windows",
			goos:     "windows",
			wsl:      isNotWSL,
			wantName: "cmd",
			wantArgs: []string{"/c", "start", "", url},
		},
		{
			name:     "opens the url on macos",
			goos:     "darwin",
			wsl:      isNotWSL,
			wantName: "open",
			wantArgs: []string{url},
		},
		{
			name:     "starts the windows browser in wsl",
			goos:     "linux",
			wsl:      isWSL,
			wantName: "cmd.exe",
			wantArgs: []string{"/c", "start", "", url},
		},
		{
			name:     "uses the desktop default on linux",
			goos:     "linux",
			wsl:      isNotWSL,
			wantName: "xdg-open",
			wantArgs: []string{url},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, args := browserCommand(tt.goos, tt.wsl, url)

			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}
