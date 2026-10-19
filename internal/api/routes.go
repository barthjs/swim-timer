package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	gorillaHandlers "github.com/gorilla/handlers"

	apidocs "github.com/barthjs/swim-timer/docs/api"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/server"
	"github.com/barthjs/swim-timer/web"
)

// contentSecurityPolicy only allows the resources of the application itself. Styles may be inline, since Nuxt UI
// injects style elements at runtime. Images and fonts may be data: and blob: URLs for the fonts inlined
// in the stylesheets and the images the editor shows from local files.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; " +
	"form-action 'self'; frame-ancestors 'none'"

// permissionsPolicy denies the browser features the application does not use.
const permissionsPolicy = "accelerometer=(), camera=(), display-capture=(), geolocation=(), gyroscope=(), hid=(), " +
	"magnetometer=(), microphone=(), midi=(), payment=(), serial=(), usb=()"

// Routes returns the HTTP router for the API.
func (a *API) Routes() http.Handler {
	r := chi.NewRouter()

	// The request ID comes first, so every later log line and error response carries it.
	// The body limit is the default for every route, uploads raise it for themselves.
	r.Use(
		server.RequestID,
		a.recoverPanic,
		server.RealIP(a.options.TrustedProxies),
		// The overlays run in the Chromium of streaming software, which may predate 'self' matching
		// websockets, so the websocket origin of the request host is listed in the policy as well.
		server.SecurityHeaders(server.SecurityHeadersOptions{
			ContentSecurityPolicy: contentSecurityPolicy,
			PermissionsPolicy:     permissionsPolicy,
			WebSocketSources:      true,
		}),
		server.CORS(server.CORSOptions{
			AllowedOrigins:   a.options.CORSAllowedOrigins,
			AllowCredentials: a.options.CORSAllowCredentials,
		}),
		server.BodyLimit(server.DefaultMaxBodySize),
	)

	r.Get("/healthz", a.health.Live)
	r.Get("/readyz", a.health.Ready)

	a.realtimeRoutes(r)
	a.apiRoutes(r)
	a.staticRoutes(r)

	return r
}

// realtimeRoutes registers the WebSocket endpoint at "/api/ws". It stays outside the "/api" group, whose timeout
// and compression do not fit a connection that stays open. It is public like the overlays it feeds.
func (a *API) realtimeRoutes(r chi.Router) {
	r.With(a.logRequest).Get("/api/ws", a.realtimeConnectHandler)
}

// apiRoutes registers the JSON API under "/api".
func (a *API) apiRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		// Every route of the JSON API is compressed and logged. State-changing requests from another origin
		// are rejected unless CORS allows that origin explicitly.
		r.Use(
			server.Timeout(server.DefaultRequestTimeout),
			gorillaHandlers.CompressHandler,
			a.logRequest,
			server.CrossOrigin(server.CrossOriginOptions{
				TrustedOrigins: a.options.CORSAllowedOrigins,
				Deny:           http.HandlerFunc(a.forbiddenResponse),
			}),
		)

		r.Group(func(r chi.Router) {
			r.NotFound(a.notFoundResponse)
			r.MethodNotAllowed(a.methodNotAllowedResponse)

			if a.options.Docs {
				r.Get("/docs", http.RedirectHandler("/api/docs/", http.StatusMovedPermanently).ServeHTTP)
				r.Handle("/docs/*", http.StripPrefix("/api/docs", apidocs.Handler(a.options.Name, a.health.Version())))
			}

			r.Get("/meta", a.metaGetHandler)
			r.Get("/ping", a.pingGetHandler)
		})

		a.authRoutes(r)
		a.configRoutes(r)
		a.liveRoutes(r)
		a.meetRoutes(r)
		a.presetRoutes(r)
		a.recordingRoutes(r)
		a.timingRoutes(r)
		a.updateRoutes(r)
		a.webhookRoutes(r)
	})
}

// authRoutes registers the auth routes under "/auth".
func (a *API) authRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave)

		r.Get("/me", a.authStatusGetHandler)
		r.With(a.requireGuest).Post("/login", a.authLoginHandler)
		r.Post("/logout", a.authLogoutHandler)
	})
}

// configRoutes registers the admin config routes under "/config".
func (a *API) configRoutes(r chi.Router) {
	r.Route("/config", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/", a.configGetHandler)
		r.Put("/", a.configSaveHandler)

		r.Post("/storage/test", a.configStorageTestHandler)
	})
}

// liveRoutes registers the public route of the live heat under "/live".
func (a *API) liveRoutes(r chi.Router) {
	r.Route("/live", func(r chi.Router) {
		r.Get("/", a.liveHeatGetHandler)
	})
}

// meetRoutes registers the meet routes under "/meet". The meet data is public, like the overlays,
// so other programs can read it, while the files it is read from can only be managed with an admin session.
func (a *API) meetRoutes(r chi.Router) {
	r.Route("/meet", func(r chi.Router) {
		r.Get("/", a.meetOverviewGetHandler)
		r.Get("/events", a.meetEventListHandler)
		r.Get("/events/{number}", a.meetEventGetHandler)
		r.Get("/events/{number}/heats/{heat}", a.meetHeatGetHandler)
		r.Get("/competitors", a.meetCompetitorListHandler)
		r.Get("/start-entries", a.meetStartEntryListHandler)

		r.Group(func(r chi.Router) {
			r.Use(a.sm.LoadAndSave, a.requireAuthentication)

			r.Get("/files", a.meetFileListHandler)
			r.Group(func(r chi.Router) {
				// One request may carry every meet file at its maximum size,
				// since the upload of several files bounds each file on its own.
				r.Use(server.BodyLimit(uploadBodyLimit(maxMeetUploadMB*len(domain.MeetFiles()))), server.Timeout(uploadTimeout))

				r.Put("/files/{file}", a.meetFileSaveHandler)
				r.Post("/files", a.meetFileUploadHandler)
			})

			r.With(a.rateLimitPerIP("meet:reload", meetReloadRateLimit, meetReloadRateLimitWindow)).Post("/reload", a.meetReloadHandler)
		})
	})
}

// presetRoutes registers the AR overlay preset routes under "/presets".
// Reading is public, since the AR overlay loads its preset without a session.
func (a *API) presetRoutes(r chi.Router) {
	r.Route("/presets", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave)

		r.Get("/", a.presetListHandler)
		r.Get("/{name}", a.presetGetHandler)

		r.Group(func(r chi.Router) {
			r.Use(a.requireAuthentication)

			r.Put("/{name}", a.presetSaveHandler)
			r.Delete("/{name}", a.presetDeleteHandler)
		})
	})
}

// recordingRoutes registers the admin routes of the recordings of the timing data under "/recordings".
func (a *API) recordingRoutes(r chi.Router) {
	r.Route("/recordings", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/", a.recordingListHandler)
		r.With(server.BodyLimit(uploadBodyLimit(maxRecordingUploadMB)), server.Timeout(uploadTimeout)).Post("/", a.recordingCreateHandler)
		r.Delete("/{name}", a.recordingDeleteHandler)
		r.With(server.Timeout(downloadTimeout)).Get("/{name}", a.recordingDownloadHandler)
		r.With(server.Timeout(downloadTimeout)).Get("/{name}/csv", a.recordingExportHandler)
	})
}

// timingRoutes registers the public timing device routes under "/timing".
func (a *API) timingRoutes(r chi.Router) {
	r.Route("/timing", func(r chi.Router) {
		r.Get("/status", a.timingStatusGetHandler)
	})
}

// updateRoutes registers the admin routes of the self-update under "/update".
func (a *API) updateRoutes(r chi.Router) {
	r.Route("/update", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/status", a.updateStatusGetHandler)
		r.Get("/", a.updateCheckHandler)
		r.Post("/", a.updateStartHandler)
	})
}

// webhookRoutes registers the webhook routes under "/webhooks", which require an admin session.
func (a *API) webhookRoutes(r chi.Router) {
	r.Route("/webhooks", func(r chi.Router) {
		r.Use(a.sm.LoadAndSave, a.requireAuthentication)

		r.Get("/", a.webhookListHandler)
		r.Get("/{id}", a.webhookGetHandler)
		r.Post("/", a.webhookCreateHandler)
		r.Put("/{id}", a.webhookUpdateHandler)
		r.Delete("/{id}", a.webhookDeleteHandler)
		r.With(a.rateLimitPerIP("webhook:test", webhookTestRateLimit, webhookTestRateLimitWindow)).Post("/{id}/test", a.webhookTestHandler)

		r.Get("/{id}/deliveries", a.webhookDeliveryListHandler)

		r.Get("/{id}/preview", a.webhookPreviewGetHandler)
	})
}

// staticRoutes serves the embedded SPA.
func (a *API) staticRoutes(r chi.Router) {
	staticFS, _ := fs.Sub(web.Files, "dist")
	fileServer := http.FileServerFS(staticFS)

	r.With(gorillaHandlers.CompressHandler, a.cacheHeaderMiddleware).Handle("/assets/*", fileServer)
	r.With(gorillaHandlers.CompressHandler).Get("/", a.indexGetHandler)
	r.With(gorillaHandlers.CompressHandler).Get("/overlays/*", a.overlayIndexGetHandler)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		cleanPath := strings.TrimPrefix(req.URL.Path, "/")
		if stat, err := fs.Stat(staticFS, cleanPath); err == nil && !stat.IsDir() {
			gorillaHandlers.CompressHandler(fileServer).ServeHTTP(w, req)
			return
		}
		gorillaHandlers.CompressHandler(http.HandlerFunc(a.indexGetHandler)).ServeHTTP(w, req)
	})
}

// indexGetHandler serves the index.html of the SPA.
func (a *API) indexGetHandler(w http.ResponseWriter, _ *http.Request) {
	rawFile, _ := web.Files.ReadFile("dist/index.html")
	_, _ = w.Write(rawFile)
}

// overlayIndexGetHandler serves the overlay.html of the separate overlay app,
// which browser sources of streaming software load without the admin pages.
func (a *API) overlayIndexGetHandler(w http.ResponseWriter, _ *http.Request) {
	rawFile, _ := web.Files.ReadFile("dist/overlay.html")
	_, _ = w.Write(rawFile)
}
