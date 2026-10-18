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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBinary = "demo"

func TestUpdater_Version(t *testing.T) {
	t.Parallel()

	t.Run("returns the running version", func(t *testing.T) {
		t.Parallel()

		u := New(Options{
			Version: "v1.2.3",
		})

		version := u.Version()

		assert.Equal(t, "v1.2.3", version)
	})
}

func TestUpdater_Check(t *testing.T) {
	t.Parallel()

	releases := []struct {
		name    string
		running string
		latest  string
		newer   bool
	}{
		{
			name:    "returns a newer release",
			running: "v1.0.0",
			latest:  "v1.1.0",
			newer:   true,
		},
		{
			name:    "returns the same release as not newer",
			running: "v1.1.0",
			latest:  "v1.1.0",
		},
		{
			name:    "returns an older release as not newer",
			running: "v2.0.0",
			latest:  "v1.1.0",
		},
		{
			name:    "returns no newer release for a development build",
			running: "dev",
			latest:  "v1.1.0",
		},
	}

	for _, tt := range releases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newFakeGitHub(tt.latest, []byte("new")).start(t)
			u := newTestUpdater(t, srv, tt.running)

			release, err := u.Check(t.Context())

			require.NoError(t, err)
			assert.Equal(t, tt.latest, release.Version)
			assert.Equal(t, "https://example.test/"+tt.latest, release.URL)
			assert.Equal(t, tt.newer, release.Newer)
		})
	}

	t.Run("sends the token", func(t *testing.T) {
		t.Parallel()

		var auth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
		}))
		t.Cleanup(srv.Close)
		u := New(Options{
			Repository: "owner/demo",
			Binary:     testBinary,
			Version:    "v1.0.0",
			APIURL:     srv.URL,
			Token:      "secret",
		})

		_, err := u.Check(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "Bearer secret", auth)
	})

	statuses := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "reports the rate limit for a forbidden response",
			status: http.StatusForbidden,
			want:   "rate limit",
		},
		{
			name:   "reports the rate limit for too many requests",
			status: http.StatusTooManyRequests,
			want:   "rate limit",
		},
		{
			name:   "reports the status of other errors",
			status: http.StatusNotFound,
			want:   "404",
		},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeGitHub("v1.0.0", nil)
			fake.status = tt.status
			u := newTestUpdater(t, fake.start(t), "v1.0.0")

			_, err := u.Check(t.Context())

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestUpdater_Supported(t *testing.T) {
	t.Parallel()

	srv := newFakeGitHub("v1.0.0", nil).start(t)

	tests := []struct {
		name   string
		modify func(u *Updater)
		want   error
	}{
		{
			name:   "accepts a release build",
			modify: func(*Updater) {},
		},
		{
			name:   "rejects a development version",
			modify: func(u *Updater) { u.options.Version = "dev" },
			want:   ErrDevelopmentBuild,
		},
		{
			name:   "rejects a build without an architecture",
			modify: func(u *Updater) { u.options.Arch = "" },
			want:   ErrDevelopmentBuild,
		},
		{
			name: "rejects a container named by the environment",
			modify: func(u *Updater) {
				u.getenv = func(key string) string {
					if key == "container" {
						return "podman"
					}
					return ""
				}
			},
			want: ErrContainer,
		},
		{
			name: "rejects a container marked by a file",
			modify: func(u *Updater) {
				marker := filepath.Join(t.TempDir(), ".dockerenv")
				require.NoError(t, os.WriteFile(marker, nil, 0o600))
				u.containerFile = []string{marker}
			},
			want: ErrContainer,
		},
		{
			name: "rejects a binary of a package",
			modify: func(u *Updater) {
				require.NoError(t, os.MkdirAll(u.dpkgInfoDir, 0o755))
				list := "/.\n/usr\n" + binaryPath(t, u) + "\n"
				require.NoError(t, os.WriteFile(filepath.Join(u.dpkgInfoDir, testBinary+".list"), []byte(list), 0o644))
			},
			want: ErrPackageManaged,
		},
		{
			name: "rejects a directory that is not writable",
			modify: func(u *Updater) {
				u.executable = func() (string, error) { return filepath.Join(t.TempDir(), "missing", testBinary), nil }
			},
			want: ErrNotWritable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := newTestUpdater(t, srv, "v1.0.0")
			tt.modify(u)

			err := u.Supported()

			if tt.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestInContainer(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), ".containerenv")
	require.NoError(t, os.WriteFile(marker, nil, 0o600))
	missing := filepath.Join(t.TempDir(), ".dockerenv")

	tests := []struct {
		name      string
		container string
		markers   []string
		want      bool
	}{
		{
			name:    "reports no container without a hint",
			markers: []string{missing},
			want:    false,
		},
		{
			name:      "detects the container variable",
			container: "podman",
			markers:   []string{missing},
			want:      true,
		},
		{
			name:    "detects a marker file",
			markers: []string{missing, marker},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string {
				if key == "container" {
					return tt.container
				}
				return ""
			}

			got := inContainer(getenv, tt.markers)

			assert.Equal(t, tt.want, got)
		})
	}
}

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
