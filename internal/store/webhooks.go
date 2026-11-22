package store

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

// webhooksFolder is the storage folder the webhooks are kept in.
const webhooksFolder = "webhooks"

// WebhookStore keeps every webhook as a JSON file in the storage, named after the id of the webhook.
type WebhookStore struct {
	files *jsonFiles[domain.Webhook]
}

// NewWebhookStore creates a WebhookStore.
func NewWebhookStore(storage media.Storage) *WebhookStore {
	return &WebhookStore{
		files: newJSONFiles[domain.Webhook](storage, webhooksFolder, "webhook"),
	}
}

// List returns all webhooks that could be read, sorted by name, and reports a file that cannot be read with ErrUnreadableFile.
func (s *WebhookStore) List(ctx context.Context) ([]domain.Webhook, error) {
	webhooks, err := s.files.list(ctx)
	slices.SortFunc(webhooks, func(a, b domain.Webhook) int {
		return cmp.Or(strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), strings.Compare(a.ID, b.ID))
	})

	return webhooks, err
}

// Get returns the webhook with the given id, or apperr.ErrNotFound if it does not exist.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Get(ctx context.Context, id string) (*domain.Webhook, error) {
	return s.files.get(ctx, id)
}

// Save writes the webhook under the given id, replacing an existing one.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Save(ctx context.Context, id string, webhook domain.Webhook) error {
	return s.files.save(ctx, id, webhook)
}

// Delete removes the webhook with the given id, or returns apperr.ErrNotFound if it does not exist.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Delete(ctx context.Context, id string) error {
	return s.files.delete(ctx, id)
}
