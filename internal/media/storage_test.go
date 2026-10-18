package media_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/media"
)

// invalidPaths are rejected by every driver before touching the backend.
var invalidPaths = []string{"", ".", "/absolute.txt", "../outside.txt", "a/../../outside.txt"}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("rejects an unknown driver", func(t *testing.T) {
		t.Parallel()

		_, err := media.New(media.Options{
			Driver: "ftp",
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "ftp")
	})
}

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("rejects an unknown driver", func(t *testing.T) {
		t.Parallel()

		err := media.Test(t.Context(), media.Options{
			Driver: "ftp",
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "ftp")
	})
}

func TestReadAll(t *testing.T) {
	t.Parallel()

	t.Run("returns the error of the open", func(t *testing.T) {
		t.Parallel()

		errOpen := errors.New("open failed")
		storage := failingStorage{
			err: errOpen,
		}

		_, err := media.ReadAll(t.Context(), storage, "file.txt")

		assert.ErrorIs(t, err, errOpen)
	})

	t.Run("returns the error of the read", func(t *testing.T) {
		t.Parallel()

		storage := failingStorage{
			reader: brokenReader{},
		}

		_, err := media.ReadAll(t.Context(), storage, "file.txt")

		assert.EqualError(t, err, "disk error")
	})
}

// failingStorage is a Storage whose reads fail, to test helpers built on top of the interface.
type failingStorage struct {
	media.Storage
	reader io.ReadCloser
	err    error
}

// Get returns the configured reader and error.
func (f failingStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return f.reader, f.err
}

// brokenReader is a reader that always fails.
type brokenReader struct{}

// Read always returns an error.
func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("disk error") }

// Close does nothing.
func (brokenReader) Close() error { return nil }

// runTestConnectionContract checks the connection test every Storage driver must offer.
// newStorage returns an empty storage for each subtest.
func runTestConnectionContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("succeeds", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		err := storage.TestConnection(t.Context())

		assert.NoError(t, err)
	})
}

// runExistsContract checks the existence check every Storage driver must offer.
func runExistsContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("returns false for an unknown object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		exists, err := storage.Exists(t.Context(), "missing.txt")

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			_, err := storage.Exists(t.Context(), path)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})
}

// runGetContract checks the reads every Storage driver must offer.
func runGetContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("returns the content", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		content := strings.Repeat("0123456789", 1000)
		putObject(t, storage, "large.bin", content)

		reader, err := storage.Get(t.Context(), "large.bin")

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()
		var buffer bytes.Buffer
		_, err = io.Copy(&buffer, reader)
		require.NoError(t, err)
		assert.Equal(t, content, buffer.String())
	})

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			_, err := storage.Get(t.Context(), path)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})

	t.Run("returns not found for an unknown object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		_, err := storage.Get(t.Context(), "missing.txt")
		_, readErr := media.ReadAll(t.Context(), storage, "missing.txt")

		assert.ErrorIs(t, err, media.ErrNotFound)
		assert.ErrorIs(t, readErr, media.ErrNotFound)
	})
}

// runPutContract checks the writes every Storage driver must offer.
func runPutContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("stores the object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		err := storage.Put(t.Context(), "docs/note.txt", strings.NewReader("hello"), 5)

		require.NoError(t, err)
		data, err := media.ReadAll(t.Context(), storage, "docs/note.txt")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
		exists, err := storage.Exists(t.Context(), "docs/note.txt")
		require.NoError(t, err)
		assert.True(t, exists)
		size, err := storage.Size(t.Context(), "docs/note.txt")
		require.NoError(t, err)
		assert.Equal(t, int64(5), size)
	})

	t.Run("replaces the object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		putObject(t, storage, "note.txt", "first")

		err := storage.Put(t.Context(), "note.txt", strings.NewReader("second"), 6)

		require.NoError(t, err)
		data, err := media.ReadAll(t.Context(), storage, "note.txt")
		require.NoError(t, err)
		assert.Equal(t, "second", string(data))
	})

	t.Run("keeps a name starting with two dots inside the root", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		err := storage.Put(t.Context(), "..note.txt", strings.NewReader("dots"), 4)

		require.NoError(t, err)
		data, err := media.ReadAll(t.Context(), storage, "..note.txt")
		require.NoError(t, err)
		assert.Equal(t, "dots", string(data))
	})

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			err := storage.Put(t.Context(), path, strings.NewReader("x"), 1)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})
}

// runDeleteContract checks the deletes every Storage driver must offer.
func runDeleteContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("removes the object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		putObject(t, storage, "note.txt", "hello")

		err := storage.Delete(t.Context(), "note.txt")

		require.NoError(t, err)
		exists, err := storage.Exists(t.Context(), "note.txt")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			err := storage.Delete(t.Context(), path)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})

	t.Run("ignores an unknown object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		err := storage.Delete(t.Context(), "missing.txt")

		assert.NoError(t, err)
	})
}

// runSizeContract checks the size lookups every Storage driver must offer.
func runSizeContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			_, err := storage.Size(t.Context(), path)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})

	t.Run("returns not found for an unknown object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		_, err := storage.Size(t.Context(), "missing.txt")

		assert.ErrorIs(t, err, media.ErrNotFound)
	})
}

// runListContract checks the listings every Storage driver must offer.
func runListContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{
			name:   "returns the objects",
			prefix: "",
			want:   []string{"images/a.png", "images/nested/b.png", "docs/c.txt"},
		},
		{
			name:   "filters by prefix",
			prefix: "images/",
			want:   []string{"images/a.png", "images/nested/b.png"},
		},
		{
			name:   "returns nothing for an unknown prefix",
			prefix: "videos/",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := newStorage(t)
			putObject(t, storage, "images/a.png", "a")
			putObject(t, storage, "images/nested/b.png", "b")
			putObject(t, storage, "docs/c.txt", "c")

			paths, err := storage.List(t.Context(), tt.prefix)

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, paths)
		})
	}
}

// runCopyContract checks the copies every Storage driver must offer.
func runCopyContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("stores the copy", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		putObject(t, storage, "source.txt", "hello")

		err := storage.Copy(t.Context(), "source.txt", "nested/copy.txt")

		require.NoError(t, err)
		data, err := media.ReadAll(t.Context(), storage, "nested/copy.txt")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
		data, err = media.ReadAll(t.Context(), storage, "source.txt")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data), "the source stays")
	})

	t.Run("rejects an invalid source path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			err := storage.Copy(t.Context(), path, "valid.txt")

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})

	t.Run("rejects an invalid destination path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		for _, path := range invalidPaths {
			err := storage.Copy(t.Context(), "valid.txt", path)

			assert.ErrorIs(t, err, media.ErrInvalidPath, path)
		}
	})

	t.Run("returns not found for an unknown source", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		err := storage.Copy(t.Context(), "missing.txt", "copy.txt")

		assert.ErrorIs(t, err, media.ErrNotFound)
	})
}

// runSignedURLContract checks the signed URLs every driver must offer:
// an absolute URL that serves the object without credentials until it expires.
func runSignedURLContract(t *testing.T, newStorage func(t *testing.T) media.Storage) {
	t.Helper()

	t.Run("serves the object", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)
		putObject(t, storage, "dir/file name.txt", "signed content")

		signed, err := storage.SignedURL(t.Context(), "dir/file name.txt", time.Minute)

		require.NoError(t, err)
		status, body := fetch(t, signed)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, []byte("signed content"), body)
	})

	t.Run("rejects an invalid lifetime", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		_, err := storage.SignedURL(t.Context(), "file.txt", 0)

		require.ErrorIs(t, err, media.ErrInvalidTTL)
	})

	t.Run("rejects an invalid path", func(t *testing.T) {
		t.Parallel()

		storage := newStorage(t)

		_, err := storage.SignedURL(t.Context(), "../escape.txt", time.Minute)

		require.ErrorIs(t, err, media.ErrInvalidPath)
	})
}

// putObject stores content at path and fails the test on an error.
func putObject(t *testing.T, storage media.Storage, path, content string) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), path, strings.NewReader(content), int64(len(content))))
}

// fetch sends a GET request to the URL and returns the status and body.
func fetch(t *testing.T, rawURL string) (int, []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response.StatusCode, body
}
