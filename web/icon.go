//go:build !windows

package web

import _ "embed"

// Icon is the system tray icon. Everything but Windows accepts a PNG.
//
//go:embed public/icons/icon-192.png
var Icon []byte
