package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("forgets the expiry", func(t *testing.T) {
		t.Parallel()

		token := apitest.Faker(t).UUID()
		store := newMemoryStore(time.Hour)
		t.Cleanup(store.stopCleanup)
		require.NoError(t, store.Commit(token, []byte("data"), time.Now().Add(time.Hour)))

		require.NoError(t, store.Delete(token))

		_, found, err := store.Find(token)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, store.expiries)
	})
}

func TestMemoryStore_stopCleanup(t *testing.T) {
	t.Parallel()

	t.Run("stops the cleanup", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore(time.Millisecond)

		stopped := make(chan struct{})
		go func() {
			store.stopCleanup()
			store.stopCleanup()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "stopping the cleanup twice blocks")
		}
		select {
		case <-store.done:
		default:
			assert.Fail(t, "the cleanup goroutine still runs after stopCleanup")
		}
	})
}

func TestMemoryStore_deleteExpired(t *testing.T) {
	t.Parallel()

	t.Run("removes the expired sessions", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		expired := faker.UUID()
		active := faker.UUID()
		now := time.Now()
		store := newMemoryStore(time.Hour)
		t.Cleanup(store.stopCleanup)
		require.NoError(t, store.Commit(expired, []byte(faker.Word()), now.Add(-time.Minute)))
		require.NoError(t, store.Commit(active, []byte(faker.Word()), now.Add(time.Hour)))

		store.deleteExpired(now)

		store.mu.Lock()
		defer store.mu.Unlock()
		assert.NotContains(t, store.expiries, expired)
		assert.Contains(t, store.expiries, active)
		all, err := store.All()
		require.NoError(t, err)
		assert.Contains(t, all, active, "an active session stays")
	})
}
