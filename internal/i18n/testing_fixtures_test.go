package i18n_test

import (
	"testing/fstest"
)

// catalogs builds a file system with one catalog per locale, as the embedded locales directory holds them.
func catalogs(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for locale, content := range files {
		fsys["locales/"+locale+".json"] = &fstest.MapFile{
			Data: []byte(content),
		}
	}
	return fsys
}
