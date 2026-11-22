package store

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
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

// errStorageFailed is the error of failingStorage.
var errStorageFailed = errors.New("storage failed")

// failingStorage is a storage that fails to write or delete one path.
type failingStorage struct {
	media.Storage
	path string
}

// Put fails for the failing path and stores everything else.
func (s *failingStorage) Put(ctx context.Context, storagePath string, r io.Reader, size int64) error {
	if storagePath == s.path {
		return errStorageFailed
	}
	return s.Storage.Put(ctx, storagePath, r, size)
}

// Delete fails for the failing path and deletes everything else.
func (s *failingStorage) Delete(ctx context.Context, storagePath string) error {
	if storagePath == s.path {
		return errStorageFailed
	}
	return s.Storage.Delete(ctx, storagePath)
}

// invalidFileNames returns names that would leave the folder of a store
// or whose file name with the extension would exceed the file name limit.
func invalidFileNames(extension string) []string {
	tooLong := strings.Repeat("a", maxFileNameBytes-len(extension)+1)
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, "/absolute", "a:b", "a?b", "a\x01b", "con", "NUL.backup", tooLong}
}

// testRecord returns a record with random values that survive the format of the tables.
func testRecord(t *testing.T) domain.Record {
	t.Helper()

	faker := apitest.Faker(t)
	return domain.Record{
		Course:     domain.CourseShort,
		Gender:     "M",
		RelayCount: 1,
		Distance:   faker.RandomInt([]int{50, 100, 200}),
		Stroke:     "FR",
		AgeFrom:    faker.IntRange(0, 12),
		AgeTo:      faker.IntRange(13, 20),
		Time:       faker.IntRange(2000, 30000),
		Approved:   faker.Bool(),
		Name:       faker.LastName() + ", " + faker.FirstName(),
		Club:       faker.Company(),
		Place:      faker.City(),
		Date:       faker.Date().Format("2006-01-02"),
		BirthYear:  faker.IntRange(1950, 2015),
	}
}
