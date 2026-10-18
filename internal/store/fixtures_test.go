package store

import (
	"bytes"
	"os"
	"path/filepath"
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

// putFile stores data at path in storage.
func putFile(t *testing.T, storage media.Storage, storagePath string, data []byte) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), storagePath, bytes.NewReader(data), int64(len(data))))
}

// readFile reads path from storage.
func readFile(t *testing.T, storage media.Storage, storagePath string) []byte {
	t.Helper()

	data, err := media.ReadAll(t.Context(), storage, storagePath)
	require.NoError(t, err)
	return data
}

// listFiles lists every file below prefix in storage.
func listFiles(t *testing.T, storage media.Storage, prefix string) []string {
	t.Helper()

	paths, err := storage.List(t.Context(), prefix)
	require.NoError(t, err)
	return paths
}
