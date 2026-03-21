package handlers

import (
	"io/fs"
	"net/http"
	"strings"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"

	"github.com/barthjs/swim-timer/web"
)

// Routes returns the http.Handler for the application.
func (h *Handlers) Routes() http.Handler {
	router := httprouter.New()

	staticFS, _ := fs.Sub(web.Files, "dist")
	httpFS := http.FileServerFS(staticFS)

	common := alice.New(
		h.logRequest,
		gorillaHandlers.CompressHandler,
	)

	auth := common.Append(h.requireAuth)

	assetChain := alice.New(
		gorillaHandlers.CompressHandler,
		h.cacheHeaderMiddleware,
	).Then(httpFS)

	rootAssetChain := alice.New(
		gorillaHandlers.CompressHandler,
	).Then(httpFS)

	router.Handler(http.MethodGet, "/", common.ThenFunc(h.indexHandler))
	router.Handler(http.MethodGet, "/assets/*filepath", assetChain)

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.notFoundResponse(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if stat, err := fs.Stat(staticFS, cleanPath); err == nil && !stat.IsDir() {
			rootAssetChain.ServeHTTP(w, r)
			return
		}

		common.ThenFunc(h.indexHandler).ServeHTTP(w, r)
	})

	router.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.methodNotAllowedResponse(w, r)
	})

	router.HandlerFunc(http.MethodGet, "/api/ping", h.pingHandler)

	router.Handler(http.MethodPost, "/api/auth/login", common.ThenFunc(h.loginHandler))

	router.Handler(http.MethodGet, "/api/config", auth.ThenFunc(h.showConfigHandler))
	router.Handler(http.MethodPut, "/api/config", auth.ThenFunc(h.updateConfigHandler))

	router.HandlerFunc(http.MethodGet, "/ws", h.webSocketHandler)

	return alice.New(
		h.recoverPanic,
		gorillaHandlers.ProxyHeaders,
		h.setRequestIP,
		h.commonHeaders,
	).Then(router)
}
