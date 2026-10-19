package api_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/config/structconf"
	"github.com/barthjs/swim-timer/internal/media"
)

func TestConfigAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/config",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/config",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusBadRequest,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/config/storage/test",
			body: api.StorageSettings{
				Driver: string(media.DriverLocal),
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
			codes: map[*clientKind]string{
				clientGuest: "authentication_required",
			},
		},
	})
}

func TestConfigGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the configuration", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.asAdmin().GET("/api/config").AssertOK().
			AssertJSONPath("app.name", h.Config.App.Name).
			AssertJSONPath("server.port", h.Config.Server.Port).
			AssertJSONPath("storage.path", h.Config.Storage.Path).
			AssertJSONPath("timing.baudRate", h.Config.Timing.BaudRate)
	})

	t.Run("hides the secrets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/config").AssertOK().
			AssertJSONPath("app.secret", "").
			AssertJSONPath("storage.s3.secretKey", "").
			AssertJSONPath(`secrets.app\.secret`, true).
			AssertJSONPath(`secrets.storage\.s3\.secretKey`, false)

		assert.NotContains(t, string(response.Body), h.Config.App.Secret)
	})

	t.Run("returns the defaults", func(t *testing.T) {
		t.Parallel()
		defaults, err := config.Defaults()
		require.NoError(t, err)
		h := newHarness(t, withConfig(func(cfg *config.Config) {
			cfg.Timing.PoolLength = defaults.Timing.PoolLength + 1
		}))

		h.asAdmin().GET("/api/config").AssertOK().
			AssertJSONPath("timing.poolLength", h.Config.Timing.PoolLength).
			AssertJSONPath("defaults.timing.poolLength", defaults.Timing.PoolLength).
			AssertJSONPath("defaults.server.port", defaults.Server.Port).
			AssertJSONPath("defaults.server.host", "").
			AssertJSONPath("defaults.app.secret", "")
	})

	t.Run("returns the sources", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfigSources(map[string]structconf.Source{
			"log.level": {
				Kind: structconf.SourceEnv,
				Name: "LOG_LEVEL",
			},
			"server.port": {
				Kind: structconf.SourceFlag,
				Name: "--server-port",
			},
			"storage.s3.secretKey": {
				Kind: structconf.SourceDefault,
			},
		}))

		h.asAdmin().GET("/api/config").AssertOK().
			AssertJSONPath(`sources.log\.level.kind`, "env").
			AssertJSONPath(`sources.log\.level.name`, "LOG_LEVEL").
			AssertJSONPath(`sources.server\.port.kind`, "flag").
			AssertJSONPath(`sources.server\.port.name`, "--server-port").
			AssertJSONPath(`sources.storage\.s3\.secretKey.kind`, "default").
			AssertJSONPath("file.path", h.ConfigPath).
			AssertJSONPath("file.exists", false)
	})

	t.Run("names the sources like the fields of the form", func(t *testing.T) {
		t.Parallel()
		sources := map[string]structconf.Source{}
		for _, leaf := range structconf.Leaves(&config.Config{}) {
			sources[leaf.Path] = structconf.Source{
				Kind: structconf.SourceDefault,
			}
		}
		h := newHarness(t, withConfigSources(sources))

		response := h.asAdmin().GET("/api/config").AssertOK()
		response.AssertJSONCount("sources", len(sources))
		body := response.Decode[map[string]any]()
		for jsonPath := range sources {
			assert.True(t, hasJSONPath(body, jsonPath), "the source %s is keyed like a field of the form", jsonPath)
		}
	})
}

func TestConfigSaveHandler(t *testing.T) {
	t.Parallel()

	t.Run("writes the config file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()
		request := newConfigRequest(h.Faker, h.Config)

		admin.PUT("/api/config", request).AssertOK().
			AssertJSONPath("app.name", request.App.Name).
			AssertJSONPath("server.port", request.Server.Port).
			AssertJSONPath("timing.serialPort", request.Timing.SerialPort)

		admin.GET("/api/config").AssertOK().
			AssertJSONPath("app.name", request.App.Name).
			AssertJSONPath("timing.serialPort", request.Timing.SerialPort)

		written, err := os.ReadFile(h.ConfigPath)
		require.NoError(t, err)
		assert.Contains(t, string(written), request.App.Name)
		assert.Contains(t, string(written), request.Timing.SerialPort)

		// The running application keeps its configuration until it is restarted
		h.Client.GET("/api/meta").AssertJSONPath("name", h.Config.App.Name)
	})

	t.Run("updates the sources and the file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfigSources(map[string]structconf.Source{
			"log.level": {
				Kind: structconf.SourceEnv,
				Name: "LOG_LEVEL",
			},
			"app.name": {
				Kind: structconf.SourceDefault,
			},
		}))

		h.asAdmin().PUT("/api/config", newConfigRequest(h.Faker, h.Config)).AssertOK().
			AssertJSONPath(`sources.app\.name.kind`, "file").
			AssertJSONPath(`sources.app\.name.name`, h.ConfigPath).
			AssertJSONPath(`sources.log\.level.kind`, "env").
			AssertJSONPath("file.exists", true)
	})

	t.Run("keeps a secret sent empty", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		request := newConfigRequest(h.Faker, h.Config)
		request.App.Secret = ""

		h.asAdmin().PUT("/api/config", request).AssertOK().
			AssertJSONPath("app.secret", "").
			AssertJSONPath(`secrets.app\.secret`, true)

		written, err := os.ReadFile(h.ConfigPath)
		require.NoError(t, err)
		assert.Contains(t, string(written), h.Config.App.Secret, "the file keeps the current secret")
	})

	t.Run("replaces the secret", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		request := newConfigRequest(h.Faker, h.Config)

		h.asAdmin().PUT("/api/config", request).AssertOK().AssertJSONPath("app.secret", "")

		written, err := os.ReadFile(h.ConfigPath)
		require.NoError(t, err)
		assert.Contains(t, string(written), request.App.Secret)
		assert.NotContains(t, string(written), h.Config.App.Secret)
	})

	t.Run("clears a listed secret", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		request := newConfigRequest(h.Faker, h.Config)
		request.App.Secret = ""
		request.ClearSecrets = []string{"app.secret"}

		h.asAdmin().PUT("/api/config", request).AssertOK().AssertJSONPath(`secrets.app\.secret`, false)

		written, err := os.ReadFile(h.ConfigPath)
		require.NoError(t, err)
		assert.NotContains(t, string(written), h.Config.App.Secret)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*api.ConfigSaveRequest, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "port out of range",
				change: func(r *api.ConfigSaveRequest, _ *gofakeit.Faker) { r.Server.Port = 65536 },
				field:  "server.port",
				tag:    "max",
			},
			{
				name:   "unknown language",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) { r.App.Language = f.LetterN(5) },
				field:  "app.language",
				tag:    "oneof",
			},
			{
				name:   "unknown log channel",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) { r.Log.Channel = f.LetterN(5) },
				field:  "log.channel",
				tag:    "oneof",
			},
			{
				name:   "color not hex",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) { r.Theme.Primary = f.Word() },
				field:  "theme.primary",
				tag:    "hexcolor",
			},
			{
				name:   "unknown storage driver",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) { r.Storage.Driver = f.LetterN(5) },
				field:  "storage.driver",
				tag:    "oneof",
			},
			{
				name:   "unsupported baud rate",
				change: func(r *api.ConfigSaveRequest, _ *gofakeit.Faker) { r.Timing.BaudRate = 1234 },
				field:  "timing.baudRate",
				tag:    "oneof",
			},
			{
				name:   "negative pool length",
				change: func(r *api.ConfigSaveRequest, _ *gofakeit.Faker) { r.Timing.PoolLength = -1 },
				field:  "timing.poolLength",
				tag:    "min",
			},
			{
				name:   "invalid trusted proxy",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) { r.Server.TrustedProxies = f.Word() },
				field:  "server.trustedProxies",
				tag:    "networks",
			},
			{
				name: "origin with a path",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) {
					r.Server.CORSAllowedOrigins = []string{f.URL() + "/app"}
				},
				field: "server.corsAllowedOrigins[0]",
				tag:   "origin",
			},
			{
				name: "cookies for every origin",
				change: func(r *api.ConfigSaveRequest, _ *gofakeit.Faker) {
					r.Server.CORSAllowedOrigins = []string{"*"}
					r.Server.CORSAllowCredentials = true
				},
				field: "server.corsAllowCredentials",
				tag:   "explicit_origins",
			},
			{
				name: "cookies for every origin written with a trailing slash",
				change: func(r *api.ConfigSaveRequest, _ *gofakeit.Faker) {
					r.Server.CORSAllowedOrigins = []string{" */"}
					r.Server.CORSAllowCredentials = true
				},
				field: "server.corsAllowCredentials",
				tag:   "explicit_origins",
			},
			{
				name: "invalid allowed webhook network",
				change: func(r *api.ConfigSaveRequest, f *gofakeit.Faker) {
					r.Webhook.AllowedNetworks = "10.0.0.0/8," + f.Word()
				},
				field: "webhook.allowedNetworks",
				tag:   "networks",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				request := newConfigRequest(h.Faker, h.Config)
				tt.change(&request, h.Faker)

				response := h.asAdmin().PUT("/api/config", request)

				assertFieldError(t, response, tt.field, tt.tag)
				_, err := os.Stat(h.ConfigPath)
				assert.ErrorIs(t, err, os.ErrNotExist, "nothing is written")
			})
		}
	})

	t.Run("requires the s3 settings", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		request := newConfigRequest(h.Faker, h.Config)
		request.Storage.Driver = string(media.DriverS3)

		response := h.asAdmin().PUT("/api/config", request)

		for _, field := range []string{"storage.s3.endpoint", "storage.s3.bucket", "storage.s3.accessKey"} {
			assertFieldError(t, response, field, "required_if")
		}
	})

	t.Run("requires missing secrets", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()

		// The secrets are never sent to the admin interface, so a secret is only missing if none is saved either
		storage := newConfigRequest(h.Faker, h.Config)
		storage.Storage = newS3StorageSettings(h.Faker)
		storage.Storage.S3.SecretKey = ""
		assertFieldError(t, admin.PUT("/api/config", storage), "storage.s3.secretKey", "required_if")
	})

	t.Run("rejects unknown keys", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		// The request type has no field for an unknown key, so the request is extended as a JSON object
		encoded, err := json.Marshal(newConfigRequest(h.Faker, h.Config))
		require.NoError(t, err)
		body := map[string]any{}
		require.NoError(t, json.Unmarshal(encoded, &body))
		app, ok := body["app"].(map[string]any)
		require.True(t, ok, "the app section is an object")
		app[h.Faker.LetterN(10)] = h.Faker.Word()

		h.asAdmin().PUT("/api/config", body).AssertBadRequest()
	})

	t.Run("reports a write error as a server error", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		// A directory in place of the config file cannot be replaced, regardless of the permissions of the user
		require.NoError(t, os.MkdirAll(h.ConfigPath, 0o755))

		h.asAdmin().PUT("/api/config", newConfigRequest(h.Faker, h.Config)).
			AssertStatus(http.StatusInternalServerError).
			AssertJSONPath("code", "server_error")
	})
}

func TestConfigStorageTestHandler(t *testing.T) {
	t.Parallel()

	t.Run("checks a local path", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		admin := h.asAdmin()

		writable := filepath.Join(t.TempDir(), h.Faker.LetterN(8))
		admin.POST("/api/config/storage/test", api.StorageSettings{
			Driver: string(media.DriverLocal),
			Path:   writable,
		}).
			AssertOK().
			AssertJSON(map[string]bool{
				"ok": true,
			})

		file := filepath.Join(t.TempDir(), h.Faker.LetterN(8))
		require.NoError(t, os.WriteFile(file, []byte(h.Faker.LoremIpsumSentence(6)), 0o600))
		admin.POST("/api/config/storage/test", api.StorageSettings{
			Driver: string(media.DriverLocal),
			Path:   file,
		}).
			AssertOK().
			AssertJSONPath("ok", false)
	})

	t.Run("reports an invalid s3 endpoint", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		settings := newS3StorageSettings(h.Faker)
		settings.S3.Endpoint = h.Faker.DomainName()

		h.asAdmin().POST("/api/config/storage/test", settings).AssertOK().AssertJSONPath("ok", false).AssertJSONCount("", 2)
	})

	t.Run("reports a missing bucket", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		// An S3 server that knows no bucket
		s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
		t.Cleanup(s3.Close)

		settings := newS3StorageSettings(h.Faker)
		settings.S3.Endpoint = s3.URL
		settings.S3.UsePathStyle = true

		h.asAdmin().POST("/api/config/storage/test", settings).AssertOK().AssertJSONPath("ok", false)
	})

	t.Run("rejects an unknown driver", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/config/storage/test", api.StorageSettings{
			Driver: h.Faker.LetterN(5),
		})
		assertFieldError(t, response, "driver", "oneof")
	})

	t.Run("rejects incomplete s3 settings", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/config/storage/test", api.StorageSettings{
			Driver: string(media.DriverS3),
		})
		assertFieldError(t, response, "s3.endpoint", "required_if")
	})
}

// hasJSONPath reports whether the decoded JSON object holds a value at the dot-separated path.
func hasJSONPath(object map[string]any, jsonPath string) bool {
	var current any = object
	for segment := range strings.SplitSeq(jsonPath, ".") {
		section, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = section[segment]
		if !ok {
			return false
		}
	}
	return true
}
