package web

import _ "embed"

// Icon is the system tray icon. Windows requires an ICO.
//
//go:embed public/favicon.ico
var Icon []byte
