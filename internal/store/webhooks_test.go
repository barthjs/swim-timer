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

	t.Run("rejects a broken file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeFile(t, filepath.Join(root, webhooksFolder, "broken.json"), "{not json")

		_, err := NewWebhookStore(storage).Get(t.Context(), "broken")

		require.Error(t, err)
		assert.NotErrorIs(t, err, apperr.ErrNotFound, "a broken file is a decoding error, not a missing webhook")
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
