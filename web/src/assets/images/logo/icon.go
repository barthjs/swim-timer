//go:build !windows

package logo

import _ "embed"

// Icon is the system tray icon. Everything but Windows accepts a PNG.
//
//go:embed favicon.png
var Icon []byte
