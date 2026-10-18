package media_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/media"
)

func TestNewLocalStorage(t *testing.T) {
	t.Parallel()

	t.Run("creates the root", func(t *testing.T) {
		t.Parallel()

		root := filepath.Join(t.TempDir(), "nested", "root")

		newLocalStorage(t, root)

		info, err := os.Stat(root)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("removes stale temporary files", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "a"), 0o755))
		stale := filepath.Join(root, "a", ".upload-1.tmp")
		fresh := filepath.Join(root, "a", ".upload-2.tmp")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(fresh, []byte("x"), 0o644))
		old := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(stale, old, old))

		newLocalStorage(t, root)

		assert.NoFileExists(t, stale, "a temporary file left behind by a crash is removed")
		assert.FileExists(t, fresh, "a temporary file of a running write stays")
	})

	t.Run("rejects an unwritable root", func(t *testing.T) {
		t.Parallel()

		// A file in place of the root directory cannot be turned into a directory
		root := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(root, []byte("x"), 0o644))

		_, err := media.New(media.Options{
			Driver:    media.DriverLocal,
			LocalPath: root,
		})

		assert.Error(t, err)
	})
}

func TestLocalStorage_TestConnection(t *testing.T) {
	t.Parallel()

	runTestConnectionContract(t, newEmptyLocalStorage)
}

func TestTestLocalStorage(t *testing.T) {
	t.Parallel()

	t.Run("accepts a creatable root without changing anything", func(t *testing.T) {
		t.Parallel()

		parent := t.TempDir()
		stale := filepath.Join(parent, ".upload-1.tmp")
		require.NoError(t, os.WriteFile(stale, []byte("x"), 0o644))
		old := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(stale, old, old))
		root := filepath.Join(parent, "nested", "root")

		err := media.Test(t.Context(), media.Options{
			Driver:    media.DriverLocal,
			LocalPath: root,
		})

		require.NoError(t, err, "a missing directory below a writable parent can be created")
		assert.NoDirExists(t, filepath.Join(parent, "nested"), "the test creates no directory")
		assert.FileExists(t, stale, "the test removes no temporary files")
		entries, err := os.ReadDir(parent)
		require.NoError(t, err)
		assert.Len(t, entries, 1, "the write check leaves no file behind")
	})

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		path string
	}{
		{
			name: "rejects a file in place of the root",
			ctx:  t.Context(),
			path: file,
		},
		{
			name: "rejects a file in place of a parent",
			ctx:  t.Context(),
			path: filepath.Join(file, "root"),
		},
		{
			name: "rejects a canceled context",
			ctx:  canceled,
			path: t.TempDir(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := media.Test(tt.ctx, media.Options{
				Driver:    media.DriverLocal,
				LocalPath: tt.path,
			})

			assert.Error(t, err)
		})
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	t.Parallel()

	runExistsContract(t, newEmptyLocalStorage)

	t.Run("returns false for a directory", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		exists, err := storage.Exists(t.Context(), "a")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLocalStorage_Get(t *testing.T) {
	t.Parallel()

	runGetContract(t, newEmptyLocalStorage)

	t.Run("returns not found for a directory", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		_, err := storage.Get(t.Context(), "a")

		assert.ErrorIs(t, err, media.ErrNotFound)
	})
}

func TestLocalStorage_Put(t *testing.T) {
	t.Parallel()

	runPutContract(t, newEmptyLocalStorage)

	t.Run("writes a readable file without leftovers", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storage := newLocalStorage(t, root)

		err := storage.Put(t.Context(), "a/b.txt", strings.NewReader("hello"), -1)

		require.NoError(t, err)
		info, err := os.Stat(filepath.Join(root, "a", "b.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
		entries, err := os.ReadDir(filepath.Join(root, "a"))
		require.NoError(t, err)
		assert.Len(t, entries, 1, "the temporary upload file is removed")
	})

	t.Run("rejects a parent that is a file", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		err := storage.Put(t.Context(), "a/b.txt/c.txt", strings.NewReader("x"), 1)

		assert.Error(t, err)
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	t.Parallel()

	runDeleteContract(t, newEmptyLocalStorage)

	t.Run("rejects a non-empty directory", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		err := storage.Delete(t.Context(), "a")

		assert.Error(t, err)
	})
}

func TestLocalStorage_Size(t *testing.T) {
	t.Parallel()

	runSizeContract(t, newEmptyLocalStorage)

	t.Run("rejects a directory", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		_, err := storage.Size(t.Context(), "a")

		assert.ErrorIs(t, err, media.ErrInvalidPath)
	})
}

func TestLocalStorage_List(t *testing.T) {
	t.Parallel()

	runListContract(t, newEmptyLocalStorage)

	partialPrefixes := []struct {
		name   string
		prefix string
		want   []string
	}{
		{
			name:   "filters by a partial directory name",
			prefix: "captures/par",
			want: []string{
				"captures/party/a.jpg",
				"captures/parade/b.jpg",
			},
		},
		{
			name:   "filters by a partial name of files and directories",
			prefix: "capture",
			want: []string{
				"captures/party/a.jpg",
				"captures/parade/b.jpg",
				"capture.txt",
			},
		},
		{
			name:   "returns nothing for an unknown directory",
			prefix: "missing/",
			want:   nil,
		},
	}

	for _, tt := range partialPrefixes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := newLocalStorage(t, t.TempDir())
			for _, path := range []string{
				"captures/party/a.jpg",
				"captures/parade/b.jpg",
				"capture.txt",
				"prints/party/c.jpg",
			} {
				putObject(t, storage, path, "x")
			}

			paths, err := storage.List(t.Context(), tt.prefix)

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, paths)
		})
	}

	t.Run("skips hidden files", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storage := newLocalStorage(t, root)
		putObject(t, storage, "a/b.txt", "hello")
		require.NoError(t, os.WriteFile(filepath.Join(root, "a", ".hidden"), []byte("x"), 0o644))

		paths, err := storage.List(t.Context(), "")

		require.NoError(t, err)
		assert.Equal(t, []string{"a/b.txt"}, paths)
	})

	t.Run("returns nothing for a missing root", func(t *testing.T) {
		t.Parallel()

		root := filepath.Join(t.TempDir(), "root")
		storage := newLocalStorage(t, root)
		require.NoError(t, os.RemoveAll(root))

		paths, err := storage.List(t.Context(), "")

		require.NoError(t, err)
		assert.Empty(t, paths)
	})
}

func TestLocalStorage_Copy(t *testing.T) {
	t.Parallel()

	runCopyContract(t, newEmptyLocalStorage)

	t.Run("keeps the content when copying onto itself", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "note.txt", "content")

		err := storage.Copy(t.Context(), "note.txt", "note.txt")

		require.NoError(t, err)
		data, err := media.ReadAll(t.Context(), storage, "note.txt")
		require.NoError(t, err)
		assert.Equal(t, "content", string(data))
	})

	t.Run("rejects a directory as the destination", func(t *testing.T) {
		t.Parallel()

		storage := newLocalStorage(t, t.TempDir())
		putObject(t, storage, "a/b.txt", "hello")

		err := storage.Copy(t.Context(), "a/b.txt", "a")

		assert.Error(t, err)
	})
}

func TestLocalStorage_SignedURL(t *testing.T) {
	t.Parallel()

	runSignedURLContract(t, func(t *testing.T) media.Storage { return newSignedLocalStorage(t, nil) })

	t.Run("returns an escaped URL below the base", func(t *testing.T) {
		t.Parallel()

		now := time.Unix(1_700_000_000, 0)
		storage := newSignedLocalStorage(t, func() time.Time { return now })

		signed, err := storage.SignedURL(t.Context(), "a b/c.txt", time.Minute)

		require.NoError(t, err)
		parsed, err := url.Parse(signed)
		require.NoError(t, err)
		assert.Equal(t, "/media/a b/c.txt", parsed.Path, "trailing slash of the base is trimmed, segments are escaped")
		assert.Contains(t, signed, "/media/a%20b/c.txt?")
		assert.Equal(t, "1700000060", parsed.Query().Get("expires"))
		assert.NotEmpty(t, parsed.Query().Get("signature"))
	})
}

func TestSignedHandler(t *testing.T) {
	t.Parallel()

	t.Run("serves the object with a private cache lifetime", func(t *testing.T) {
		t.Parallel()

		_, signed := newSignedLocalObject(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, signed, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "private, max-age=60", response.Header.Get("Cache-Control"))
	})

	t.Run("serves the object up to and including the expiry", func(t *testing.T) {
		t.Parallel()

		clock, signed := newSignedLocalObject(t)
		clock.now = clock.now.Add(time.Minute)

		status, _ := fetch(t, signed)

		assert.Equal(t, http.StatusOK, status, "valid up to and including the expiry")
	})

	t.Run("serves a range", func(t *testing.T) {
		t.Parallel()

		signed := newSignedDigits(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, signed, nil)
		require.NoError(t, err)
		request.Header.Set("Range", "bytes=2-4")

		response, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusPartialContent, response.StatusCode)
		assert.Equal(t, "234", string(body))
	})

	t.Run("answers a head request", func(t *testing.T) {
		t.Parallel()

		signed := newSignedDigits(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodHead, signed, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("rejects other methods", func(t *testing.T) {
		t.Parallel()

		signed := newSignedDigits(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, signed, strings.NewReader("x"))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")

		response, err := http.DefaultClient.Do(request)

		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
		assert.Equal(t, "GET, HEAD", response.Header.Get("Allow"))
	})

	storage := newSignedLocalStorage(t, nil)
	putObject(t, storage, "file.txt", "x")
	putObject(t, storage, "dir/inner.txt", "x")
	signed, err := storage.SignedURL(t.Context(), "file.txt", time.Minute)
	require.NoError(t, err)
	parsed, err := url.Parse(signed)
	require.NoError(t, err)

	withQuery := func(path string, change func(url.Values)) string {
		query := parsed.Query()
		change(query)
		copied := *parsed
		copied.Path = path
		copied.RawQuery = query.Encode()
		return copied.String()
	}

	tests := []struct {
		name   string
		target string
		status int
	}{
		{
			name:   "rejects a tampered signature",
			target: withQuery(parsed.Path, func(q url.Values) { q.Set("signature", "AAAA") }),
			status: http.StatusForbidden,
		},
		{
			name:   "rejects a missing signature",
			target: withQuery(parsed.Path, func(q url.Values) { q.Del("signature") }),
			status: http.StatusForbidden,
		},
		{
			name:   "rejects an extended expiry",
			target: withQuery(parsed.Path, func(q url.Values) { q.Set("expires", "99999999999") }),
			status: http.StatusForbidden,
		},
		{
			name:   "rejects an invalid expiry",
			target: withQuery(parsed.Path, func(q url.Values) { q.Set("expires", "soon") }),
			status: http.StatusForbidden,
		},
		{
			name:   "rejects another path",
			target: withQuery("/media/dir/inner.txt", func(url.Values) {}),
			status: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, _ := fetch(t, tt.target)

			assert.Equal(t, tt.status, status)
		})
	}

	for _, path := range []string{"missing.txt", "dir"} {
		t.Run("returns not found for an unknown object at "+path, func(t *testing.T) {
			t.Parallel()

			storage := newSignedLocalStorage(t, nil)
			putObject(t, storage, "dir/inner.txt", "x")
			signed, err := storage.SignedURL(t.Context(), path, time.Minute)
			require.NoError(t, err)

			status, _ := fetch(t, signed)

			assert.Equal(t, http.StatusNotFound, status)
		})
	}

	t.Run("rejects an expired URL", func(t *testing.T) {
		t.Parallel()

		clock, signed := newSignedLocalObject(t)
		clock.now = clock.now.Add(time.Minute + time.Second)

		status, _ := fetch(t, signed)

		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("rejects a URL signed with another key", func(t *testing.T) {
		t.Parallel()

		signing := newSignedLocalStorage(t, nil)
		other := newLocalStorage(t, t.TempDir())
		putObject(t, other, "file.txt", "x")
		signed, err := signing.SignedURL(t.Context(), "file.txt", time.Minute)
		require.NoError(t, err)
		parsed, err := url.Parse(signed)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/file.txt?"+parsed.RawQuery, nil)

		// A storage with a generated key does not accept URLs signed with another key
		media.SignedHandler(other).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("returns not found for other drivers", func(t *testing.T) {
		t.Parallel()

		storage, _ := newS3Storage(t)
		recorder := httptest.NewRecorder()

		media.SignedHandler(storage).ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/file.txt", nil))

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// newEmptyLocalStorage creates a local storage in a fresh temporary directory.
func newEmptyLocalStorage(t *testing.T) media.Storage {
	t.Helper()

	return newLocalStorage(t, t.TempDir())
}

// newLocalStorage creates a local storage in root.
func newLocalStorage(t *testing.T, root string) media.Storage {
	t.Helper()

	storage, err := media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: root,
	})
	require.NoError(t, err)
	return storage
}

// fakeClock is a settable time source for the expiry of signed URLs.
type fakeClock struct {
	now time.Time
}

// Now returns the fixed fake time.
func (c *fakeClock) Now() time.Time {
	return c.now
}

// newSignedLocalObject stores an object in a signing local storage with a fake clock
// and returns the clock and a URL signed for one minute.
func newSignedLocalObject(t *testing.T) (*fakeClock, string) {
	t.Helper()

	clock := &fakeClock{
		now: time.Unix(1_700_000_000, 0),
	}
	storage := newSignedLocalStorage(t, clock.Now)
	putObject(t, storage, "file.txt", "x")
	signed, err := storage.SignedURL(t.Context(), "file.txt", time.Minute)
	require.NoError(t, err)
	return clock, signed
}

// newSignedDigits stores the ten digits in a signing local storage and returns a URL signed for one minute.
func newSignedDigits(t *testing.T) string {
	t.Helper()

	storage := newSignedLocalStorage(t, nil)
	putObject(t, storage, "file.txt", "0123456789")
	signed, err := storage.SignedURL(t.Context(), "file.txt", time.Minute)
	require.NoError(t, err)
	return signed
}

// newSignedLocalStorage creates a local storage whose signed URLs point to a test server
// that mounts SignedHandler below /media.
func newSignedLocalStorage(t *testing.T, clock func() time.Time) media.Storage {
	t.Helper()

	var storage media.Storage
	mux := http.NewServeMux()
	mux.Handle("/media/", http.StripPrefix("/media", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		media.SignedHandler(storage).ServeHTTP(w, r)
	})))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	storage, err := media.New(media.Options{
		Driver:        media.DriverLocal,
		LocalPath:     t.TempDir(),
		SignedURLBase: server.URL + "/media/",
		SigningKey:    []byte("test signing key"),
		Clock:         clock,
	})
	require.NoError(t, err)
	return storage
}
