package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/systray"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/i18n"
	"github.com/barthjs/swim-timer/internal/server"
	"github.com/barthjs/swim-timer/web/src/assets/images/logo"
)

// supervisedEnv marks a process that a service manager runs and restarts once it exits. The systemd units,
// the Windows service, and the container image set it, since such a process runs outside a desktop session.
var supervisedEnv = strings.ToUpper(strings.ReplaceAll(config.AppName, "-", "_")) + "_SUPERVISED"

// errUpdated ends the serve context after an installed update, to tell it from a stop or quit.
var errUpdated = errors.New("update installed")

// runServe starts the application and serves it until the process receives SIGINT or SIGTERM.
// On a desktop, a tray icon offers to open the web interface and to quit.
// After an installed update, the application stops and the process restarts into the new binary,
// unless a signal or the tray asked it to quit before.
func runServe(cmd *cobra.Command, s *state) error {
	ctx, cancel := context.WithCancelCause(cmd.Context())
	stop := func() { cancel(nil) }
	defer stop()

	texts, err := i18n.New()
	if err != nil {
		return fmt.Errorf("loading the translations: %w", err)
	}

	sources, err := structconf.PathSources(config.AppName, cmd.Flags(), &config.Config{})
	if err != nil {
		return fmt.Errorf("resolving the configuration sources: %w", err)
	}

	// The serve command keeps the updater, so it can restart into the binary an update installs
	updater := app.NewUpdater(s.build.Version)
	application, err := app.New(ctx, s.logger, s.cfg, app.Deps{
		Version:        s.build.Version,
		Updater:        updater,
		ConfigFilePath: s.configPath,
		ConfigSources:  sources,
		OnUpdated: func() {
			cancel(errUpdated)
		},
	})
	if err != nil {
		return fmt.Errorf("initializing the application: %w", err)
	}

	srv := server.New(s.logger, serverOptions(s.cfg.Server), application.Handler())
	application.Start(ctx)

	if desktopSession(runtime.GOOS, os.Getenv) {
		language := string(s.cfg.App.Language)
		tray := trayOptions{
			Title:       s.cfg.App.Name,
			URL:         srv.URL(),
			OpenLabel:   texts.T(language, "tray.open", nil),
			QuitLabel:   texts.T(language, "tray.quit", nil),
			OpenOnStart: openOnStart(s.cfg.App.Env),
		}
		go runTray(s.logger, tray, stop)
	}

	runErr := srv.Run(ctx)

	// The background work ends with the context, Close waits for it before releasing the application
	stop()
	if err := application.Close(); err != nil {
		s.logger.Error().Err(err).Msg("failed to close the application")
	}

	if runErr != nil {
		return fmt.Errorf("serving the application: %w", runErr)
	}
	if restartAfterUpdate(cmd.Context(), ctx) {
		return updater.Restart(os.Getenv(supervisedEnv) != "")
	}
	return nil
}

// restartAfterUpdate reports whether the process restarts into an installed update. Only an update that ended
// the serve context restarts it, and a signal that arrived since still ends the process instead.
func restartAfterUpdate(signals, serve context.Context) bool {
	if signals.Err() != nil {
		return false
	}
	return errors.Is(context.Cause(serve), errUpdated)
}

// desktopSession reports whether the process runs in a desktop session, which the tray icon and the browser need.
// A process of a service manager has none. Otherwise Windows and macOS always have one, and Linux needs
// an X11 or Wayland display.
func desktopSession(goos string, getenv func(string) string) bool {
	if getenv(supervisedEnv) != "" {
		return false
	}

	switch goos {
	case "windows", "darwin":
		return true
	}
	return getenv("DISPLAY") != "" || getenv("WAYLAND_DISPLAY") != ""
}

// openOnStart reports whether the web interface opens in the browser once the server runs in a desktop session,
// which it only does in production.
func openOnStart(env config.Environment) bool {
	return env == config.EnvProduction
}

// serverOptions maps the server settings of the configuration to the server options.
func serverOptions(cfg config.ServerConfig) server.Options {
	return server.Options{
		Host:        cfg.Host,
		Port:        cfg.Port,
		SSLEnabled:  cfg.SSLEnabled,
		SSLCertFile: cfg.SSLCertFile,
		SSLKeyFile:  cfg.SSLKeyFile,
		AppName:     config.AppName,
	}
}

// trayOptions holds what the tray icon shows, in the language of the user interface.
type trayOptions struct {
	Title     string
	URL       string
	OpenLabel string
	QuitLabel string
	// OpenOnStart opens the web interface in the browser right away.
	OpenOnStart bool
}

// runTray creates a system tray icon with a menu item to open and quit the application.
func runTray(logger zerolog.Logger, options trayOptions, quit func()) {
	systray.Run(func() {
		systray.SetIcon(logo.Icon)
		systray.SetTitle(options.Title)

		mOpen := systray.AddMenuItem(options.URL, options.OpenLabel)
		mQuit := systray.AddMenuItem(options.QuitLabel, options.QuitLabel)

		go func() {
			if options.OpenOnStart {
				if err := openURL(options.URL); err != nil {
					logger.Error().Err(err).Msg("failed to open URL")
				}
			}

			for {
				select {
				case <-mOpen.ClickedCh:
					if err := openURL(options.URL); err != nil {
						logger.Error().Err(err).Msg("failed to open URL")
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					quit()
					return
				}
			}
		}()
	}, func() {
	})
}

// openURL opens the specified URL in the default browser of the user.
func openURL(url string) error {
	name, args := browserCommand(runtime.GOOS, isWSL, url)
	return exec.Command(name, args...).Start() //nolint:noctx // The browser outlives the application, so no context ends it.
}

// browserCommand returns the command that opens url in the default browser on the given operating system.
func browserCommand(goos string, wsl func() bool, url string) (string, []string) {
	switch goos {
	case "windows":
		return "cmd", startArgs(url)
	case "darwin":
		return "open", []string{url}
	}
	if wsl() {
		return "cmd.exe", startArgs(url)
	}
	return "xdg-open", []string{url}
}

// startArgs returns the arguments of cmd that open url with start. The empty argument is the window title,
// since start would otherwise take a quoted URL as the title.
func startArgs(url string) []string {
	return []string{"/c", "start", "", url}
}

// isWSL reports whether the process runs inside the Windows Subsystem for Linux.
func isWSL() bool {
	releaseData, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}
