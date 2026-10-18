// Package api serves the interactive API reference built from the swag annotations.
package api

import (
	"embed"
	"io/fs"
	"net/http"
)

// apiDocsAssets holds the built page. Some of its chunks start with an underscore, which go:embed only includes with all:.
//
//go:embed all:dist
var apiDocsAssets embed.FS

// Handler serves the interactive API reference: the page with Scalar at "/"
// and the OpenAPI document at "/doc.json", rendered with the given title and version.
// It renders a copy of SwaggerInfo, so the generated global is never changed.
func Handler(title, version string) http.Handler {
	spec := *SwaggerInfo
	spec.Title = title
	spec.Version = version
	doc := spec.ReadDoc()

	assets, err := fs.Sub(apiDocsAssets, "dist")
	if err != nil {
		// Only fails if "dist" does not exist, which the go:embed directive above
		// already guarantees at compile time.
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/doc.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(doc))
	})
	mux.Handle("/", http.FileServerFS(assets))

	return mux
}
