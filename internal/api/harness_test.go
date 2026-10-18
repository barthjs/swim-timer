// Package api_test holds the feature tests of the HTTP API. Every test builds the whole application through app.New
// with its own temporary storage and fakes for everything outside the process, then drives it over HTTP
// like the web interface does.
package api_test

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/app"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/service"
)

// eventually bounds how long a test waits for background work.
const (
	eventuallyTimeout = 10 * time.Second
	eventuallyTick    = 10 * time.Millisecond
)

// harnessVersion is the version the application runs as.
const harnessVersion = "v1.0.0"

// harness is a fresh instance of the application for one test, which shares no state with other tests.
type harness struct {
	t          *testing.T
	Faker      *gofakeit.Faker
	Config     config.Config
	Deps       app.Deps
	Storage    media.Storage
	Device     *fakeDevice
	ConfigPath string

	// Client is a guest. asAdmin and newClient return clients with their own session.
	Client *apitest.Client

	// latestRelease is the version the fake GitHub API reports as the latest release.
	latestRelease string
	handler       http.Handler
}

// option changes the harness before the application is built.
type option func(*harness)

// withConfig changes the configuration before the application is built.
func withConfig(change func(*config.Config)) option {
	return func(h *harness) { change(&h.Config) }
}

// withConfigSources sets where the settings were loaded from, by JSON path.
func withConfigSources(sources map[string]structconf.Source) option {
	return func(h *harness) { h.Deps.ConfigSources = sources }
}

// withLatestRelease sets the version of the latest release the fake GitHub API reports.
func withLatestRelease(version string) option {
	return func(h *harness) { h.latestRelease = version }
}

// withUpdater replaces the updater, which otherwise asks a fake GitHub API and cannot install a release.
func withUpdater(updater service.Updater) option {
	return func(h *harness) { h.Deps.Updater = updater }
}

// accessRoute is one route of an access table with the status each kind of client receives and the error code
// where it matters. The path names the subtest. A route that needs stored data sets arrange, which stores the data
// and returns the path and body to send instead.
type accessRoute struct {
	method   string
	path     string
	body     any
	arrange  func(h *harness) (path string, body any)
	statuses map[*clientKind]int
	codes    map[*clientKind]string
}

// clientKind is a kind of client of the access tables, such as a guest or an admin. client returns a new client
// of the kind for the harness.
type clientKind struct {
	name   string
	client func(h *harness) *apitest.Client
}

// testAccess runs every route in its own harness built with options. It sends the request as each kind of client
// the route lists, in the order of clientKinds, and checks the status and the error code each of them receives.
func testAccess(t *testing.T, routes []accessRoute, opts ...option) {
	t.Helper()

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, opts...)
			for kind := range route.codes {
				require.Contains(t, route.statuses, kind, "the %s with an error code has a status", kind.name)
			}
			path, body := route.path, route.body
			if route.arrange != nil {
				path, body = route.arrange(h)
			}

			responses := make(map[*clientKind]*apitest.Response, len(route.statuses))
			for _, kind := range clientKinds {
				if _, listed := route.statuses[kind]; listed {
					responses[kind] = kind.client(h).Do(route.method, path, body)
				}
			}

			require.Len(t, responses, len(route.statuses), "every kind of client of the route is in clientKinds")
			for _, kind := range clientKinds {
				status, listed := route.statuses[kind]
				if !listed {
					continue
				}
				response := responses[kind]
				assert.Equal(t, status, response.Status, "status for the %s", kind.name)
				if code := route.codes[kind]; code != "" {
					response.AssertJSONPath("code", code)
				}
			}
		})
	}
}

// clientKinds are the kinds of clients of the access tables in the order testAccess sends their requests.
var clientKinds = []*clientKind{
	clientGuest,
	clientAdmin,
}

// clientGuest is the guest client of the harness.
var clientGuest = &clientKind{
	name:   "guest",
	client: func(h *harness) *apitest.Client { return h.Client },
}

// clientAdmin is a client with an admin session.
var clientAdmin = &clientKind{
	name:   "admin",
	client: (*harness).asAdmin,
}

// newHarness builds and starts the application. Everything it starts stops when the test ends.
func newHarness(t *testing.T, opts ...option) *harness {
	t.Helper()

	f := apitest.Faker(t)
	dir := t.TempDir()

	cfg, err := config.Defaults()
	require.NoError(t, err)
	cfg.App.Secret = f.Password(true, true, true, false, false, 16)
	cfg.Storage.Driver = media.DriverLocal
	cfg.Storage.Path = filepath.Join(dir, "data")
	cfg.Timing.SerialPort = ""
	cfg.Timing.SimulateEvent = 0
	cfg.Log.Path = filepath.Join(dir, "logs")

	h := &harness{
		t:          t,
		Faker:      f,
		Config:     cfg,
		Device:     newFakeDevice(f.LetterN(8)),
		ConfigPath: filepath.Join(dir, "config.yaml"),

		latestRelease: harnessVersion,
	}
	h.Deps = app.Deps{
		Version:        harnessVersion,
		ConfigFilePath: h.ConfigPath,
		Device:         h.Device,
	}

	for _, opt := range opts {
		opt(h)
	}
	if h.Deps.Updater == nil {
		h.Deps.Updater = newFakeUpdater(t, h.latestRelease)
	}
	require.NoError(t, config.Validate(&h.Config))

	h.Storage, err = media.New(media.Options{
		Driver:    media.DriverLocal,
		LocalPath: h.Config.Storage.Path,
	})
	require.NoError(t, err)
	h.Deps.Storage = h.Storage

	application, err := app.New(t.Context(), zerolog.Nop(), h.Config, h.Deps)
	require.NoError(t, err)
	application.Start(t.Context())
	t.Cleanup(func() { require.NoError(t, application.Close()) })

	h.handler = application.Handler()
	h.Client = apitest.NewClient(t, h.handler)

	if h.Device.connect {
		require.Eventually(t, h.Device.connected, eventuallyTimeout, eventuallyTick, "the timing service opens the device")
	}
	return h
}

// asAdmin returns a new client with an admin session.
func (h *harness) asAdmin() *apitest.Client {
	h.t.Helper()

	client := h.newClient()
	client.POST("/api/auth/login", api.AuthLoginRequest{
		Password: h.Config.App.Secret,
	}).AssertNoContent()
	return client
}

// newClient returns a client with its own session, like a second browser.
func (h *harness) newClient() *apitest.Client {
	return apitest.NewClient(h.t, h.handler)
}

// assertFieldError checks that a 422 response reports a validation error with the tag on the field.
// Field names can be dotted paths, so the body is decoded instead of read with AssertJSONPath.
func assertFieldError(t *testing.T, response *apitest.Response, field, tag string) {
	t.Helper()

	body := response.AssertUnprocessable().AssertJSONPath("code", "validation_failed").Decode[api.ValidationErrorResponse]()
	require.Contains(t, body.Errors, field, "validation errors in %s", response.Body)
	tags := make([]string, 0, len(body.Errors[field]))
	for _, fieldErr := range body.Errors[field] {
		tags = append(tags, fieldErr.Tag)
	}
	assert.Contains(t, tags, tag, "validation errors of %q", field)
}

// withDisconnectedDevice builds the application with a timing device that never opens a port.
func withDisconnectedDevice() option {
	return func(h *harness) { h.Device.connect = false }
}
