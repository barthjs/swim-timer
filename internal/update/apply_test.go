package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_Apply(t *testing.T) {
	t.Parallel()

	t.Run("replaces the binary", func(t *testing.T) {
		t.Parallel()

		srv := newFakeGitHub("v1.1.0", []byte("new binary")).start(t)
		u := newTestUpdater(t, srv, "v1.0.0")
		release, err := u.Check(t.Context())
		require.NoError(t, err)

		err = u.Apply(t.Context(), release)

		require.NoError(t, err)
		content, err := os.ReadFile(binaryPath(t, u))
		require.NoError(t, err)
		assert.Equal(t, "new binary", string(content))
	})

	tests := []struct {
		name   string
		modify func(f *fakeGitHub, u *Updater)
		want   error
		text   string
	}{
		{
			name:   "rejects a release without an asset",
			modify: func(f *fakeGitHub, _ *Updater) { f.assets = false },
			want:   ErrNoAsset,
		},
		{
			name: "rejects a wrong checksum",
			modify: func(f *fakeGitHub, _ *Updater) {
				f.checksums = fmt.Sprintf("%064d  %s\n", 0, assetFor())
			},
			text: "checksum mismatch",
		},
		{
			name:   "rejects an asset missing from the checksums",
			modify: func(f *fakeGitHub, _ *Updater) { f.checksums = "" },
			text:   "do not list",
		},
		{
			name:   "rejects a download beyond the maximum size",
			modify: func(_ *fakeGitHub, u *Updater) { u.options.MaxDownloadSize = 10 },
			text:   "larger than",
		},
		{
			name: "rejects an archive without the binary",
			modify: func(f *fakeGitHub, _ *Updater) {
				f.archive = tarGzWith("other", []byte("x"))
				if runtime.GOOS == "windows" {
					f.archive = zipWith("other.exe", []byte("x"))
				}
				sum := sha256.Sum256(f.archive)
				f.checksums = fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), assetFor())
			},
			text: "does not contain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeGitHub("v1.1.0", []byte("new binary"))
			srv := fake.start(t)
			u := newTestUpdater(t, srv, "v1.0.0")
			tt.modify(fake, u)
			release, err := u.Check(t.Context())
			require.NoError(t, err)

			err = u.Apply(t.Context(), release)

			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
			if tt.text != "" {
				assert.ErrorContains(t, err, tt.text)
			}
			content, readErr := os.ReadFile(binaryPath(t, u))
			require.NoError(t, readErr)
			assert.Equal(t, "old", string(content), "the binary stays unchanged")
		})
	}

	t.Run("rejects a development build before the download", func(t *testing.T) {
		t.Parallel()

		srv := newFakeGitHub("v1.1.0", []byte("new")).start(t)
		u := newTestUpdater(t, srv, "dev")

		err := u.Apply(t.Context(), Release{})

		assert.ErrorIs(t, err, ErrDevelopmentBuild)
	})

	t.Run("keeps the binary when stopped before the replacement", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		srv := newFakeGitHub("v1.1.0", []byte("new binary")).start(t)
		u := newTestUpdater(t, srv, "v1.0.0")
		u.client = &http.Client{
			Transport: cancelAfterDownload{
				cancel: cancel,
			},
		}
		release, err := u.Check(t.Context())
		require.NoError(t, err)

		err = u.Apply(ctx, release)

		assert.ErrorIs(t, err, context.Canceled)
		content, readErr := os.ReadFile(binaryPath(t, u))
		require.NoError(t, readErr)
		assert.Equal(t, "old", string(content), "the binary stays unchanged")
	})
}

// cancelAfterDownload is a transport that stops the update once the archive is read to its end,
// so the stop arrives after the download and before the replacement.
type cancelAfterDownload struct {
	cancel context.CancelFunc
}

// RoundTrip sends the request and wraps the body of the archive, so reading it to its end cancels the update.
func (c cancelAfterDownload) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil || request.URL.Path != "/archive" {
		return response, err
	}
	response.Body = cancelOnEOF{
		ReadCloser: response.Body,
		cancel:     c.cancel,
	}
	return response, nil
}

// cancelOnEOF cancels a context once the body it wraps is read to its end.
type cancelOnEOF struct {
	io.ReadCloser
	cancel context.CancelFunc
}

// Read reads from the body and cancels the context at its end.
func (b cancelOnEOF) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if errors.Is(err, io.EOF) {
		b.cancel()
	}
	return n, err
}

func TestUpdater_restartPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the running binary without an update", func(t *testing.T) {
		t.Parallel()

		srv := newFakeGitHub("v1.1.0", nil).start(t)
		u := newTestUpdater(t, srv, "v1.0.0")

		restartPath, err := u.restartPath()

		require.NoError(t, err)
		assert.Equal(t, binaryPath(t, u), restartPath)
	})

	t.Run("keeps the path of the replaced binary", func(t *testing.T) {
		t.Parallel()

		srv := newFakeGitHub("v1.1.0", []byte("new binary")).start(t)
		u := newTestUpdater(t, srv, "v1.0.0")
		target := binaryPath(t, u)
		release, err := u.Check(t.Context())
		require.NoError(t, err)
		require.NoError(t, u.Apply(t.Context(), release))
		// On Linux, the running process reports the renamed old binary as its executable after the replacement
		old := filepath.Join(filepath.Dir(target), "."+testBinary+".old")
		u.executable = func() (string, error) { return old, nil }

		restartPath, err := u.restartPath()

		require.NoError(t, err)
		assert.Equal(t, target, restartPath)
		content, err := os.ReadFile(restartPath)
		require.NoError(t, err)
		assert.Equal(t, "new binary", string(content))
	})
}

func TestUpdater_extract(t *testing.T) {
	t.Parallel()

	t.Run("returns the binary of this platform", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, archiveWith([]byte("new binary")))
		u := New(Options{
			Binary: testBinary,
		})

		reader, closeReader, err := u.extract(archive)

		require.NoError(t, err)
		t.Cleanup(closeReader)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "new binary", string(content))
	})

	t.Run("rejects a closed archive", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, archiveWith([]byte("new binary")))
		require.NoError(t, archive.Close())
		u := New(Options{
			Binary: testBinary,
		})

		_, _, err := u.extract(archive)

		assert.ErrorIs(t, err, os.ErrClosed)
	})
}

func TestExtractTarGz(t *testing.T) {
	t.Parallel()

	t.Run("returns the binary", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, tarGzWith("demo", []byte("linux binary")))

		reader, closeReader, err := extractTarGz(archive, "demo")

		require.NoError(t, err)
		t.Cleanup(closeReader)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "linux binary", string(content))
	})

	t.Run("returns the binary from a directory", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, tarGzWith("release/demo", []byte("linux binary")))

		reader, closeReader, err := extractTarGz(archive, "demo")

		require.NoError(t, err)
		t.Cleanup(closeReader)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "linux binary", string(content))
	})

	complete := tarGzWith("demo", []byte("linux binary"))
	tests := []struct {
		name    string
		archive []byte
		text    string
	}{
		{
			name:    "rejects a file that is no archive",
			archive: []byte("no archive"),
			text:    "reading the archive",
		},
		{
			name:    "rejects a truncated archive",
			archive: complete[:len(complete)/2],
			text:    "reading the archive",
		},
		{
			name:    "rejects an archive without the binary",
			archive: tarGzWith("other", []byte("x")),
			text:    "does not contain demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			archive := openArchive(t, tt.archive)

			_, _, err := extractTarGz(archive, "demo")

			assert.ErrorContains(t, err, tt.text)
		})
	}
}

func TestExtractZip(t *testing.T) {
	t.Parallel()

	t.Run("returns the binary", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, zipWith("demo.exe", []byte("windows binary")))

		reader, closeReader, err := extractZip(archive, "demo.exe")

		require.NoError(t, err)
		t.Cleanup(closeReader)
		var buf bytes.Buffer
		_, err = buf.ReadFrom(reader)
		require.NoError(t, err)
		assert.Equal(t, "windows binary", buf.String())
	})

	t.Run("returns the binary from a directory", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, zipWith("release/demo.exe", []byte("windows binary")))

		reader, closeReader, err := extractZip(archive, "demo.exe")

		require.NoError(t, err)
		t.Cleanup(closeReader)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "windows binary", string(content))
	})

	tests := []struct {
		name    string
		archive []byte
		text    string
	}{
		{
			name:    "rejects a file that is no archive",
			archive: []byte("no archive"),
			text:    "reading the archive",
		},
		{
			name:    "rejects an archive without the binary",
			archive: zipWith("other.exe", []byte("x")),
			text:    "does not contain demo.exe",
		},
		{
			name:    "rejects a binary with an unknown compression",
			archive: zipWithMethod("demo.exe", []byte("windows binary"), 99),
			text:    "reading the archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			archive := openArchive(t, tt.archive)

			_, _, err := extractZip(archive, "demo.exe")

			assert.ErrorContains(t, err, tt.text)
		})
	}

	t.Run("rejects a closed archive", func(t *testing.T) {
		t.Parallel()

		archive := openArchive(t, zipWith("demo.exe", []byte("windows binary")))
		require.NoError(t, archive.Close())

		_, _, err := extractZip(archive, "demo.exe")

		assert.ErrorIs(t, err, os.ErrClosed)
	})
}

// openArchive writes content to a temporary file and opens it for reading.
func openArchive(t *testing.T, content []byte) *os.File {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	archive, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = archive.Close() })
	return archive
}

// zipWithMethod returns a zip archive that holds content under name with method as the compression
// in its central directory, which the reader uses to open the file.
func zipWithMethod(name string, content []byte, method uint16) []byte {
	archive := zipWith(name, content)
	header := bytes.Index(archive, []byte("PK\x01\x02"))
	binary.LittleEndian.PutUint16(archive[header+10:], method)
	return archive
}
