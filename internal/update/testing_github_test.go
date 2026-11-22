package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const testBinary = "demo"

// fakeGitHub serves the latest release of owner/demo with an archive and its checksums.
type fakeGitHub struct {
	tag       string
	archive   []byte
	checksums string
	status    int
	assets    bool
}

// newFakeGitHub creates a release whose archive contains content as the binary.
func newFakeGitHub(tag string, content []byte) *fakeGitHub {
	archive := archiveWith(content)
	sum := sha256.Sum256(archive)
	return &fakeGitHub{
		tag:       tag,
		archive:   archive,
		checksums: fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), assetFor()),
		status:    http.StatusOK,
		assets:    true,
	}
}

// archiveWith builds the release archive of this platform with content as the binary.
func archiveWith(content []byte) []byte {
	name := testBinary
	if runtime.GOOS == "windows" {
		name += ".exe"
		return zipWith(name, content)
	}
	return tarGzWith(name, content)
}

// zipWith returns a zip archive that holds content under name.
func zipWith(name string, content []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create(name)
	_, _ = w.Write(content)
	_ = zw.Close()
	return buf.Bytes()
}

// tarGzWith returns a gzipped tar archive that holds a readme and content under name.
func tarGzWith(name string, content []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{
		Name:     "README.md",
		Mode:     0o644,
		Size:     2,
		Typeflag: tar.TypeReg,
	})
	_, _ = tw.Write([]byte("hi"))
	_ = tw.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	})
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// start serves the release from a test server.
func (f *fakeGitHub) start(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/demo/releases/latest":
			if f.status != http.StatusOK {
				w.WriteHeader(f.status)
				return
			}
			// The request names the server, which the handler cannot reference before NewServer returns
			base := "http://" + r.Host
			assets := "[]"
			if f.assets {
				assets = fmt.Sprintf(`[{"name":%q,"browser_download_url":"%s/archive"},{"name":"checksums.txt","browser_download_url":"%s/checksums"}]`,
					assetFor(), base, base)
			}
			_, _ = fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://example.test/%s","assets":%s}`,
				f.tag, f.tag, assets)
		case "/archive":
			_, _ = w.Write(f.archive)
		case "/checksums":
			_, _ = w.Write([]byte(f.checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// assetFor returns the archive name the updater looks for on this platform.
func assetFor() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return testBinary + "_" + runtime.GOOS + "_amd64" + ext
}

// newTestUpdater creates an updater against srv whose binary is a writable temporary file.
func newTestUpdater(t *testing.T, srv *httptest.Server, version string) *Updater {
	t.Helper()

	dir := t.TempDir()
	target := filepath.Join(dir, testBinary)
	require.NoError(t, os.WriteFile(target, []byte("old"), 0o755))

	u := New(Options{
		Repository: "owner/demo",
		Binary:     testBinary,
		Version:    version,
		Arch:       "amd64",
		APIURL:     srv.URL,
	})
	u.executable = func() (string, error) { return target, nil }
	u.getenv = func(string) string { return "" }
	u.containerFile = []string{filepath.Join(dir, "missing")}
	u.dpkgInfoDir = filepath.Join(dir, "dpkg")
	return u
}

// binaryPath returns the path of the binary the test updater replaces.
func binaryPath(t *testing.T, u *Updater) string {
	t.Helper()

	path, err := u.executable()
	require.NoError(t, err)
	return path
}
