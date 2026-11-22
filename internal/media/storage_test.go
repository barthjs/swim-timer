package media_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/media"
)

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
