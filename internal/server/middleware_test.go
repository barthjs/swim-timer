package server_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/server"
)

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		incoming string
		keep     bool
	}{
		{
			name:     "keeps a valid id",
			incoming: "proxy-1234",
			keep:     true,
		},
		{
			name:     "keeps an id at the limit",
			incoming: strings.Repeat("a", 128),
			keep:     true,
		},
		{
			name:     "creates a missing id",
			incoming: "",
		},
		{
			name:     "replaces an id with spaces",
			incoming: "a b",
		},
		{
			name:     "replaces an id with control characters",
			incoming: "a\x01b",
		},
		{
			name:     "replaces a too long id",
			incoming: strings.Repeat("a", 129),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seen string
			handler := server.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = server.RequestIDFromContext(r.Context())
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(server.RequestIDHeader, tt.incoming)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			returned := rec.Header().Get(server.RequestIDHeader)
			assert.Equal(t, seen, returned)
			if tt.keep {
				assert.Equal(t, tt.incoming, returned)
				return
			}
			assert.Len(t, returned, 32)
			assert.NotEqual(t, tt.incoming, returned)
		})
	}
}

func TestRequestIDFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nothing without the middleware", func(t *testing.T) {
		t.Parallel()

		id := server.RequestIDFromContext(t.Context())

		assert.Empty(t, id)
	})
}

func TestRealIP(t *testing.T) {
	t.Parallel()

	_, proxies, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	trusted := []*net.IPNet{proxies}

	tests := []struct {
		name      string
		peer      string
		forwarded string
		realIP    string
		want      string
	}{
		{
			name: "keeps a trusted peer without headers",
			peer: "10.0.0.1:1234",
			want: "10.0.0.1",
		},
		{
			name:   "uses the X-Real-IP of a trusted peer",
			peer:   "10.0.0.1:1234",
			realIP: "198.51.100.1",
			want:   "198.51.100.1",
		},
		{
			name:   "uses an IPv6 X-Real-IP of a trusted peer",
			peer:   "10.0.0.1:1234",
			realIP: " 2001:db8::1 ",
			want:   "2001:db8::1",
		},
		{
			name:      "uses the first untrusted hop from the right",
			peer:      "10.0.0.1:1234",
			forwarded: "198.51.100.1, 203.0.113.9, 10.0.0.2",
			want:      "203.0.113.9",
		},
		{
			name:      "skips invalid hops",
			peer:      "10.0.0.1:1234",
			forwarded: "198.51.100.1, garbage",
			want:      "198.51.100.1",
		},
		{
			name:      "falls back to the left-most hop when all hops are trusted",
			peer:      "10.0.0.1:1234",
			forwarded: "10.0.0.3, 10.0.0.2",
			want:      "10.0.0.3",
		},
		{
			name:      "skips an invalid left-most hop when all hops are trusted",
			peer:      "10.0.0.1:1234",
			forwarded: "garbage, 10.0.0.3, 10.0.0.2",
			want:      "10.0.0.3",
		},
		{
			name: "accepts a peer without port",
			peer: "203.0.113.5",
			want: "203.0.113.5",
		},
		{
			name:      "keeps the peer when all hops are invalid",
			peer:      "10.0.0.1:1234",
			forwarded: "garbage",
			want:      "10.0.0.1",
		},
		{
			name:   "ignores an X-Real-IP that is no address",
			peer:   "10.0.0.1:1234",
			realIP: "not-an-ip",
			want:   "10.0.0.1",
		},
		{
			name:      "ignores the forwarding headers of an untrusted peer",
			peer:      "203.0.113.5:1234",
			forwarded: "198.51.100.1",
			want:      "203.0.113.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seen string
			handler := server.RealIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = r.RemoteAddr
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			req.RemoteAddr = tt.peer
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tt.want, seen)
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	options := server.SecurityHeadersOptions{
		ContentSecurityPolicy: "default-src 'self'; connect-src 'self'",
		PermissionsPolicy:     "camera=()",
	}

	t.Run("sets the security headers", func(t *testing.T) {
		t.Parallel()

		rec := securityHeadersRequest(t, options, "app.test")

		assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Opener-Policy"))
		assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Resource-Policy"))
		assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		assert.Equal(t, options.ContentSecurityPolicy, rec.Header().Get("Content-Security-Policy"))
		assert.Equal(t, options.PermissionsPolicy, rec.Header().Get("Permissions-Policy"))
	})

	t.Run("leaves out empty policies", func(t *testing.T) {
		t.Parallel()

		rec := securityHeadersRequest(t, server.SecurityHeadersOptions{}, "app.test")

		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.NotContains(t, rec.Header(), "Content-Security-Policy")
		assert.NotContains(t, rec.Header(), "Permissions-Policy")
	})

	t.Run("adds the websocket origins of the request host", func(t *testing.T) {
		t.Parallel()
		withSources := options
		withSources.ContentSecurityPolicy = "default-src 'self'; connect-src 'self'; frame-ancestors 'none';"
		withSources.WebSocketSources = true

		rec := securityHeadersRequest(t, withSources, "app.test:8080")

		assert.Equal(t, "default-src 'self'; connect-src 'self' ws://app.test:8080 wss://app.test:8080; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
	})

	t.Run("adds the websocket origins of an ip address", func(t *testing.T) {
		t.Parallel()
		withSources := options
		withSources.WebSocketSources = true

		rec := securityHeadersRequest(t, withSources, "[2001:db8::1]:8080")

		assert.Equal(t, "default-src 'self'; connect-src 'self' ws://[2001:db8::1]:8080 wss://[2001:db8::1]:8080", rec.Header().Get("Content-Security-Policy"))
	})

	t.Run("keeps the policy for an invalid host", func(t *testing.T) {
		t.Parallel()
		withSources := options
		withSources.WebSocketSources = true

		rec := securityHeadersRequest(t, withSources, "app.test;script-src")

		assert.Equal(t, options.ContentSecurityPolicy, rec.Header().Get("Content-Security-Policy"))
	})

	t.Run("keeps a policy without connect-src", func(t *testing.T) {
		t.Parallel()
		withoutConnect := server.SecurityHeadersOptions{
			ContentSecurityPolicy: "default-src 'self'",
			WebSocketSources:      true,
		}

		rec := securityHeadersRequest(t, withoutConnect, "app.test")

		assert.Equal(t, withoutConnect.ContentSecurityPolicy, rec.Header().Get("Content-Security-Policy"))
	})
}

// securityHeadersRequest sends a request for host through the SecurityHeaders middleware and records the response.
func securityHeadersRequest(t *testing.T, options server.SecurityHeadersOptions, host string) *httptest.ResponseRecorder {
	t.Helper()

	handler := server.SecurityHeaders(options)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Host = host
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	return rec
}

func TestCORS(t *testing.T) {
	t.Parallel()

	allowed := server.CORSOptions{
		AllowedOrigins:   []string{"https://app.test"},
		AllowCredentials: true,
		MaxAge:           time.Minute,
	}

	t.Run("sets the headers for an allowed origin", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, allowed, http.MethodGet, "https://app.test", false)

		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Equal(t, "https://app.test", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, rec.Header().Get("Access-Control-Expose-Headers"), "Retry-After")
		assert.Contains(t, rec.Header().Values("Vary"), "Origin")
	})

	t.Run("answers a preflight", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, allowed, http.MethodOptions, "https://app.test", true)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), http.MethodPut)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "60", rec.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("passes a same-origin request through", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, allowed, http.MethodGet, "", false)

		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("passes an options request without preflight to the handler", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, allowed, http.MethodOptions, "https://app.test", false)

		assert.Equal(t, http.StatusTeapot, rec.Code)
	})

	t.Run("allows any origin without credentials for a wildcard", func(t *testing.T) {
		t.Parallel()

		options := server.CORSOptions{
			AllowedOrigins: []string{"*"},
		}

		rec := corsRequest(t, options, http.MethodGet, "https://any.test", false)

		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("allows only listed origins for a wildcard with credentials", func(t *testing.T) {
		t.Parallel()

		options := server.CORSOptions{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
		}

		rec := corsRequest(t, options, http.MethodGet, "https://any.test", false)

		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("replaces the defaults with custom methods and headers", func(t *testing.T) {
		t.Parallel()

		options := server.CORSOptions{
			AllowedOrigins: []string{"https://app.test"},
			AllowedMethods: []string{"GET"},
			AllowedHeaders: []string{"X-Custom"},
		}

		rec := corsRequest(t, options, http.MethodOptions, "https://app.test", true)

		assert.Equal(t, "GET", rec.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "X-Custom", rec.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("sets no headers for another origin", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, allowed, http.MethodGet, "https://evil.test", false)

		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("sets no headers without allowed origins", func(t *testing.T) {
		t.Parallel()

		rec := corsRequest(t, server.CORSOptions{}, http.MethodGet, "https://other.test", false)

		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

// corsRequest sends a request with the origin through the CORS middleware and returns the response.
func corsRequest(t *testing.T, options server.CORSOptions, method, origin string, preflight bool) *httptest.ResponseRecorder {
	t.Helper()

	handler := server.CORS(options)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	req := httptest.NewRequestWithContext(t.Context(), method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCrossOrigin(t *testing.T) {
	t.Parallel()

	trusted := "https://trusted.example"

	tests := []struct {
		name     string
		method   string
		headers  map[string]string
		wantPass bool
	}{
		{
			name:     "passes a safe method from another site",
			method:   http.MethodGet,
			headers:  map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"},
			wantPass: true,
		},
		{
			name:     "passes a request without browser headers",
			method:   http.MethodPost,
			wantPass: true,
		},
		{
			name:     "passes a same-origin request",
			method:   http.MethodPost,
			headers:  map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://example.com"},
			wantPass: true,
		},
		{
			name:     "passes an origin matching the host without Sec-Fetch-Site",
			method:   http.MethodPost,
			headers:  map[string]string{"Origin": "http://example.com"},
			wantPass: true,
		},
		{
			name:     "passes a trusted origin",
			method:   http.MethodPost,
			headers:  map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": trusted},
			wantPass: true,
		},
		{
			name:    "rejects a cross-site request",
			method:  http.MethodPost,
			headers: map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"},
		},
		{
			name:    "rejects a foreign origin without Sec-Fetch-Site",
			method:  http.MethodDelete,
			headers: map[string]string{"Origin": "https://evil.example"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := server.CrossOrigin(server.CrossOriginOptions{
				TrustedOrigins: []string{"*", trusted},
				Deny:           http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }),
			})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.wantPass {
				assert.Equal(t, http.StatusNoContent, rec.Code)
				return
			}
			assert.Equal(t, http.StatusTeapot, rec.Code, "the deny handler answers")
		})
	}

	t.Run("rejects as forbidden without a deny handler", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		handler := server.CrossOrigin(server.CrossOriginOptions{})(next)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
