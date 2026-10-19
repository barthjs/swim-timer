package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
)

func TestWebhookStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhooks sorted by name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewWebhookStore(newTestStorage(t))
		second := testWebhook(t, "b")
		first := testWebhook(t, "A")
		require.NoError(t, st.Save(ctx, second.ID, second))
		require.NoError(t, st.Save(ctx, first.ID, first))

		webhooks, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Webhook{
			first,
			second,
		}, webhooks, "the names are compared without letter case")
	})

	t.Run("returns no webhooks without the folder", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		webhooks, err := st.List(t.Context())

		require.NoError(t, err)
		assert.Empty(t, webhooks)
	})

	t.Run("skips other files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		st := NewWebhookStore(storage)
		webhook := testWebhook(t, "valid")
		require.NoError(t, st.Save(ctx, webhook.ID, webhook))
		writeFile(t, filepath.Join(root, webhooksFolder, "notes.txt"), "")
		writeFile(t, filepath.Join(root, webhooksFolder, "nested", "deep.json"), "{}")
		writeFile(t, filepath.Join(root, "other.json"), "{}")

		webhooks, err := st.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Webhook{
			webhook,
		}, webhooks)
	})

	t.Run("reports a broken file with the webhooks it could read", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		st := NewWebhookStore(storage)
		webhook := testWebhook(t, "valid")
		require.NoError(t, st.Save(ctx, webhook.ID, webhook))
		writeFile(t, filepath.Join(root, webhooksFolder, "broken.json"), "{not json")

		webhooks, err := st.List(ctx)

		assert.ErrorIs(t, err, ErrUnreadableFile)
		assert.Equal(t, []domain.Webhook{
			webhook,
		}, webhooks)
	})
}

func TestWebhookStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the webhook", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewWebhookStore(newTestStorage(t))
		want := testWebhook(t, apitest.Faker(t).Name())
		require.NoError(t, st.Save(ctx, want.ID, want))

		got, err := st.Get(ctx, want.ID)

		require.NoError(t, err)
		assert.Equal(t, want, *got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		for _, id := range invalidFileNames(webhookExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				_, err := st.Get(t.Context(), id)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		_, err := st.Get(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeFile(t, filepath.Join(root, webhooksFolder, "broken.json"), "{not json")

		_, err := NewWebhookStore(storage).Get(t.Context(), "broken")

		require.Error(t, err)
		assert.NotErrorIs(t, err, apperr.ErrNotFound, "a broken file is a decoding error, not a missing webhook")
	})
}

func TestWebhookStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the webhook", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewWebhookStore(newTestStorage(t))
		want := testWebhook(t, apitest.Faker(t).Name())

		err := st.Save(ctx, want.ID, want)

		require.NoError(t, err)
		got, err := st.Get(ctx, want.ID)
		require.NoError(t, err)
		assert.Equal(t, want, *got)
	})

	t.Run("replaces the webhook", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewWebhookStore(newTestStorage(t))
		first := testWebhook(t, "a")
		require.NoError(t, st.Save(ctx, first.ID, first))
		second := first
		second.URL = apitest.Faker(t).URL()
		second.Enabled = false

		err := st.Save(ctx, first.ID, second)

		require.NoError(t, err)
		got, err := st.Get(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, second, *got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		for _, id := range invalidFileNames(webhookExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := st.Save(t.Context(), id, testWebhook(t, id))

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})
}

func TestWebhookStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the webhook", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewWebhookStore(newTestStorage(t))
		webhook := testWebhook(t, apitest.Faker(t).Name())
		require.NoError(t, st.Save(ctx, webhook.ID, webhook))

		err := st.Delete(ctx, webhook.ID)

		require.NoError(t, err)
		_, err = st.Get(ctx, webhook.ID)
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		for _, id := range invalidFileNames(webhookExtension) {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := st.Delete(t.Context(), id)

				assert.ErrorIs(t, err, ErrInvalidName)
			})
		}
	})

	t.Run("returns not found for an unknown webhook", func(t *testing.T) {
		t.Parallel()

		st := NewWebhookStore(newTestStorage(t))

		err := st.Delete(t.Context(), "missing")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

// testWebhook returns an enabled webhook under the given name with a random id, url, and secret.
func testWebhook(t *testing.T, name string) domain.Webhook {
	t.Helper()

	f := apitest.Faker(t)
	return domain.Webhook{
		ID:   f.LetterN(26),
		Name: name,
		URL:  f.URL(),
		Events: []domain.WebhookEvent{
			domain.WebhookEventHeatUpdated,
		},
		Enabled: true,
		Secret:  "whsec_" + f.LetterN(32),
	}
}
