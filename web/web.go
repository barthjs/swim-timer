package web

import "embed"

// Files holds the built frontend. Vite names shared chunks with a leading underscore,
// which go:embed only includes with all:.
//
//go:embed all:dist
var Files embed.FS
