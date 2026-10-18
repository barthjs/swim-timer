package service

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/update"
)

func TestUpdateService_Check(t *testing.T) {
	t.Parallel()

	t.Run("returns the latest release", func(t *testing.T) {
		t.Parallel()

		svc := newTestUpdateService(t, "v1.0.0", "v1.2.0")

		check, err := svc.Check(t.Context())

		require.NoError(t, err)
		assert.Equal(t, UpdateCheck{
			Current:     "v1.0.0",
			Latest:      "v1.2.0",
			URL:         "https://example.test/v1.2.0",
			Newer:       true,
			Unsupported: UnsupportedDevelopmentBuild,
		}, check)
	})
}

func TestUpdateService_Start(t *testing.T) {
	t.Parallel()

	t.Run("marks the update as running", func(t *testing.T) {
		t.Parallel()

		svc := NewUpdateService(zerolog.Nop(), newFakeUpdater(t), nil)

		err := svc.Start()

		require.NoError(t, err)
		assert.Equal(t, UpdateStatus{
			State: UpdateRunning,
		}, svc.Status())
	})

	t.Run("rejects an unsupported installation", func(t *testing.T) {
		t.Parallel()

		svc := newTestUpdateService(t, "dev", "v1.2.0")

		err := svc.Start()

		assertValidation(t, err, "update", UnsupportedDevelopmentBuild)
		assert.Equal(t, UpdateStatus{
			State: UpdateIdle,
		}, svc.Status())
	})

	t.Run("rejects a second update while the first runs", func(t *testing.T) {
		t.Parallel()

		svc := NewUpdateService(zerolog.Nop(), newFakeUpdater(t), nil)
		require.NoError(t, svc.Start())

		err := svc.Start()

		assert.ErrorIs(t, err, apperr.ErrConflict)
	})

	t.Run("rejects a new update until the restart", func(t *testing.T) {
		t.Parallel()

		svc := NewUpdateService(zerolog.Nop(), newFakeUpdater(t), nil)
		require.NoError(t, svc.Start())
		svc.Run(t.Context())

		err := svc.Start()

		assert.ErrorIs(t, err, apperr.ErrConflict, "an installed update waits for the restart")
	})
}

func TestUpdateService_Run(t *testing.T) {
	t.Parallel()

	t.Run("records the installed release", func(t *testing.T) {
		t.Parallel()

		updater := newFakeUpdater(t)
		statusOnApplied := make(chan UpdateStatus, 1)
		svc := NewUpdateService(zerolog.Nop(), updater, nil)
		svc.onApplied = func() {
			// Reading the status deadlocks if onApplied runs while the status is locked
			statusOnApplied <- svc.Status()
		}
		require.NoError(t, svc.Start())

		done := make(chan struct{})
		go func() {
			svc.Run(t.Context())
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Run did not return, onApplied runs while the status is locked")
		}
		want := UpdateStatus{
			State:   UpdateApplied,
			Version: updater.latest,
		}
		assert.Equal(t, want, svc.Status())
		assert.Equal(t, want, <-statusOnApplied, "onApplied runs once the update is recorded")
	})

	t.Run("records a failure", func(t *testing.T) {
		t.Parallel()

		svc := newTestUpdateService(t, "v1.0.0", "v1.2.0")
		svc.status = UpdateStatus{
			State: UpdateRunning,
		}

		svc.Run(t.Context())

		status := svc.Status()
		assert.Equal(t, UpdateFailed, status.State)
		assert.Equal(t, "v1.2.0", status.Version)
		assert.NotEmpty(t, status.Error)
	})

	t.Run("ends with the context", func(t *testing.T) {
		t.Parallel()

		svc := newTestUpdateService(t, "v1.0.0", "v1.2.0")
		svc.status = UpdateStatus{
			State: UpdateRunning,
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		svc.Run(ctx)

		status := svc.Status()
		assert.Equal(t, UpdateFailed, status.State)
		assert.Contains(t, status.Error, context.Canceled.Error())
	})
}

func TestUnsupportedReason(t *testing.T) {
	t.Parallel()

	unknown := apitest.Faker(t).LoremIpsumSentence(4)
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "returns no reason for a supported installation",
			err:  nil,
			want: "",
		},
		{
			name: "returns the reason for a development build",
			err:  update.ErrDevelopmentBuild,
			want: UnsupportedDevelopmentBuild,
		},
		{
			name: "returns the reason for a container",
			err:  update.ErrContainer,
			want: UnsupportedContainer,
		},
		{
			name: "returns the reason for a wrapped package manager error",
			err:  fmt.Errorf("wrapped: %w", update.ErrPackageManaged),
			want: UnsupportedPackageManaged,
		},
		{
			name: "returns the reason for a binary that is not writable",
			err:  update.ErrNotWritable,
			want: UnsupportedNotWritable,
		},
		{
			name: "names an unknown error",
			err:  errors.New(unknown),
			want: "unknown: " + unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reason := unsupportedReason(tt.err)

			assert.Equal(t, tt.want, reason)
		})
	}
}

// newTestUpdateService returns an UpdateService whose updater asks a fake GitHub API that reports latest.
func newTestUpdateService(t *testing.T, version, latest string) *UpdateService {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.MarshalWrite(w, map[string]any{"tag_name": latest, "html_url": "https://example.test/" + latest})
	}))
	t.Cleanup(srv.Close)

	updater := update.New(update.Options{
		Repository: "owner/app",
		Binary:     "app",
		Version:    version,
		APIURL:     srv.URL,
	})
	return NewUpdateService(zerolog.Nop(), updater, nil)
}

// fakeUpdater stands in for an installation that can replace its own binary, which a test binary must never do.
type fakeUpdater struct {
	latest string
}

// newFakeUpdater returns an updater that reports a newer release and installs it successfully.
func newFakeUpdater(t *testing.T) *fakeUpdater {
	t.Helper()

	return &fakeUpdater{
		latest: "v" + apitest.Faker(t).AppVersion(),
	}
}

// Version returns the running version.
func (u *fakeUpdater) Version() string {
	return "v0.0.1"
}

// Check reports the latest release as newer.
func (u *fakeUpdater) Check(context.Context) (update.Release, error) {
	return update.Release{
		Version: u.latest,
		URL:     "https://example.test/" + u.latest,
		Newer:   true,
	}, nil
}

// Supported reports that the installation can replace itself.
func (u *fakeUpdater) Supported() error {
	return nil
}

// Apply succeeds without replacing any binary.
func (u *fakeUpdater) Apply(context.Context, update.Release) error {
	return nil
}
