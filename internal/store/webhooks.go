package store

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
)

const (
	// webhooksFolder is the storage folder the webhooks are kept in.
	webhooksFolder = "webhooks"
	// webhookExtension is the extension of a webhook file.
	webhookExtension = ".json"
)

// WebhookStore keeps every webhook as a JSON file in the storage, named after the id of the webhook.
type WebhookStore struct {
	storage media.Storage

	mu sync.Mutex
}

// NewWebhookStore creates a WebhookStore.
func NewWebhookStore(storage media.Storage) *WebhookStore {
	return &WebhookStore{
		storage: storage,
	}
}

// List returns all webhooks, sorted by name. A file that cannot be read is left out and reported
// with ErrUnreadableFile, together with the webhooks that could be read.
func (s *WebhookStore) List(ctx context.Context) ([]domain.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths, err := s.storage.List(ctx, webhooksFolder+"/")
	if err != nil {
		return nil, fmt.Errorf("store: listing webhooks: %w", err)
	}

	webhooks := make([]domain.Webhook, 0, len(paths))
	var unreadable []error
	for _, p := range paths {
		id, ok := fileName(webhooksFolder, p, webhookExtension)
		if !ok {
			continue
		}
		webhook, err := s.read(ctx, id)
		if err != nil {
			unreadable = append(unreadable, fmt.Errorf("%w: %w", ErrUnreadableFile, err))
			continue
		}
		webhooks = append(webhooks, *webhook)
	}
	slices.SortFunc(webhooks, func(a, b domain.Webhook) int {
		return cmp.Or(strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), strings.Compare(a.ID, b.ID))
	})

	return webhooks, errors.Join(unreadable...)
}

// Get returns the webhook with the given id, or apperr.ErrNotFound if it does not exist.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Get(ctx context.Context, id string) (*domain.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read(ctx, id)
}

// Save writes the webhook under the given id, replacing an existing one.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Save(ctx context.Context, id string, webhook domain.Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(webhooksFolder, id, webhookExtension)
	if err != nil {
		return err
	}

	data, err := json.Marshal(webhook, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("store: encoding webhook %q: %w", id, err)
	}

	if err := s.storage.Put(ctx, p, bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("store: saving webhook %q: %w", id, err)
	}

	return nil
}

// Delete removes the webhook with the given id, or returns apperr.ErrNotFound if it does not exist.
// An id that cannot address a webhook file fails with ErrInvalidName.
func (s *WebhookStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := filePath(webhooksFolder, id, webhookExtension)
	if err != nil {
		return err
	}

	exists, err := s.storage.Exists(ctx, p)
	if err != nil {
		return fmt.Errorf("store: checking webhook %q: %w", id, err)
	}
	if !exists {
		return apperr.ErrNotFound
	}

	if err := s.storage.Delete(ctx, p); err != nil {
		return fmt.Errorf("store: deleting webhook %q: %w", id, err)
	}

	return nil
}

// read returns the webhook with the given id while the caller holds the lock, see Get.
func (s *WebhookStore) read(ctx context.Context, id string) (*domain.Webhook, error) {
	p, err := filePath(webhooksFolder, id, webhookExtension)
	if err != nil {
		return nil, err
	}

	data, err := media.ReadAll(ctx, s.storage, p)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("store: reading webhook %q: %w", id, err)
	}

	var webhook domain.Webhook
	if err := json.Unmarshal(data, &webhook); err != nil {
		return nil, fmt.Errorf("store: decoding webhook %q: %w", id, err)
	}

	return &webhook, nil
}
