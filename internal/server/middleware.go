package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// RequestIDHeader is the header a request ID is read from and written to.
	RequestIDHeader = "X-Request-ID"
	// maxRequestIDLength limits an incoming request ID.
	maxRequestIDLength = 128
)

// requestIDKey is the context key of the request ID.
type requestIDKey struct{}

// RequestID takes the request ID from the X-Request-ID header, or creates one if the header is missing or invalid,
// stores it in the request context, and returns it in the response header. A proxy in front of the application
// can set the header, so its logs and the application's logs share the same ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}

		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFromContext returns the request ID stored by RequestID, or an empty string outside of it.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// validRequestID accepts printable ASCII without spaces, which covers UUIDs, ULIDs,
// and proxy formats, and keeps control characters out of headers and logs.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for i := range len(id) {
		if id[i] <= ' ' || id[i] > '~' {
			return false
		}
	}
	return true
}

// newRequestID returns 16 random bytes in hex.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// RealIP replaces RemoteAddr with the address of the client. Forwarding headers are only honored
// when the direct peer is one of the trusted proxies, so a client cannot spoof its address to bypass rate limits.
func RealIP(trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = clientIP(r, trustedProxies)
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP determines the originating client IP for a request. If the direct peer is not a trusted proxy,
// its address is used as-is and any forwarding headers are ignored. Otherwise, the X-Forwarded-For chain is walked
// from right to left, and the first untrusted address is returned as the client. Values that are no IP address
// are ignored in both forwarding headers.
func clientIP(r *http.Request, trusted []*net.IPNet) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		peer = host
	}

	parsedPeer := net.ParseIP(peer)
	if parsedPeer == nil || !ipInNetworks(parsedPeer, trusted) {
		return peer
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
		if net.ParseIP(realIP) == nil {
			return peer
		}
		return realIP
	}

	leftmost := ""
	for _, part := range slices.Backward(strings.Split(forwarded, ",")) {
		candidate := strings.TrimSpace(part)
		parsed := net.ParseIP(candidate)
		if parsed == nil {
			continue
		}
		if !ipInNetworks(parsed, trusted) {
			return candidate
		}
		leftmost = candidate
	}

	// Every valid hop is trusted: fall back to the left-most of them
	if leftmost != "" {
		return leftmost
	}
	return peer
}

// ipInNetworks reports whether ip is contained in any of the given networks.
func ipInNetworks(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// SecurityHeadersOptions holds the policies the SecurityHeaders middleware sends.
type SecurityHeadersOptions struct {
	// ContentSecurityPolicy is the value of the Content-Security-Policy header, empty leaves the header out.
	ContentSecurityPolicy string
	// PermissionsPolicy is the value of the Permissions-Policy header, empty leaves the header out.
	PermissionsPolicy string
	// WebSocketSources adds ws:// and wss:// of the request host to the connect-src directive of the policy.
	// Older browsers do not match websockets with 'self', so a same-origin websocket needs its origin listed.
	// A policy without connect-src stays as it is, since adding the directive would override default-src.
	WebSocketSources bool
}

// SecurityHeaders sets response headers that keep browsers from sniffing content types, embedding the response
// in frames, leaking full URLs to other origins, and sharing the browsing context or resources with other origins.
// The policies of the options are added when they are set.
func SecurityHeaders(options SecurityHeadersOptions) func(http.Handler) http.Handler {
	var directives []string
	connectIndex := -1
	for directive := range strings.SplitSeq(options.ContentSecurityPolicy, ";") {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		name, _, _ := strings.Cut(directive, " ")
		if strings.EqualFold(name, "connect-src") {
			connectIndex = len(directives)
		}
		directives = append(directives, directive)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("Cross-Origin-Opener-Policy", "same-origin")
			header.Set("Cross-Origin-Resource-Policy", "same-origin")
			header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			header.Set("X-Content-Type-Options", "nosniff")
			// Browsers that predate frame-ancestors in the policy still honor this header
			header.Set("X-Frame-Options", "DENY")

			if options.ContentSecurityPolicy != "" {
				policy := options.ContentSecurityPolicy
				if options.WebSocketSources && connectIndex >= 0 && validHost(r.Host) {
					policy = withWebSocketSources(directives, connectIndex, r.Host)
				}
				header.Set("Content-Security-Policy", policy)
			}
			if options.PermissionsPolicy != "" {
				header.Set("Permissions-Policy", options.PermissionsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validHost accepts a host name or IP address with an optional port, so a forged Host header
// cannot add directives or sources to the policy.
func validHost(host string) bool {
	if host == "" {
		return false
	}
	for i := range len(host) {
		c := host[i]
		isAlphanumeric := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlphanumeric && !strings.ContainsRune(".-:[]", rune(c)) {
			return false
		}
	}
	return true
}

// withWebSocketSources returns the policy with the websocket origins of host appended to the connect-src directive.
func withWebSocketSources(directives []string, connectIndex int, host string) string {
	extended := slices.Clone(directives)
	extended[connectIndex] += " ws://" + host + " wss://" + host
	return strings.Join(extended, "; ")
}

// Default CORS settings, used when CORSOptions leaves them empty.
var (
	defaultCORSMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	defaultCORSHeaders = []string{"Accept", "Authorization", "Content-Type", RequestIDHeader}
)

// defaultCORSMaxAge is how long browsers may cache a preflight answer.
const defaultCORSMaxAge = 10 * time.Minute

// CORSOptions holds the settings of the CORS middleware. Without allowed origins, CORS stays off.
type CORSOptions struct {
	// AllowedOrigins are exact origins such as "https://example.com", or "*" for any origin.
	// "*" cannot be combined with AllowCredentials, browsers reject that.
	AllowedOrigins []string
	// AllowCredentials lets browsers send cookies and authorization headers cross-origin.
	AllowCredentials bool
	// AllowedMethods defaults to the common methods of a JSON API.
	AllowedMethods []string
	// AllowedHeaders defaults to the headers a JSON API client sends, including the request ID.
	AllowedHeaders []string
	// ExposedHeaders are response headers scripts may read.
	ExposedHeaders []string
	// MaxAge defaults to 10 minutes.
	MaxAge time.Duration
}

// CORS answers preflight requests and adds the CORS headers to requests from allowed origins.
// Requests from other origins pass through without CORS headers, so the browser blocks reading the response.
func CORS(options CORSOptions) func(http.Handler) http.Handler {
	if len(options.AllowedOrigins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	anyOrigin := slices.Contains(options.AllowedOrigins, "*") && !options.AllowCredentials
	methods := strings.Join(orDefault(options.AllowedMethods, defaultCORSMethods), ", ")
	headers := strings.Join(orDefault(options.AllowedHeaders, defaultCORSHeaders), ", ")
	exposed := strings.Join(append([]string{RequestIDHeader, "RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset", "Retry-After"}, options.ExposedHeaders...), ", ")
	maxAge := options.MaxAge
	if maxAge <= 0 {
		maxAge = defaultCORSMaxAge
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")
			if !anyOrigin && !slices.Contains(options.AllowedOrigins, origin) {
				next.ServeHTTP(w, r)
				return
			}

			if anyOrigin {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(maxAge.Seconds())))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.Header().Set("Access-Control-Expose-Headers", exposed)
			next.ServeHTTP(w, r)
		})
	}
}

// orDefault returns values, or fallback if values is empty.
func orDefault(values, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}

// CrossOriginOptions holds the settings of the CrossOrigin middleware.
type CrossOriginOptions struct {
	// TrustedOrigins may send state-changing requests from another origin, such as "https://example.com".
	// "*" and malformed origins are never trusted, a wildcard only lets other origins read responses.
	TrustedOrigins []string
	// Deny answers a rejected request and defaults to a plain 403 Forbidden.
	Deny http.Handler
}

// CrossOrigin rejects state-changing requests that the browser marks as cross-origin in the Sec-Fetch-Site
// or Origin header, unless they come from a trusted origin. GET, HEAD, and OPTIONS pass, as do requests without
// these headers, such as those of command-line tools, which cannot carry the session of a browser.
func CrossOrigin(options CrossOriginOptions) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range options.TrustedOrigins {
		// A rejected origin such as "*" stays untrusted, so it never exempts a request
		_ = protection.AddTrustedOrigin(origin)
	}
	if options.Deny != nil {
		protection.SetDenyHandler(options.Deny)
	}

	return protection.Handler
}
