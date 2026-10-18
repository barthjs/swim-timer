package service

import (
	"context"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestAuthService_Login(t *testing.T) {
	t.Parallel()

	f := apitest.Faker(t)
	password := f.Password(true, true, true, false, false, 16)
	wrong := f.Password(true, true, true, false, false, 20)

	tests := []struct {
		name      string
		password  string
		candidate string
		want      bool
	}{
		{
			name:      "accepts the matching password",
			password:  password,
			candidate: password,
			want:      true,
		},
		{
			name:      "accepts an empty candidate without a password",
			password:  "",
			candidate: "",
			want:      true,
		},
		{
			name:      "rejects a wrong password",
			password:  password,
			candidate: wrong,
			want:      false,
		},
		{
			name:      "rejects any other candidate without a password",
			password:  "",
			candidate: password,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sm := newTestSessionManager()
			svc := NewAuthService(AuthOptions{
				Password: tt.password,
			}, sm)
			ctx := sessionContext(t, sm)

			ok, err := svc.Login(ctx, tt.candidate)

			require.NoError(t, err)
			assert.Equal(t, tt.want, ok)
			assert.Equal(t, tt.want, svc.Authenticated(ctx))
		})
	}

	t.Run("renews the session token", func(t *testing.T) {
		t.Parallel()

		sm := newTestSessionManager()
		svc := NewAuthService(AuthOptions{
			Password: password,
		}, sm)
		ctx := sessionContext(t, sm)
		sm.Put(ctx, "before", true)
		before, _, err := sm.Commit(ctx)
		require.NoError(t, err)

		_, err = svc.Login(ctx, password)

		require.NoError(t, err)
		after, _, err := sm.Commit(ctx)
		require.NoError(t, err)
		assert.NotEqual(t, before, after, "a new token prevents session fixation")
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Parallel()

	t.Run("ends the session", func(t *testing.T) {
		t.Parallel()

		password := apitest.Faker(t).Password(true, true, true, false, false, 16)
		sm := newTestSessionManager()
		svc := NewAuthService(AuthOptions{
			Password: password,
		}, sm)
		ctx := sessionContext(t, sm)
		_, err := svc.Login(ctx, password)
		require.NoError(t, err)

		err = svc.Logout(ctx)

		require.NoError(t, err)
		assert.False(t, svc.Authenticated(ctx))
	})
}

// newTestSessionManager returns a session manager with an in-memory store.
func newTestSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	return sm
}

// sessionContext returns a context carrying a fresh session of sm, as the session middleware would.
func sessionContext(t *testing.T, sm *scs.SessionManager) context.Context {
	t.Helper()

	ctx, err := sm.Load(t.Context(), "")
	require.NoError(t, err)
	return ctx
}
