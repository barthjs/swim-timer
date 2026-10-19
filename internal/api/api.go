package api

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/health"
	"github.com/barthjs/swim-timer/internal/ratelimiter"
	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/ws"
)

// API holds the HTTP handlers and the services they call.
type API struct {
	// ctx ends with the application, so background work started by a request stops with it.
	ctx     context.Context
	options Options
	logger  zerolog.Logger
	now     func() time.Time
	wg      *sync.WaitGroup

	health      *health.Checker
	hub         *ws.Hub
	rateLimiter ratelimiter.Limiter
	sm          *scs.SessionManager

	authService      *service.AuthService
	configService    *service.ConfigService
	presetService    *service.PresetService
	meetService      *service.MeetService
	recordingService *service.RecordingService
	timingService    *service.TimingService
	updateService    *service.UpdateService
	webhookService   *service.WebhookService
}

// Options holds the settings the API reads at runtime.
type Options struct {
	// Name is the application name reported to the frontend.
	Name string
	// Language is the language of the user interface.
	Language string
	// Theme holds the accent colors of the user interface.
	Theme ThemeSettings
	// TrustedProxies may set the forwarding headers that name the client address.
	TrustedProxies []*net.IPNet
	// CORSAllowedOrigins may call the API from another domain, empty disables CORS.
	CORSAllowedOrigins []string
	// CORSAllowCredentials lets the allowed origins send cookies.
	CORSAllowCredentials bool
	// Docs serves the API documentation under /api/docs.
	Docs bool
}

// Deps are the options, infrastructure, and services the API is built from.
type Deps struct {
	Options Options
	Logger  zerolog.Logger
	// Now is the clock of the API, time.Now when nil.
	Now func() time.Time
	WG  *sync.WaitGroup

	Health      *health.Checker
	Hub         *ws.Hub
	RateLimiter ratelimiter.Limiter
	Session     *scs.SessionManager

	AuthService      *service.AuthService
	ConfigService    *service.ConfigService
	PresetService    *service.PresetService
	MeetService      *service.MeetService
	RecordingService *service.RecordingService
	TimingService    *service.TimingService
	UpdateService    *service.UpdateService
	WebhookService   *service.WebhookService
}

// New creates an API. ctx ends with the application.
func New(ctx context.Context, d Deps) *API {
	now := d.Now
	if now == nil {
		now = time.Now
	}

	return &API{
		ctx:     ctx,
		options: d.Options,
		logger:  d.Logger,
		now:     now,
		wg:      d.WG,

		health:      d.Health,
		hub:         d.Hub,
		rateLimiter: d.RateLimiter,
		sm:          d.Session,

		authService:      d.AuthService,
		configService:    d.ConfigService,
		presetService:    d.PresetService,
		meetService:      d.MeetService,
		recordingService: d.RecordingService,
		timingService:    d.TimingService,
		updateService:    d.UpdateService,
		webhookService:   d.WebhookService,
	}
}

// background runs fn in a separate goroutine with panic recovery and a context that ends after timeout
// or with the application, whichever comes first. It adds a counter to the API.wg,
// so the application waits for it before exiting.
func (a *API) background(timeout time.Duration, fn func(ctx context.Context)) {
	a.wg.Go(func() {
		defer func() {
			if err := recover(); err != nil {
				a.logger.Error().Err(fmt.Errorf("%s", err)).Msg("panic in background task")
			}
		}()

		ctx, cancel := context.WithTimeout(a.ctx, timeout)
		defer cancel()

		fn(ctx)
	})
}
