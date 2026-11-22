package store

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/media"
)

// newTestStorage returns a local storage in a temporary directory of the test.
func newTestStorage(t *testing.T) media.Storage {
	t.Helper()

	storage, _ := newTestStorageWithRoot(t)
	return storage
}

// newTestStorageWithRoot returns a local storage in a temporary directory of the test and that directory.
func newTestStorageWithRoot(t *testing.T) (media.Storage, string) {
	t.Helper()

	root := t.TempDir()
	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: root,
	})
	require.NoError(t, err)
	return storage, root
}

// writeMeetFile stores content as a Windows-1252 encoded meet file, as the timing software exports it.
func writeMeetFile(t *testing.T, root, name, content string) {
	t.Helper()

	encoded, err := charmap.Windows1252.NewEncoder().String(content)
	require.NoError(t, err)

	dir := filepath.Join(root, meetFolder)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(encoded), 0o644))
}

// writeFile writes content to path and creates the missing parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// putFile stores data at path in storage.
func putFile(t *testing.T, storage media.Storage, storagePath string, data []byte) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), storagePath, bytes.NewReader(data), int64(len(data))))
}

// invalidFileNames returns names that would leave the folder of a store
// or whose file name with the extension would exceed the file name limit.
func invalidFileNames(extension string) []string {
	tooLong := strings.Repeat("a", maxFileNameBytes-len(extension)+1)
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, "/absolute", "a:b", "a?b", "a\x01b", "con", "NUL.backup", tooLong}
}
