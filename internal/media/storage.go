// Package media provides file storage behind a driver-agnostic interface,
// with interchangeable local filesystem and S3-compatible backends.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

// Driver selects the backend that stores the files.
type Driver string

const (
	DriverLocal Driver = "local"
	DriverS3    Driver = "s3"
)

var (
	// ErrInvalidPath is returned when a storage path is empty, absolute, or escapes the root.
	ErrInvalidPath = errors.New("media: invalid storage path")
	// ErrNotFound is returned when no object exists at the requested path.
	ErrNotFound = errors.New("media: object not found")
	// ErrInvalidTTL is returned when a signed URL is requested with a lifetime that is not positive.
	ErrInvalidTTL = errors.New("media: signed URL lifetime must be positive")
)

// Storage abstracts a file storage.
type Storage interface {
	// TestConnection verifies that the storage backend is reachable and usable.
	TestConnection(ctx context.Context) error

	// Exists reports whether an object exists at a path.
	Exists(ctx context.Context, path string) (bool, error)
	// Get opens the object at a path for reading, or returns ErrNotFound.
	// The caller must close the returned reader. Use ReadAll to fetch the full content at once.
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	// Put streams r to the path, creating parent prefixes as needed.
	// size is the number of bytes to write, or -1 when unknown (the backend then streams).
	Put(ctx context.Context, path string, r io.Reader, size int64) error
	// Delete removes the object at a path. Deleting a missing object is not an error.
	Delete(ctx context.Context, path string) error

	// Size returns the size of the object in bytes.
	Size(ctx context.Context, path string) (int64, error)
	// List returns all object paths (relative, slash-separated) starting with the given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Copy duplicates an existing object to a new path.
	Copy(ctx context.Context, src, dest string) error

	// SignedURL returns a URL that serves the object at path for ttl. S3 returns a presigned URL
	// that points to the bucket. The local driver returns a URL below Options.SignedURLBase,
	// which SignedHandler serves once it has checked the signature and the expiry.
	SignedURL(ctx context.Context, path string, ttl time.Duration) (string, error)
}

// Options holds the settings of a Storage.
type Options struct {
	Driver         Driver
	LocalPath      string
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool
	// S3Transport sends the S3 requests, the client's default transport if nil.
	// A custom transport can trust a private certificate authority.
	S3Transport http.RoundTripper

	// SignedURLBase is the URL prefix of the signed URLs of the local driver, where SignedHandler is mounted,
	// for example "/media" or "https://example.com/media". See Storage.SignedURL.
	SignedURLBase string
	// SigningKey signs the URLs of the local driver, a random key generated at startup if empty.
	SigningKey []byte
	// Clock returns the current time for the expiry of signed URLs of the local driver, time.Now if nil.
	// S3 checks the expiry on the server.
	Clock func() time.Time
}

// New creates a Storage for the configured Driver.
func New(options Options) (Storage, error) {
	switch options.Driver {
	case DriverLocal:
		return newLocalStorage(options)
	case DriverS3:
		return newS3Storage(options)
	default:
		return nil, fmt.Errorf("media: unknown driver %q", options.Driver)
	}
}

// Test checks that a storage with options would work, for settings that are not saved yet. Unlike New,
// it changes nothing: it creates no directory and leaves the temporary files of interrupted writes alone.
// The local driver checks that the directory, or its nearest existing parent, is writable.
// The S3 driver checks that the bucket exists and releases its connections afterward.
func Test(ctx context.Context, options Options) error {
	switch options.Driver {
	case DriverLocal:
		return testLocalStorage(ctx, options.LocalPath)
	case DriverS3:
		return testS3Storage(ctx, options)
	default:
		return fmt.Errorf("media: unknown driver %q", options.Driver)
	}
}

// ReadAll fetches the full content stored at a path and closes the reader.
// It is a convenience for callers that need the whole object in memory rather than streaming it.
func ReadAll(ctx context.Context, s Storage, path string) ([]byte, error) {
	reader, err := s.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// validatePath rejects empty, absolute, or traversal storage paths.
// A name that only starts with two dots, such as "..file", stays inside the root and is valid.
func validatePath(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned == "." || !filepath.IsLocal(cleaned) {
		return ErrInvalidPath
	}
	return nil
}
