package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/update"
)

// UpdateTimeout bounds the download and the replacement of the binary.
const UpdateTimeout = 5 * time.Minute

// UpdateState is the phase of an update started with UpdateService.Start.
type UpdateState string

const (
	UpdateIdle    UpdateState = "idle"
	UpdateRunning UpdateState = "running"
	UpdateFailed  UpdateState = "failed"
	UpdateApplied UpdateState = "applied"
)

// UpdateStatus is the state of the last update and, once it ran, its target version or error.
type UpdateStatus struct {
	State   UpdateState
	Version string
	Error   string
}

// UpdateCheck is the result of a check for a newer release.
type UpdateCheck struct {
	Current string
	Latest  string
	URL     string
	Newer   bool
	// Unsupported names why this installation cannot replace itself, empty if it can.
	Unsupported string
}

// Reasons an installation cannot replace itself, as reported in UpdateCheck.Unsupported.
const (
	UnsupportedDevelopmentBuild = "development_build"
	UnsupportedContainer        = "container"
	UnsupportedPackageManaged   = "package_managed"
	UnsupportedNotWritable      = "not_writable"
)

// Updater finds and installs the releases of the application. update.Updater implements it,
// and tests replace it, since installing a release replaces the binary of the running process.
type Updater interface {
	Version() string
	Check(ctx context.Context) (update.Release, error)
	Supported() error
	Apply(ctx context.Context, release update.Release) error
}

// UpdateService checks for a newer release and installs it on request.
// An installed update takes effect after a restart, which onApplied triggers.
type UpdateService struct {
	logger    zerolog.Logger
	updater   Updater
	onApplied func()

	mu     sync.Mutex
	status UpdateStatus
}

// NewUpdateService creates an UpdateService. onApplied runs once the new binary is in place.
func NewUpdateService(logger zerolog.Logger, updater Updater, onApplied func()) *UpdateService {
	return &UpdateService{
		logger:    logger,
		updater:   updater,
		onApplied: onApplied,
		status: UpdateStatus{
			State: UpdateIdle,
		},
	}
}

// Check asks for the latest release and reports whether this installation can install it.
func (s *UpdateService) Check(ctx context.Context) (UpdateCheck, error) {
	release, err := s.updater.Check(ctx)
	if err != nil {
		return UpdateCheck{}, err
	}

	return UpdateCheck{
		Current:     s.updater.Version(),
		Latest:      release.Version,
		URL:         release.URL,
		Newer:       release.Newer,
		Unsupported: unsupportedReason(s.updater.Supported()),
	}, nil
}

// Start marks an update as running, the caller then installs it with Run. It fails if an update already runs,
// or with a validation error if this installation cannot replace itself.
func (s *UpdateService) Start() error {
	if reason := unsupportedReason(s.updater.Supported()); reason != "" {
		return apperr.NewValidationError("update", reason)
	}

	s.mu.Lock()
	if s.status.State == UpdateRunning || s.status.State == UpdateApplied {
		s.mu.Unlock()
		return apperr.ErrConflict
	}
	s.status = UpdateStatus{
		State: UpdateRunning,
	}
	s.mu.Unlock()

	return nil
}

// Status returns the state of the last update.
func (s *UpdateService) Status() UpdateStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Run downloads and installs the latest release after Start and records the outcome.
// ctx bounds it, UpdateTimeout is the time it needs at most.
func (s *UpdateService) Run(ctx context.Context) {
	release, err := s.updater.Check(ctx)
	if err == nil && !release.Newer {
		err = errors.New("the installed version is the latest")
	}
	if err == nil {
		err = s.updater.Apply(ctx, release)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("update failed")
		s.setStatus(UpdateStatus{
			State:   UpdateFailed,
			Version: release.Version,
			Error:   err.Error(),
		})
		return
	}

	s.logger.Info().Str("version", release.Version).Msg("update installed, restarting")
	s.setStatus(UpdateStatus{
		State:   UpdateApplied,
		Version: release.Version,
	})

	// onApplied runs without the lock, since the restart it triggers may ask for the status
	if s.onApplied != nil {
		s.onApplied()
	}
}

// setStatus records the state of the update.
func (s *UpdateService) setStatus(status UpdateStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

// unsupportedReason maps the error of update.Updater.Supported to its reason, empty if updates are supported.
func unsupportedReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, update.ErrDevelopmentBuild):
		return UnsupportedDevelopmentBuild
	case errors.Is(err, update.ErrContainer):
		return UnsupportedContainer
	case errors.Is(err, update.ErrPackageManaged):
		return UnsupportedPackageManaged
	case errors.Is(err, update.ErrNotWritable):
		return UnsupportedNotWritable
	default:
		return fmt.Sprintf("unknown: %v", err)
	}
}
