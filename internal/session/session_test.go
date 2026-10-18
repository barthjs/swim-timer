package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/session"
)

func TestWithLifetime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		lifetime     time.Duration
		wantLifetime time.Duration
	}{
		{
			name:         "sets the lifetime",
			lifetime:     24 * time.Hour,
			wantLifetime: 24 * time.Hour,
		},
		{
			name:         "ignores a non-positive lifetime",
			lifetime:     0,
			wantLifetime: session.DefaultLifetime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sm, err := session.New(false, session.WithLifetime(tt.lifetime))

			require.NoError(t, err)
			assert.Equal(t, tt.wantLifetime, sm.Lifetime)
		})
	}
}

func TestWithIdleTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		timeout  time.Duration
		wantIdle time.Duration
	}{
		{
			name:     "sets the idle timeout",
			timeout:  time.Hour,
			wantIdle: time.Hour,
		},
		{
			name:     "ignores a non-positive idle timeout",
			timeout:  -time.Minute,
			wantIdle: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sm, err := session.New(false, session.WithIdleTimeout(tt.timeout))

			require.NoError(t, err)
			assert.Equal(t, tt.wantIdle, sm.IdleTimeout)
		})
	}
}

func TestWithRememberMe(t *testing.T) {
	t.Parallel()

	t.Run("keeps the cookie of a remembered login", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false, session.WithRememberMe(), session.WithLifetime(48*time.Hour))
		require.NoError(t, err)

		cookie := loginCookie(t, sm, true)

		assert.WithinDuration(t, time.Now().Add(48*time.Hour), cookie.Expires, time.Minute)
	})

	t.Run("ends a login that is not remembered with the browser session", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false, session.WithRememberMe())
		require.NoError(t, err)

		cookie := loginCookie(t, sm, false)

		assert.True(t, cookie.Expires.IsZero(), "no Expires attribute makes it a browser session cookie")
		assert.Zero(t, cookie.MaxAge)
	})

	t.Run("keeps every cookie for the lifetime without the option", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false)
		require.NoError(t, err)

		cookie := loginCookie(t, sm, false)

		assert.False(t, cookie.Expires.IsZero())
	})
}

// loginCookie logs in on a fresh session with the given remember choice and returns the session cookie.
func loginCookie(t *testing.T, sm *scs.SessionManager, remember bool) *http.Cookie {
	t.Helper()

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, sm.RenewToken(r.Context()))
		sm.Put(r.Context(), "user", "id")
		session.Remember(r.Context(), sm, remember)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/login", nil))
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	return cookies[0]
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("configures the cookie", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			secure bool
		}{
			{
				name:   "configures the cookie for development",
				secure: false,
			},
			{
				name:   "configures the cookie for production",
				secure: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				sm, err := session.New(tt.secure)

				require.NoError(t, err)
				assert.Equal(t, 168*time.Hour, sm.Lifetime)
				assert.True(t, sm.HashTokenInStore)
				assert.True(t, sm.Cookie.HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, sm.Cookie.SameSite)
				assert.Equal(t, tt.secure, sm.Cookie.Secure)
			})
		}
	})

	t.Run("uses the default lifetime without idle timeout", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false)

		require.NoError(t, err)
		assert.Equal(t, session.DefaultLifetime, sm.Lifetime)
		assert.Equal(t, time.Duration(0), sm.IdleTimeout)
	})

	t.Run("keeps memory sessions between requests", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false)

		require.NoError(t, err)
		roundTrip(t, sm)
	})
}

// roundTrip stores a value in the session on one request and reads it back on the next one with the returned cookie,
// so it checks that the store keeps sessions between requests.
func roundTrip(t *testing.T, sm *scs.SessionManager) {
	t.Helper()

	value := apitest.Faker(t).Word()
	mux := http.NewServeMux()
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) { sm.Put(r.Context(), "name", value) })
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sm.GetString(r.Context(), "name")))
	})
	handler := sm.LoadAndSave(mux)

	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/put", nil))
	cookies := put.Result().Cookies()
	require.Len(t, cookies, 1)

	get := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/get", nil)
	get.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, get)

	assert.Equal(t, value, rec.Body.String())
}

func TestStopCleanup(t *testing.T) {
	t.Parallel()

	t.Run("returns on repeated calls", func(t *testing.T) {
		t.Parallel()

		sm, err := session.New(false)
		require.NoError(t, err)
		stopped := make(chan struct{})

		go func() {
			session.StopCleanup(sm)
			session.StopCleanup(sm)
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "StopCleanup blocks")
		}
	})
}
