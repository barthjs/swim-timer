// Package app assembles the application from its configuration. New builds the whole application,
// and the entry point and the feature tests both use it, so they run the same wiring.
// Deps replaces the parts a test cannot use. Commands that need only some parts use the building blocks New is made of.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/health"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/ratelimiter"
	"github.com/barthjs/swim-timer/internal/serial"
	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/webhook"
	"github.com/barthjs/swim-timer/internal/ws"
)

// rateLimitCleanupInterval is how often expired rate limit entries are removed.
const rateLimitCleanupInterval = 15 * time.Minute

// Deps holds what New cannot derive from the configuration, where a zero field selects the production default.
type Deps struct {
	// ConfigFilePath is where a saved configuration is written, empty if no config file was found.
	ConfigFilePath string
	// ConfigSources reports by JSON path where each setting was loaded from, see structconf.PathSources.
	ConfigSources map[string]structconf.Source
	// OnUpdated runs once a newer release is installed, to restart into it.
	OnUpdated func()
	// Version is reported by the health endpoints and the public metadata.
	Version string

	// Device replaces the timing device selected by the configuration.
	Device service.TimingDevice
	// Storage replaces the storage selected by the configuration.
	// The recordings are still written below the local path of the configuration,
	// so a replacement must use the same root to list them.
	Storage media.Storage
	// Updater installs newer releases. Nil builds one for Version, see NewUpdater.
	Updater service.Updater
}

// App is the assembled application.
type App struct {
	logger  zerolog.Logger
	handler http.Handler

	hub         *ws.Hub
	timing      *service.TimingService
	webhooks    *service.WebhookService
	rateLimiter ratelimiter.Limiter

	// simulate replays simulated frames into the virtual port, nil without the simulator.
	simulate func(ctx context.Context)
	// background tracks the work Start runs and the work requests start, so Close can wait for it to finish.
	background sync.WaitGroup
	// closers release what New opened, in the order New opened it.
	closers   []io.Closer
	closeOnce sync.Once
}

// New assembles the application and loads its data without starting background work, see Start,
// and closes what it opened when it fails.
func New(ctx context.Context, logger zerolog.Logger, cfg config.Config, deps Deps) (_ *App, err error) {
	a := &App{
		logger: logger,
	}
	defer func() {
		if err != nil {
			_ = a.Close()
		}
	}()

	in, err := a.newInfra(logger, cfg, deps)
	if err != nil {
		return nil, err
	}
	a.rateLimiter = in.RateLimiter

	stores := NewStores(ctx, logger, in.Storage)
	meetStore := stores.Meet
	presetStore := stores.Preset
	recordingStore := stores.Recording

	device := deps.Device
	if device == nil {
		built, err := a.newDevice(logger, cfg.Timing, meetStore, recordingStore)
		if err != nil {
			return nil, err
		}
		device = built
	}

	// The preview of a webhook sends the current data of its event, which the services below hold
	var timing *service.TimingService
	var meetService *service.MeetService
	var recordBreakService *service.RecordBreakService
	a.webhooks = service.NewWebhookService(logger, webhook.New(webhookOptions(cfg, deps.Version)), stores.Webhook, func(event domain.WebhookEvent) any {
		return webhookSample(ctx, logger, event, timing, meetService, recordBreakService)
	})

	// The hub asks the timing service for the snapshot a newly connected overlay receives,
	// while the timing service broadcasts through the hub, so the snapshot is resolved lazily.
	// Clients can only connect once the application runs, after the timing service is assigned.
	a.hub = ws.New(logger, func() ([]byte, error) { return timing.Snapshot() })
	// A record set in an official heat goes to the webhooks as the list of records shows it
	recordBreakService = service.NewRecordBreakService(logger, stores.RecordBreak, stores.RecordTable, func(breaks []domain.RecordBreak) {
		a.webhooks.Publish(domain.WebhookEventRecordSet, func() any { return toRecordBreakResponses(breaks) })
	})
	// Close waits for the record breaks of the last heats, which are written in the background
	a.closers = append(a.closers, closerFunc(func() error {
		recordBreakService.Wait()
		return nil
	}))
	timing = service.NewTimingService(logger, timingOptions(cfg.Timing), a.hub, device, in.Recorder, meetStore, recordBreakService, func(heat *domain.LiveHeat) {
		a.webhooks.Publish(domain.WebhookEventHeatUpdated, func() any { return heat })
	})
	a.timing = timing

	// A reload records the meet files in use and sends the new meet to the webhooks, as the meet overview shows it
	meetService = service.NewMeetService(logger, a.hub, meetStore, func() {
		timing.RecordMeet()
		a.webhooks.Publish(domain.WebhookEventMeetReloaded, func() any { return api.ToMeetResponse(meetService.GetOverview()) })
	})

	a.handler = api.New(ctx, api.Deps{
		Options: apiOptions(cfg),
		Logger:  logger,
		WG:      &a.background,

		// Without the timing device, the overlays, the editor, and the meet overview keep working,
		// so a missing device degrades the application instead of making it unavailable.
		Health:      health.NewChecker(deps.Version, nil, health.WithOptional("timing", device)),
		Hub:         a.hub,
		RateLimiter: in.RateLimiter,
		Session:     in.Session,

		AuthService: service.NewAuthService(authOptions(cfg.App), in.Session),
		ConfigService: service.NewConfigService(cfg, service.ConfigOptions{
			FilePath: deps.ConfigFilePath,
			Sources:  deps.ConfigSources,
		}),
		PresetService:      service.NewPresetService(presetStore),
		MeetService:        meetService,
		RecordBreakService: recordBreakService,
		RecordTableService: service.NewRecordTableService(stores.RecordTable, meetStore),
		RecordingService:   service.NewRecordingService(recordingOptions(cfg.Timing), in.Recorder, recordingStore),
		TimingService:      timing,
		UpdateService:      service.NewUpdateService(logger, in.Updater, deps.OnUpdated),
		WebhookService:     a.webhooks,
	}).Routes()

	return a, nil
}

// Handler returns the HTTP handler of the application.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Start runs the background work until ctx is canceled.
func (a *App) Start(ctx context.Context) {
	a.background.Go(func() { a.hub.Run(ctx) })
	a.background.Go(func() { a.timing.Run(ctx) })
	a.background.Go(func() { a.webhooks.Run(ctx) })
	if a.simulate != nil {
		a.background.Go(func() { a.simulate(ctx) })
	}
	a.background.Go(func() { ratelimiter.RunCleanup(ctx, a.rateLimiter, rateLimitCleanupInterval) })
}

// Close waits for the background work to end, the work of Start and the work requests started,
// and releases what New opened. Cancel the context passed to Start first, otherwise Close blocks.
func (a *App) Close() error {
	a.background.Wait()

	var errs []error
	a.closeOnce.Do(func() {
		for _, closer := range a.closers {
			errs = append(errs, closer.Close())
		}
	})
	return errors.Join(errs...)
}

// newDevice opens the configured serial port. With the simulator or a replay enabled, it opens a virtual port instead
// and prepares the simulator to write into it, so simulated frames take the path of real ones.
func (a *App) newDevice(
	logger zerolog.Logger,
	cfg config.TimingConfig,
	meetStore *store.MeetStore,
	recordingStore *store.RecordingStore,
) (service.TimingDevice, error) {
	port := cfg.SerialPort

	if cfg.SimulateEvent > 0 || cfg.ReplayPath != "" {
		writer, virtualPort, err := serial.OpenVirtual()
		if err != nil {
			return nil, fmt.Errorf("opening the virtual serial port for the simulator: %w", err)
		}
		port = virtualPort

		// The simulator closes the port once its context ends, to unblock a pending write,
		// and Close releases it as well in case the simulator never ran, so both share one close.
		// The simulation returns only after its close, so the background work leaves no goroutine behind.
		closeWriter := sync.OnceValue(writer.Close)
		a.closers = append(a.closers, closerFunc(closeWriter))

		// The simulator only runs after New has assigned the timing service, see Start
		simulator := service.NewSimulatorService(logger, simulatorOptions(cfg), meetStore, recordingStore, func() { a.timing.ResetClock() })
		a.simulate = func(ctx context.Context) {
			closed := make(chan struct{})
			go func() {
				defer close(closed)
				<-ctx.Done()
				_ = closeWriter()
			}()
			simulator.Run(ctx, writer)
			<-closed
		}
	}

	return serial.New(logger, port, omega.SerialMode(cfg.BaudRate)), nil
}

// closerFunc adapts a function to io.Closer.
type closerFunc func() error

// Close calls the function.
func (f closerFunc) Close() error {
	return f()
}

// webhookSample returns the data a webhook receives with an event, as it is now.
// The sample of a record is the records of the heat that set the most recent one,
// none before the first record or if they cannot be read.
func webhookSample(
	ctx context.Context,
	logger zerolog.Logger,
	event domain.WebhookEvent,
	timing *service.TimingService,
	meetService *service.MeetService,
	recordBreakService *service.RecordBreakService,
) any {
	switch event {
	case domain.WebhookEventHeatUpdated:
		return timing.GetLiveHeat()
	case domain.WebhookEventMeetReloaded:
		return api.ToMeetResponse(meetService.GetOverview())
	case domain.WebhookEventRecordSet:
		breaks, err := recordBreakService.List(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to read the record breaks for the webhook sample")
			return []api.RecordBreakResponse{}
		}
		if len(breaks) == 0 {
			return []api.RecordBreakResponse{}
		}
		latest := breaks[0]
		return toRecordBreakResponses(slices.DeleteFunc(breaks, func(b domain.RecordBreak) bool {
			return b.Event != latest.Event || b.Heat != latest.Heat
		}))
	default:
		return nil
	}
}

// toRecordBreakResponses maps record breaks onto their responses.
func toRecordBreakResponses(breaks []domain.RecordBreak) []api.RecordBreakResponse {
	responses := make([]api.RecordBreakResponse, 0, len(breaks))
	for _, b := range breaks {
		responses = append(responses, api.ToRecordBreakResponse(b))
	}
	return responses
}
