package media

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	// s3ConnectTimeout bounds the connection check when the storage is created.
	s3ConnectTimeout = 15 * time.Second
	// objectCacheControl marks objects as immutable.
	objectCacheControl = "public, max-age=31536000, immutable"
)

// s3Storage stores objects in an S3-compatible bucket.
type s3Storage struct {
	client *minio.Client
	bucket string
}

// newS3Storage creates an s3Storage for the S3 endpoint of options.
func newS3Storage(options Options) (*s3Storage, error) {
	// The storage keeps its connections for the lifetime of the process
	storage, _, err := newS3Client(options)
	if err != nil {
		return nil, err
	}

	// Without a bound, the retries of the client would keep a start without network waiting
	ctx, cancel := context.WithTimeout(context.Background(), s3ConnectTimeout)
	defer cancel()

	if err := storage.TestConnection(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

// testS3Storage checks that the bucket of options exists, within ctx and s3ConnectTimeout,
// and closes the connections of the client afterward.
func testS3Storage(ctx context.Context, options Options) error {
	storage, closeIdle, err := newS3Client(options)
	if err != nil {
		return err
	}
	defer closeIdle()

	ctx, cancel := context.WithTimeout(ctx, s3ConnectTimeout)
	defer cancel()

	return storage.TestConnection(ctx)
}

// newS3Client creates an s3Storage for the S3 endpoint of options without checking the connection.
// Without options.S3Transport, the client gets a transport of its own, whose idle connections the returned function closes.
// With a transport of the caller, the function does nothing.
func newS3Client(options Options) (*s3Storage, func(), error) {
	endpoint, err := url.Parse(options.S3Endpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("media: invalid s3 endpoint: %w", err)
	}
	if endpoint.Host == "" {
		return nil, nil, fmt.Errorf("media: s3 endpoint %q must be a full URL like https://s3.example.com", options.S3Endpoint)
	}
	secure := endpoint.Scheme == "https"

	transport := options.S3Transport
	closeIdle := func() {}
	if transport == nil {
		own, err := minio.DefaultTransport(secure)
		if err != nil {
			return nil, nil, fmt.Errorf("media: failed to create s3 transport: %w", err)
		}
		transport = own
		closeIdle = own.CloseIdleConnections
	}

	lookup := minio.BucketLookupDNS
	if options.S3UsePathStyle {
		lookup = minio.BucketLookupPath
	}

	client, err := minio.New(endpoint.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(options.S3AccessKey, options.S3SecretKey, ""),
		Secure:       secure,
		Region:       options.S3Region,
		BucketLookup: lookup,
		Transport:    transport,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("media: failed to create s3 client: %w", err)
	}

	storage := &s3Storage{
		client: client,
		bucket: options.S3Bucket,
	}
	return storage, closeIdle, nil
}

// TestConnection implements Storage.
func (s *s3Storage) TestConnection(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("media: failed to reach s3 bucket %q: %w", s.bucket, err)
	}
	if !exists {
		return fmt.Errorf("media: s3 bucket %q does not exist", s.bucket)
	}
	return nil
}

// Exists implements Storage.
func (s *s3Storage) Exists(ctx context.Context, objectPath string) (bool, error) {
	if err := validatePath(objectPath); err != nil {
		return false, err
	}

	_, err := s.client.StatObject(ctx, s.bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Get implements Storage.
func (s *s3Storage) Get(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	if err := validatePath(objectPath); err != nil {
		return nil, err
	}

	object, err := s.client.GetObject(ctx, s.bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	// GetObject is lazy, so Stat() forces the request now to surface a missing object before any read
	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		if isS3NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return object, nil
}

// Put streams r to the given path with content type and immutable cache metadata.
// A size of -1 lets the client upload with an unknown length via multipart.
func (s *s3Storage) Put(ctx context.Context, objectPath string, r io.Reader, size int64) error {
	if err := validatePath(objectPath); err != nil {
		return err
	}

	contentType := mime.TypeByExtension(path.Ext(objectPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, s.bucket, objectPath, r, size, minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: objectCacheControl,
	})

	return err
}

// Delete implements Storage.
func (s *s3Storage) Delete(ctx context.Context, objectPath string) error {
	if err := validatePath(objectPath); err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.bucket, objectPath, minio.RemoveObjectOptions{})
}

// Size implements Storage.
func (s *s3Storage) Size(ctx context.Context, objectPath string) (int64, error) {
	if err := validatePath(objectPath); err != nil {
		return 0, err
	}

	info, err := s.client.StatObject(ctx, s.bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}

	return info.Size, nil
}

// Copy implements Storage.
func (s *s3Storage) Copy(ctx context.Context, src, dest string) error {
	if err := validatePath(src); err != nil {
		return err
	}
	if err := validatePath(dest); err != nil {
		return err
	}

	srcOpts := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: src,
	}

	destOpts := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dest,
	}

	_, err := s.client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		if isS3NotFound(err) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// SignedURL implements Storage with a presigned GET URL. S3 accepts lifetimes of up to seven days.
func (s *s3Storage) SignedURL(ctx context.Context, objectPath string, ttl time.Duration) (string, error) {
	if err := validatePath(objectPath); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", ErrInvalidTTL
	}

	signed, err := s.client.PresignedGetObject(ctx, s.bucket, objectPath, ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

// List implements Storage.
func (s *s3Storage) List(ctx context.Context, prefix string) ([]string, error) {
	var paths []string

	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		paths = append(paths, object.Key)
	}

	return paths, nil
}

// isS3NotFound reports whether the error is an S3 "object does not exist" response.
func isS3NotFound(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.StatusCode == http.StatusNotFound
}
