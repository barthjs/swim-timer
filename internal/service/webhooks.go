package service

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/webhook"
)

const (
	// webhookTestEvent is the type of the message a test delivery sends.
	webhookTestEvent = "webhook.test"
	// webhookFirstBackoff is the wait before the first retry, doubled for every further one.
	webhookFirstBackoff = time.Second
	// webhookMaxBackoff caps the wait between retries, so the data flows again soon after an outage of a target.
	webhookMaxBackoff = time.Minute
	// webhookMaxDeferral caps the wait a target may ask for with Retry-After.
	webhookMaxDeferral = 5 * time.Minute
	// webhookDeliveryLogSize is how many deliveries the log of a webhook keeps, one page of the table that shows them.
	webhookDeliveryLogSize = 25
)

// WebhookDeliveryOutcome is how a delivery to a webhook ended.
type WebhookDeliveryOutcome string

const (
	WebhookDeliveryDelivered WebhookDeliveryOutcome = "delivered"
	WebhookDeliveryRetry     WebhookDeliveryOutcome = "retry"
	WebhookDeliveryDeferred  WebhookDeliveryOutcome = "deferred"
	WebhookDeliveryGone      WebhookDeliveryOutcome = "gone"
	WebhookDeliveryRejected  WebhookDeliveryOutcome = "rejected"
)

// WebhookDeliveryErrorCode names why a delivery got no response, so the interface can describe it in its language.
type WebhookDeliveryErrorCode string

const (
	WebhookDeliveryErrorBlockedAddress WebhookDeliveryErrorCode = "blocked_address"
	WebhookDeliveryErrorCertificate    WebhookDeliveryErrorCode = "untrusted_certificate"
	WebhookDeliveryErrorInvalidURL     WebhookDeliveryErrorCode = "invalid_url"
	WebhookDeliveryErrorInvalidSecret  WebhookDeliveryErrorCode = "invalid_secret"
	WebhookDeliveryErrorTimeout        WebhookDeliveryErrorCode = "timeout"
	WebhookDeliveryErrorUnreachable    WebhookDeliveryErrorCode = "unreachable"
)

// WebhookStatus is the state of the deliveries to a webhook since the application started.
type WebhookStatus struct {
	// LastDeliveredAt and LastFailedAt are zero before the first delivery of their kind.
	LastDeliveredAt time.Time
	LastFailedAt    time.Time
	// LastStatusCode and LastErrorCode describe the last failed delivery by the status of its response,
	// or by the reason it got none. Both are zero before the first failure.
	LastStatusCode      int
	LastErrorCode       WebhookDeliveryErrorCode
	ConsecutiveFailures int
}

// WebhookState is a webhook with the state of its deliveries.
type WebhookState struct {
	Webhook domain.Webhook
	Status  WebhookStatus
}

// WebhookDelivery is one attempt to deliver a message to a webhook.
type WebhookDelivery struct {
	MessageID   string
	Event       string
	AttemptedAt time.Time
	Outcome     WebhookDeliveryOutcome
	// StatusCode is the status of the response, 0 if none was received.
	StatusCode int
	Duration   time.Duration
	// ResponseBody is the start of the response body.
	ResponseBody string
	// ErrorCode names why a delivery got no response, and Error gives the technical details for the log.
	// Both are empty if the target answered.
	ErrorCode WebhookDeliveryErrorCode
	Error     string
}

// WebhookCreateInput is a new webhook. An empty secret is generated.
type WebhookCreateInput struct {
	Name    string
	URL     string
	Events  []domain.WebhookEvent
	Enabled bool
	Secret  string
	// SkipTLSVerify accepts any certificate of the endpoint, such as a self-signed one.
	SkipTLSVerify bool
}

// WebhookUpdateInput replaces the settings of a webhook. An empty secret is generated anew.
type WebhookUpdateInput struct {
	Name    string
	URL     string
	Events  []domain.WebhookEvent
	Enabled bool
	Secret  string
	// SkipTLSVerify accepts any certificate of the endpoint, such as a self-signed one.
	SkipTLSVerify bool
}

// WebhookService manages the webhooks and delivers the published events to them. Every webhook has its own
// delivery goroutine that keeps only the latest message per event, since a newer state of the live data replaces
// an older one. A slow or unreachable target therefore receives fewer intermediate states but never delays
// the overlays or the other targets.
type WebhookService struct {
	logger zerolog.Logger
	client *webhook.Client
	store  *store.WebhookStore
	// sample returns the current data of an event for a preview, nil if there is none.
	sample func(event domain.WebhookEvent) any

	// writeMu serializes the changes of the stored webhooks with the changes of their targets,
	// so a deleted webhook never gets a target again and a 410 never overwrites a newer setting.
	writeMu sync.Mutex

	// mu guards the fields below. runCtx is the context of Run, nil while no delivery goroutine may start.
	mu      sync.Mutex
	targets map[string]*webhookTarget
	runCtx  context.Context
	workers sync.WaitGroup
}

// webhookTarget is a webhook with its pending messages, the state of its deliveries, and its log.
type webhookTarget struct {
	// wake signals the delivery goroutine that a message is pending.
	wake chan struct{}
	// stop ends the delivery goroutine, nil while none runs.
	stop context.CancelFunc

	mu         sync.Mutex
	webhook    domain.Webhook
	pending    map[domain.WebhookEvent]webhook.Payload
	status     WebhookStatus
	deliveries []WebhookDelivery
}

// NewWebhookService creates a WebhookService. Run loads the webhooks and starts their deliveries.
// sample returns the current data of an event, which Preview sends as the data of the message. It may be nil.
func NewWebhookService(
	logger zerolog.Logger,
	client *webhook.Client,
	store *store.WebhookStore,
	sample func(event domain.WebhookEvent) any,
) *WebhookService {
	return &WebhookService{
		logger:  logger,
		client:  client,
		store:   store,
		sample:  sample,
		targets: make(map[string]*webhookTarget),
	}
}

// List returns all webhooks with the state of their deliveries, sorted by name.
func (s *WebhookService) List(ctx context.Context) ([]WebhookState, error) {
	webhooks, err := s.listStored(ctx)
	if err != nil {
		return nil, err
	}

	states := make([]WebhookState, 0, len(webhooks))
	for _, wh := range webhooks {
		states = append(states, WebhookState{
			Webhook: wh,
			Status:  s.status(wh.ID),
		})
	}
	return states, nil
}

// ListDeliveries returns the last deliveries to the webhook with the given id since the application started,
// the most recent first, or apperr.ErrNotFound if it does not exist.
func (s *WebhookService) ListDeliveries(ctx context.Context, id string) ([]WebhookDelivery, error) {
	if _, err := s.get(ctx, id); err != nil {
		return nil, err
	}

	target := s.target(id)
	if target == nil {
		return []WebhookDelivery{}, nil
	}
	target.mu.Lock()
	defer target.mu.Unlock()

	deliveries := make([]WebhookDelivery, 0, len(target.deliveries))
	for i := len(target.deliveries) - 1; i >= 0; i-- {
		deliveries = append(deliveries, target.deliveries[i])
	}
	return deliveries, nil
}

// Get returns the webhook with the given id and the state of its deliveries,
// or apperr.ErrNotFound if it does not exist or id cannot address a webhook file.
func (s *WebhookService) Get(ctx context.Context, id string) (WebhookState, error) {
	wh, err := s.get(ctx, id)
	if err != nil {
		return WebhookState{}, err
	}
	return WebhookState{
		Webhook: *wh,
		Status:  s.status(id),
	}, nil
}

// Create stores a new webhook under a generated id and starts its deliveries.
// A URL that cannot be delivered to or a malformed secret fails with a validation error.
func (s *WebhookService) Create(ctx context.Context, input WebhookCreateInput) (WebhookState, error) {
	wh := domain.Webhook{
		ID:            strings.ToLower(rand.Text()),
		Name:          input.Name,
		URL:           input.URL,
		Events:        input.Events,
		Enabled:       input.Enabled,
		Secret:        input.Secret,
		SkipTLSVerify: input.SkipTLSVerify,
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.save(ctx, &wh); err != nil {
		return WebhookState{}, err
	}
	return WebhookState{
		Webhook: wh,
		Status:  s.status(wh.ID),
	}, nil
}

// Update replaces the settings of the webhook with the given id, which take effect with its next delivery,
// or returns apperr.ErrNotFound if it does not exist. It validates like Create.
func (s *WebhookService) Update(ctx context.Context, id string, input WebhookUpdateInput) (WebhookState, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := s.get(ctx, id); err != nil {
		return WebhookState{}, err
	}

	wh := domain.Webhook{
		ID:            id,
		Name:          input.Name,
		URL:           input.URL,
		Events:        input.Events,
		Enabled:       input.Enabled,
		Secret:        input.Secret,
		SkipTLSVerify: input.SkipTLSVerify,
	}
	if err := s.save(ctx, &wh); err != nil {
		return WebhookState{}, err
	}
	return WebhookState{
		Webhook: wh,
		Status:  s.status(id),
	}, nil
}

// Delete removes the webhook with the given id and ends its deliveries,
// or returns apperr.ErrNotFound if it does not exist or id cannot address a webhook file.
func (s *WebhookService) Delete(ctx context.Context, id string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	err := s.store.Delete(ctx, id)
	if errors.Is(err, store.ErrInvalidName) {
		return apperr.ErrNotFound
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if target, ok := s.targets[id]; ok {
		if target.stop != nil {
			target.stop()
		}
		delete(s.targets, id)
	}
	return nil
}

// Test delivers a test message to the webhook with the given id once, even if it is disabled, and returns
// the delivery, or apperr.ErrNotFound if the webhook does not exist. The delivery counts towards its state.
func (s *WebhookService) Test(ctx context.Context, id string) (WebhookDelivery, error) {
	wh, err := s.get(ctx, id)
	if err != nil {
		return WebhookDelivery{}, err
	}

	payload, err := webhook.NewPayload(webhookTestEvent, nil)
	if err != nil {
		return WebhookDelivery{}, err
	}
	result := s.client.Send(ctx, webhook.Target{
		URL: wh.URL,
		Secrets: []string{
			wh.Secret,
		},
		SkipTLSVerify: wh.SkipTLSVerify,
	}, payload)
	delivery := newDelivery(webhookTestEvent, payload, result)

	// The webhook may have been changed or deleted while the request ran, and a deleted one gets no target again
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	current, err := s.get(ctx, id)
	if errors.Is(err, apperr.ErrNotFound) {
		return delivery, nil
	}
	if err != nil {
		return WebhookDelivery{}, err
	}
	s.ensureTarget(*current).record(delivery, result.Outcome)
	return delivery, nil
}

// Preview returns the message of an event with its current data as it would be sent to the webhook with the given id
// now, signed with its secret, or apperr.ErrNotFound if the webhook does not exist. Nothing is sent.
func (s *WebhookService) Preview(ctx context.Context, id string, event domain.WebhookEvent) (webhook.Request, error) {
	wh, err := s.get(ctx, id)
	if err != nil {
		return webhook.Request{}, err
	}

	var data any
	if s.sample != nil {
		data = s.sample(event)
	}
	payload, err := webhook.NewPayload(string(event), data)
	if err != nil {
		return webhook.Request{}, err
	}
	return s.client.Preview(webhook.Target{
		URL: wh.URL,
		Secrets: []string{
			wh.Secret,
		},
		SkipTLSVerify: wh.SkipTLSVerify,
	}, payload)
}

// Publish hands an event to every enabled webhook that subscribes to it. data builds the data of the message only
// if one does, and it is encoded once, right away, so the caller may change it afterwards. A webhook still delivering
// keeps only the latest message of the event.
func (s *WebhookService) Publish(event domain.WebhookEvent, data func() any) {
	s.mu.Lock()
	targets := make([]*webhookTarget, 0, len(s.targets))
	for _, target := range s.targets {
		if target.subscribes(event) {
			targets = append(targets, target)
		}
	}
	s.mu.Unlock()
	if len(targets) == 0 {
		return
	}

	payload, err := webhook.NewPayload(string(event), data())
	if err != nil {
		s.logger.Error().Err(err).Str("event", string(event)).Msg("failed to encode webhook message")
		return
	}
	for _, target := range targets {
		target.offer(event, payload)
	}
}

// Run loads the webhooks and delivers the published events to them until ctx is canceled,
// then waits for the deliveries in progress to end. A webhook that cannot be loaded is logged and left out.
func (s *WebhookService) Run(ctx context.Context) {
	webhooks, err := s.listStored(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to load webhooks, no webhook is delivered until one is saved")
	}

	s.mu.Lock()
	s.runCtx = ctx
	for _, wh := range webhooks {
		s.addTargetLocked(wh)
	}
	for id, target := range s.targets {
		s.startLocked(id, target)
	}
	s.mu.Unlock()

	<-ctx.Done()
	// A webhook saved from now on gets no delivery goroutine, since the wait below may already have begun
	s.mu.Lock()
	s.runCtx = nil
	s.mu.Unlock()
	s.workers.Wait()
}

// listStored returns the stored webhooks. A file that cannot be read is logged and left out, so the others
// stay available and can still be delivered to.
func (s *WebhookService) listStored(ctx context.Context) ([]domain.Webhook, error) {
	webhooks, err := s.store.List(ctx)
	if errors.Is(err, store.ErrUnreadableFile) {
		s.logger.Error().Err(err).Msg("left out webhooks whose files cannot be read")
		return webhooks, nil
	}
	return webhooks, err
}

// get returns the stored webhook with the given id, or apperr.ErrNotFound if it does not exist
// or id cannot address a webhook file.
func (s *WebhookService) get(ctx context.Context, id string) (*domain.Webhook, error) {
	wh, err := s.store.Get(ctx, id)
	if errors.Is(err, store.ErrInvalidName) {
		return nil, apperr.ErrNotFound
	}
	return wh, err
}

// save checks the URL and the secret of a webhook, generates a missing secret, stores the webhook,
// and hands it to its delivery goroutine.
func (s *WebhookService) save(ctx context.Context, wh *domain.Webhook) error {
	if err := s.client.CheckURL(wh.URL); err != nil {
		return apperr.NewValidationError("url", urlErrorTag(err))
	}
	if wh.Secret == "" {
		wh.Secret = webhook.GenerateSecret()
	} else if _, err := webhook.ParseSecret(wh.Secret); err != nil {
		return apperr.NewValidationError("secret", "webhook_secret")
	}

	if err := s.store.Save(ctx, wh.ID, *wh); err != nil {
		return err
	}
	s.ensureTarget(*wh)
	return nil
}

// status returns the state of the deliveries to the webhook with the given id.
func (s *WebhookService) status(id string) WebhookStatus {
	target := s.target(id)
	if target == nil {
		return WebhookStatus{}
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	return target.status
}

// target returns the target of the webhook with the given id, nil if it has none yet.
func (s *WebhookService) target(id string) *webhookTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targets[id]
}

// ensureTarget hands a stored webhook to its target, which it creates and starts if Run already runs.
func (s *WebhookService) ensureTarget(wh domain.Webhook) *webhookTarget {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.addTargetLocked(wh)
	if s.runCtx != nil {
		s.startLocked(wh.ID, target)
	}
	return target
}

// addTargetLocked replaces the webhook of an existing target or adds a new target for it. The caller holds mu.
func (s *WebhookService) addTargetLocked(wh domain.Webhook) *webhookTarget {
	if target, ok := s.targets[wh.ID]; ok {
		target.replaceWebhook(wh)
		return target
	}

	target := &webhookTarget{
		wake:    make(chan struct{}, 1),
		webhook: wh,
		pending: make(map[domain.WebhookEvent]webhook.Payload),
	}
	s.targets[wh.ID] = target
	return target
}

// startLocked starts the delivery goroutine of a target unless it runs already. The caller holds mu.
func (s *WebhookService) startLocked(id string, target *webhookTarget) {
	if target.stop != nil {
		return
	}

	ctx, stop := context.WithCancel(s.runCtx)
	target.stop = stop
	s.workers.Go(func() { s.deliverLoop(ctx, id, target) })
}

// deliverLoop delivers the pending messages of a target until ctx is canceled. Each round sends every pending event
// once, so a message that keeps failing never holds back the other events. After a failed round it waits
// with a backoff, or as long as the target asked for.
func (s *WebhookService) deliverLoop(ctx context.Context, id string, target *webhookTarget) {
	backoff := webhookFirstBackoff
	for {
		if !target.hasPending() {
			select {
			case <-ctx.Done():
				return
			case <-target.wake:
			}
		}

		deferral, failed := s.deliverPending(ctx, id, target)
		if ctx.Err() != nil {
			return
		}
		if !failed {
			backoff = webhookFirstBackoff
			continue
		}

		wait := deferral
		if wait == 0 {
			wait = backoff
			backoff = min(2*backoff, webhookMaxBackoff)
		}
		if !sleep(ctx, wait) {
			return
		}
	}
}

// deliverPending sends the pending message of every event once. A message that failed returns to the pending ones,
// unless a newer one of its event arrived meanwhile. It reports whether one failed and the longest wait a target
// asked for.
func (s *WebhookService) deliverPending(ctx context.Context, id string, target *webhookTarget) (time.Duration, bool) {
	var deferral time.Duration
	failed := false
	for _, event := range domain.WebhookEvents() {
		payload, ok := target.take(event)
		if !ok {
			continue
		}
		wh, subscribed := target.subscription(event)
		if !subscribed {
			continue
		}

		result := s.client.Send(ctx, webhook.Target{
			URL: wh.URL,
			Secrets: []string{
				wh.Secret,
			},
			SkipTLSVerify: wh.SkipTLSVerify,
		}, payload)
		if ctx.Err() != nil {
			return 0, false
		}
		target.record(newDelivery(string(event), payload, result), result.Outcome)

		switch result.Outcome {
		case webhook.OutcomeDelivered, webhook.OutcomeRejected:
			// The message is done, delivered or never deliverable
		case webhook.OutcomeGone:
			s.disable(ctx, id, target, wh.URL)
			return 0, false
		case webhook.OutcomeRetry:
			target.requeue(event, payload)
			failed = true
		case webhook.OutcomeDeferred:
			target.requeue(event, payload)
			failed = true
			deferral = max(deferral, min(result.RetryAfter, webhookMaxDeferral))
		}
	}
	return deferral, failed
}

// disable turns off a webhook whose target answered at url that it no longer exists. A webhook that was deleted
// or got another URL meanwhile is left as it is, since the answer no longer applies to it.
func (s *WebhookService) disable(ctx context.Context, id string, target *webhookTarget, url string) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	wh, err := s.store.Get(ctx, id)
	if errors.Is(err, apperr.ErrNotFound) {
		return
	}
	if err != nil {
		s.logger.Error().Err(err).Str("webhook", id).Msg("failed to read the webhook to disable it")
		return
	}
	if wh.URL != url {
		return
	}

	wh.Enabled = false
	if err := s.store.Save(ctx, id, *wh); err != nil {
		s.logger.Error().Err(err).Str("webhook", id).Msg("failed to disable the webhook")
		return
	}
	target.replaceWebhook(*wh)
	s.logger.Warn().Str("webhook", id).Str("url", wh.URL).Msg("disabled the webhook, since its target no longer exists")
}

// replaceWebhook makes the stored settings of a webhook the ones its target delivers with.
func (t *webhookTarget) replaceWebhook(wh domain.Webhook) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.webhook = wh
}

// subscribes reports whether the webhook of the target subscribes to the event.
func (t *webhookTarget) subscribes(event domain.WebhookEvent) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.webhook.Subscribes(event)
}

// subscription returns the current webhook of the target and whether it subscribes to the event.
func (t *webhookTarget) subscription(event domain.WebhookEvent) (domain.Webhook, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.webhook, t.webhook.Subscribes(event)
}

// offer makes a message the pending one of its event, replacing an older one, and wakes the delivery goroutine.
func (t *webhookTarget) offer(event domain.WebhookEvent, payload webhook.Payload) {
	t.mu.Lock()
	t.pending[event] = payload
	t.mu.Unlock()

	select {
	case t.wake <- struct{}{}:
	default:
		// The goroutine is woken already and takes the message with the others
	}
}

// hasPending reports whether a message waits for delivery.
func (t *webhookTarget) hasPending() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.pending) > 0
}

// take removes and returns the pending message of an event.
func (t *webhookTarget) take(event domain.WebhookEvent) (webhook.Payload, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	payload, ok := t.pending[event]
	if ok {
		delete(t.pending, event)
	}
	return payload, ok
}

// requeue returns a message that failed to the pending ones, unless a newer message of its event waits already.
func (t *webhookTarget) requeue(event domain.WebhookEvent, payload webhook.Payload) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.pending[event]; !ok {
		t.pending[event] = payload
	}
}

// record adds a delivery to the log of the target and updates its state by the outcome. A deferred delivery
// is no failure, since the target only asked to wait.
func (t *webhookTarget) record(delivery WebhookDelivery, outcome webhook.Outcome) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch outcome {
	case webhook.OutcomeDelivered:
		t.status.LastDeliveredAt = delivery.AttemptedAt
		t.status.ConsecutiveFailures = 0
	case webhook.OutcomeDeferred:
		// The target asked to wait, which is no failure
	case webhook.OutcomeRetry, webhook.OutcomeGone, webhook.OutcomeRejected:
		t.status.LastFailedAt = delivery.AttemptedAt
		t.status.LastStatusCode = delivery.StatusCode
		t.status.LastErrorCode = delivery.ErrorCode
		t.status.ConsecutiveFailures++
	}

	t.deliveries = append(t.deliveries, delivery)
	if len(t.deliveries) > webhookDeliveryLogSize {
		t.deliveries = t.deliveries[len(t.deliveries)-webhookDeliveryLogSize:]
	}
}

// urlErrorTag returns the validation tag of a URL that cannot be delivered to.
func urlErrorTag(err error) string {
	switch {
	case errors.Is(err, webhook.ErrBlockedAddress):
		return "blocked_address"
	default:
		return "url"
	}
}

// sleep waits for d and reports whether ctx is still active afterwards.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// newDelivery describes one attempt to deliver a message by its result.
func newDelivery(event string, payload webhook.Payload, result webhook.Result) WebhookDelivery {
	delivery := WebhookDelivery{
		MessageID:    payload.ID,
		Event:        event,
		AttemptedAt:  time.Now(),
		Outcome:      deliveryOutcome(result.Outcome),
		StatusCode:   result.StatusCode,
		Duration:     result.Duration,
		ResponseBody: result.ResponseBody,
	}
	if result.Err != nil {
		delivery.ErrorCode = deliveryErrorCode(result.Err)
		delivery.Error = result.Err.Error()
	}
	return delivery
}

// deliveryOutcome maps the outcome of a delivery to the outcome the service reports.
func deliveryOutcome(outcome webhook.Outcome) WebhookDeliveryOutcome {
	switch outcome {
	case webhook.OutcomeDelivered:
		return WebhookDeliveryDelivered
	case webhook.OutcomeRetry:
		return WebhookDeliveryRetry
	case webhook.OutcomeDeferred:
		return WebhookDeliveryDeferred
	case webhook.OutcomeGone:
		return WebhookDeliveryGone
	default:
		return WebhookDeliveryRejected
	}
}

// deliveryErrorCode returns why a delivery got no response.
func deliveryErrorCode(err error) WebhookDeliveryErrorCode {
	switch {
	case errors.Is(err, webhook.ErrBlockedAddress):
		return WebhookDeliveryErrorBlockedAddress
	case errors.Is(err, webhook.ErrUntrustedCertificate):
		return WebhookDeliveryErrorCertificate
	case errors.Is(err, webhook.ErrInvalidURL):
		return WebhookDeliveryErrorInvalidURL
	case errors.Is(err, webhook.ErrInvalidSecret):
		return WebhookDeliveryErrorInvalidSecret
	case isTimeout(err):
		return WebhookDeliveryErrorTimeout
	default:
		return WebhookDeliveryErrorUnreachable
	}
}

// isTimeout reports whether a request ran out of time, by its deadline or by a timeout of the network.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	netErr, ok := errors.AsType[net.Error](err)
	return ok && netErr.Timeout()
}
