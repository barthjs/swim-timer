package media_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
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
