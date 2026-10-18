package logo

import _ "embed"

// Icon is the system tray icon. Windows requires an ICO.
//
//go:embed favicon.ico
var Icon []byte
