package media_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/media"
)

const testBucket = "test-bucket"

// s3Server is an in-memory S3 server with one bucket.
type s3Server struct {
	*httptest.Server
	backend *s3mem.Backend
	// denyNext makes the next request fail with 403. Clients do not retry that status,
	// unlike server errors, so a single denied request reaches the caller.
	denyNext atomic.Bool

	mu sync.Mutex
	// putHeaders are the headers of the last PUT per object key. The backend keeps only some of them.
	putHeaders map[string]http.Header
}

func TestNewS3Storage(t *testing.T) {
	t.Parallel()

	t.Run("connects over TLS with a custom transport", func(t *testing.T) {
		t.Parallel()

		server := newS3Server(t, true)
		options := s3Options(server.URL, testBucket)
		options.S3Transport = server.Client().Transport

		storage, err := media.New(options)

		require.NoError(t, err)
		require.NoError(t, storage.Put(t.Context(), "secure.txt", strings.NewReader("tls"), 3))
		data, err := media.ReadAll(t.Context(), storage, "secure.txt")
		require.NoError(t, err)
		assert.Equal(t, "tls", string(data))
	})

	server := newS3Server(t, false)

	tests := []struct {
		name    string
		options media.Options
		want    string
	}{
		{
			name:    "rejects an endpoint without scheme",
			options: s3Options("s3.example.com", testBucket),
			want:    "full URL",
		},
		{
			name:    "rejects an unparsable endpoint",
			options: s3Options("http://[::1", testBucket),
			want:    "invalid s3 endpoint",
		},
		{
			name:    "rejects a missing bucket",
			options: s3Options(server.URL, "other-bucket"),
			want:    "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := media.New(tt.options)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.want)
		})
	}

	t.Run("rejects an untrusted certificate", func(t *testing.T) {
		t.Parallel()

		server := newS3Server(t, true)

		_, err := media.New(s3Options(server.URL, testBucket))

		assert.Error(t, err, "the test certificate is not trusted by default")
	})
}

func TestTestS3Storage(t *testing.T) {
	t.Parallel()

	t.Run("accepts an existing bucket", func(t *testing.T) {
		t.Parallel()

		fake := newS3Server(t, false)

		err := media.Test(t.Context(), s3Options(fake.URL, testBucket))

		require.NoError(t, err)
	})

	t.Run("rejects a missing bucket", func(t *testing.T) {
		t.Parallel()

		fake := newS3Server(t, false)

		err := media.Test(t.Context(), s3Options(fake.URL, "other-bucket"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "does not exist")
	})
}

func TestS3Storage_TestConnection(t *testing.T) {
	t.Parallel()

	runTestConnectionContract(t, newEmptyS3Storage)

	t.Run("returns the error of the server", func(t *testing.T) {
		t.Parallel()

		storage, fake := newS3Storage(t)
		fake.denyNext.Store(true)

		err := storage.TestConnection(t.Context())

		assert.Error(t, err)
	})
}

func TestS3Storage_Exists(t *testing.T) {
	t.Parallel()

	runExistsContract(t, newEmptyS3Storage)

	t.Run("returns other errors of the server", func(t *testing.T) {
		t.Parallel()

		storage := newDeniedS3Storage(t)

		_, err := storage.Exists(t.Context(), "note.txt")

		// Other errors pass through instead of being mistaken for a missing object
		assert.Error(t, err)
		assert.NotErrorIs(t, err, media.ErrNotFound)
	})
}

func TestS3Storage_Get(t *testing.T) {
	t.Parallel()

	runGetContract(t, newEmptyS3Storage)

	t.Run("returns other errors of the server", func(t *testing.T) {
		t.Parallel()

		storage := newDeniedS3Storage(t)

		_, err := storage.Get(t.Context(), "note.txt")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, media.ErrNotFound)
	})
}

func TestS3Storage_Put(t *testing.T) {
	t.Parallel()

	runPutContract(t, newEmptyS3Storage)

	t.Run("sends the content type and an immutable cache header", func(t *testing.T) {
		t.Parallel()

		storage, fake := newS3Storage(t)

		err := storage.Put(t.Context(), "images/photo.png", strings.NewReader("png"), 3)

		require.NoError(t, err)
		photo := fake.putHeader(t, "images/photo.png")
		assert.Equal(t, "image/png", photo.Get("Content-Type"))
		assert.Contains(t, photo.Get("Cache-Control"), "immutable")
	})

	t.Run("sends a generic content type for an unknown extension", func(t *testing.T) {
		t.Parallel()

		storage, fake := newS3Storage(t)

		err := storage.Put(t.Context(), "data.unknownext", strings.NewReader("raw"), 3)

		require.NoError(t, err)
		raw := fake.putHeader(t, "data.unknownext")
		assert.Equal(t, "application/octet-stream", raw.Get("Content-Type"))
	})
}

func TestS3Storage_Delete(t *testing.T) {
	t.Parallel()

	runDeleteContract(t, newEmptyS3Storage)
}

func TestS3Storage_Size(t *testing.T) {
	t.Parallel()

	runSizeContract(t, newEmptyS3Storage)

	t.Run("returns other errors of the server", func(t *testing.T) {
		t.Parallel()

		storage := newDeniedS3Storage(t)

		_, err := storage.Size(t.Context(), "note.txt")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, media.ErrNotFound)
	})
}

func TestS3Storage_Copy(t *testing.T) {
	t.Parallel()

	runCopyContract(t, newEmptyS3Storage)

	t.Run("returns the error of the server", func(t *testing.T) {
		t.Parallel()

		storage := newDeniedS3Storage(t)

		err := storage.Copy(t.Context(), "note.txt", "copy.txt")

		assert.Error(t, err)
	})
}

func TestS3Storage_SignedURL(t *testing.T) {
	t.Parallel()

	runSignedURLContract(t, newEmptyS3Storage)

	t.Run("returns a presigned URL", func(t *testing.T) {
		t.Parallel()

		storage, _ := newS3Storage(t)

		signed, err := storage.SignedURL(t.Context(), "a.txt", time.Hour)

		require.NoError(t, err)
		parsed, err := url.Parse(signed)
		require.NoError(t, err)
		assert.Equal(t, "/"+testBucket+"/a.txt", parsed.Path)
		assert.Equal(t, "3600", parsed.Query().Get("X-Amz-Expires"))
		assert.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
	})
}

func TestS3Storage_List(t *testing.T) {
	t.Parallel()

	runListContract(t, newEmptyS3Storage)

	t.Run("returns the error of the server", func(t *testing.T) {
		t.Parallel()

		storage := newDeniedS3Storage(t)

		_, err := storage.List(t.Context(), "")

		assert.Error(t, err)
	})
}

// newDeniedS3Storage creates an S3 storage with a stored note.txt whose server denies the next request.
func newDeniedS3Storage(t *testing.T) media.Storage {
	t.Helper()

	storage, fake := newS3Storage(t)
	fake.put(t, "note.txt", []byte("hello"))
	fake.denyNext.Store(true)
	return storage
}

// put stores an object directly in the backend.
func (s *s3Server) put(t *testing.T, key string, data []byte) {
	t.Helper()

	_, err := s.backend.PutObject(testBucket, key, nil, bytes.NewReader(data), int64(len(data)), nil)
	require.NoError(t, err)
}

// putHeader returns the headers the last PUT of key was sent with.
func (s *s3Server) putHeader(t *testing.T, key string) http.Header {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	header, ok := s.putHeaders[key]
	require.True(t, ok, "no PUT for %s", key)
	return header
}

// newEmptyS3Storage creates an S3 storage backed by an empty fake S3 server.
func newEmptyS3Storage(t *testing.T) media.Storage {
	t.Helper()

	storage, _ := newS3Storage(t)
	return storage
}

// newS3Storage creates an S3 storage backed by a fake S3 server.
func newS3Storage(t *testing.T) (media.Storage, *s3Server) {
	t.Helper()

	server := newS3Server(t, false)
	storage, err := media.New(s3Options(server.URL, testBucket))
	require.NoError(t, err)
	return storage, server
}

// newS3Server starts an in-memory S3 server with testBucket, over TLS if requested.
func newS3Server(t *testing.T, useTLS bool) *s3Server {
	t.Helper()

	backend := s3mem.New()
	require.NoError(t, backend.CreateBucket(testBucket))
	fake := gofakes3.New(backend, gofakes3.WithLogger(gofakes3.DiscardLog()))

	server := &s3Server{
		backend:    backend,
		putHeaders: map[string]http.Header{},
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.denyNext.CompareAndSwap(true, false) {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPut {
			server.mu.Lock()
			server.putHeaders[strings.TrimPrefix(r.URL.Path, "/"+testBucket+"/")] = r.Header.Clone()
			server.mu.Unlock()
		}
		fake.Server().ServeHTTP(w, r)
	})

	if useTLS {
		server.Server = httptest.NewTLSServer(handler)
	} else {
		server.Server = httptest.NewServer(handler)
	}
	t.Cleanup(server.Close)
	return server
}

// s3Options returns the storage options for the given S3 endpoint and bucket.
func s3Options(endpoint, bucket string) media.Options {
	return media.Options{
		Driver:         media.DriverS3,
		S3Endpoint:     endpoint,
		S3Region:       "us-east-1",
		S3Bucket:       bucket,
		S3AccessKey:    "access",
		S3SecretKey:    "secret",
		S3UsePathStyle: true,
	}
}
