// Package docs serves the documentation and the API reference built from this directory.
package docs

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// assets holds the built site. Its bundles live in _astro, which go:embed only includes with all:.
//
//go:embed all:dist
var assets embed.FS

// Handler serves the built site mounted at the path root. Its pages reference their assets relative to their own path,
// so a path without a page is redirected to the start page instead of showing a page that cannot load its styles.
func Handler(root string) http.Handler {
	site, err := fs.Sub(assets, "dist")
	if err != nil {
		// Only fails if "dist" does not exist, which the go:embed directive above
		// already guarantees at compile time.
		panic(err)
	}

	fileServer := http.FileServerFS(site)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !exists(site, r.URL.Path) {
			http.Redirect(w, r, root, http.StatusFound)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

// exists reports whether a path names a file of the site or a directory with a page.
func exists(site fs.FS, urlPath string) bool {
	name := strings.Trim(path.Clean("/"+urlPath), "/")
	if name == "" {
		name = "."
	}

	info, err := fs.Stat(site, name)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return true
	}

	_, err = fs.Stat(site, path.Join(name, "index.html"))

	return err == nil
}
