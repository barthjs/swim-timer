package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
