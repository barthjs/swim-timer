package main

import (
	"context"

	"fyne.io/systray"
	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/web/src/assets/images/logo"
)

// runTray creates a system tray icon with a menu item to open and quit the application.
func runTray(cancel context.CancelFunc, logLevel zerolog.Level, logger zerolog.Logger, url string) {
	systray.Run(func() {
		systray.SetIcon(logo.Icon)
		systray.SetTitle("Swim Timer")

		mOpen := systray.AddMenuItem(url, "Open")
		mQuit := systray.AddMenuItem("Quit", "Quit")

		go func() {
			if logLevel > zerolog.DebugLevel {
				_ = openURL(url)
			}

			for {
				select {
				case <-mOpen.ClickedCh:
					if err := openURL(url); err != nil {
						logger.Warn().Err(err).Msg("failed to open URL")
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					cancel()
					return
				}
			}
		}()
	}, func() {
	})
}
