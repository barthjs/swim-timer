package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/service"
	"github.com/barthjs/swim-timer/internal/webhook"
)

const (
	// webhookTestRateLimit bounds how often a client can send test deliveries, since each one requests another host.
	webhookTestRateLimit       = 10
	webhookTestRateLimitWindow = time.Minute
)

// webhookListHandler returns all webhooks.
//
//	@Summary		List webhooks
//	@Description	Returns all webhooks with the state of their deliveries since the start, sorted by name (Admin only).
//	@Tags			webhooks
//	@Produce		json
//	@Success		200	{array}		WebhookResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks [get]
func (a *API) webhookListHandler(w http.ResponseWriter, r *http.Request) {
	states, err := a.webhookService.List(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	resp := make([]WebhookResponse, 0, len(states))
	for _, state := range states {
		resp = append(resp, toWebhookResponse(state))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

type WebhookResponse struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Events  []string `json:"events" enums:"heat.updated,meet.reloaded"`
	Enabled bool     `json:"enabled"`
	// Secret is only set in the response that created the webhook, since the secret is never shown again.
	Secret        string                `json:"secret,omitzero"`
	SkipTLSVerify bool                  `json:"skipTlsVerify"`
	Status        WebhookStatusResponse `json:"status"`
}

// WebhookStatusResponse is the state of the deliveries to a webhook since the start of the application,
// whose times are left out before the first delivery of their kind.
type WebhookStatusResponse struct {
	LastDeliveredAt *time.Time `json:"lastDeliveredAt,omitzero"`
	LastFailedAt    *time.Time `json:"lastFailedAt,omitzero"`
	// LastStatusCode is the status of the response to the last failed delivery, left out before the first failure.
	LastStatusCode int `json:"lastStatusCode,omitzero"`
	// LastErrorCode names why the last failed delivery got no response, left out before the first failure.
	LastErrorCode       string `json:"lastErrorCode,omitzero" enums:"blocked_address,invalid_url,invalid_secret,untrusted_certificate,timeout,unreachable"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
}

// webhookGetHandler returns a single webhook.
//
//	@Summary		Get webhook
//	@Description	Returns the webhook with the given id and the state of its deliveries since the start (Admin only).
//	@Tags			webhooks
//	@Produce		json
//	@Param			id	path		string	true	"Webhook id"
//	@Success		200	{object}	WebhookResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id} [get]
func (a *API) webhookGetHandler(w http.ResponseWriter, r *http.Request) {
	state, err := a.webhookService.Get(r.Context(), pathParam(r, "id"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toWebhookResponse(state), nil)
}

type WebhookCreateRequest struct {
	Name    string   `json:"name" validate:"required,max=100"`
	URL     string   `json:"url" validate:"required,max=2048"`
	Events  []string `json:"events" validate:"required,min=1,max=2,unique,dive,oneof=heat.updated meet.reloaded" enums:"heat.updated,meet.reloaded"`
	Enabled bool     `json:"enabled"`
	// Secret is generated when empty.
	Secret        string `json:"secret" validate:"max=255"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
}

// webhookCreateHandler creates a webhook.
//
//	@Summary		Create webhook
//	@Description	Creates a webhook under a generated id and starts its deliveries. An empty secret is generated. The response is the only one that holds the secret. The URL may use http or https and must not point to a private or local network outside the allowed networks of the configuration (Admin only).
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		WebhookCreateRequest	true	"Webhook"
//	@Success		201		{object}	WebhookResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks [post]
func (a *API) webhookCreateHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[WebhookCreateRequest](w, r)
	if !ok {
		return
	}

	state, err := a.webhookService.Create(r.Context(), service.WebhookCreateInput{
		Name:          req.Name,
		URL:           req.URL,
		Events:        toWebhookEvents(req.Events),
		Enabled:       req.Enabled,
		Secret:        req.Secret,
		SkipTLSVerify: req.SkipTLSVerify,
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	resp := toWebhookResponse(state)
	resp.Secret = state.Webhook.Secret
	a.writeJSON(w, r, http.StatusCreated, resp, nil)
}

type WebhookUpdateRequest struct {
	Name    string   `json:"name" validate:"required,max=100"`
	URL     string   `json:"url" validate:"required,max=2048"`
	Events  []string `json:"events" validate:"required,min=1,max=2,unique,dive,oneof=heat.updated meet.reloaded" enums:"heat.updated,meet.reloaded"`
	Enabled bool     `json:"enabled"`
	// Secret keeps the stored one when empty.
	Secret        string `json:"secret" validate:"max=255"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
}

// webhookUpdateHandler replaces the settings of a webhook.
//
//	@Summary		Update webhook
//	@Description	Replaces the settings of the webhook with the given id, which take effect with its next delivery. An empty secret keeps the stored one. The URL is checked as on create (Admin only).
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Webhook id"
//	@Param			request	body		WebhookUpdateRequest	true	"Webhook"
//	@Success		200		{object}	WebhookResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id} [put]
func (a *API) webhookUpdateHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[WebhookUpdateRequest](w, r)
	if !ok {
		return
	}

	state, err := a.webhookService.Update(r.Context(), pathParam(r, "id"), service.WebhookUpdateInput{
		Name:          req.Name,
		URL:           req.URL,
		Events:        toWebhookEvents(req.Events),
		Enabled:       req.Enabled,
		Secret:        req.Secret,
		SkipTLSVerify: req.SkipTLSVerify,
	})
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toWebhookResponse(state), nil)
}

// webhookDeleteHandler deletes a webhook.
//
//	@Summary		Delete webhook
//	@Description	Deletes the webhook with the given id and ends its deliveries (Admin only).
//	@Tags			webhooks
//	@Param			id	path	string	true	"Webhook id"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id} [delete]
func (a *API) webhookDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.webhookService.Delete(r.Context(), pathParam(r, "id")); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// webhookTestHandler sends a test message to a webhook.
//
//	@Summary		Test webhook
//	@Description	Sends a message of the type webhook.test to the webhook with the given id once, even if it is disabled, and returns the delivery. The delivery shows in the log but leaves the state of the webhook alone (Admin only).
//	@Tags			webhooks
//	@Produce		json
//	@Param			id	path		string	true	"Webhook id"
//	@Success		200	{object}	WebhookDeliveryResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id}/test [post]
func (a *API) webhookTestHandler(w http.ResponseWriter, r *http.Request) {
	delivery, err := a.webhookService.Test(r.Context(), pathParam(r, "id"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toWebhookDeliveryResponse(delivery), nil)
}

type WebhookDeliveryResponse struct {
	MessageID   string    `json:"messageId"`
	Event       string    `json:"event"`
	AttemptedAt time.Time `json:"attemptedAt"`
	Outcome     string    `json:"outcome" enums:"delivered,retry,deferred,gone,rejected"`
	// StatusCode is the status of the response, 0 if none was received.
	StatusCode int   `json:"statusCode"`
	DurationMs int64 `json:"durationMs"`
	// ResponseBody is the start of the response body.
	ResponseBody string `json:"responseBody"`
	// ErrorCode names why a delivery got no response, empty if the target answered.
	ErrorCode string `json:"errorCode" enums:"blocked_address,invalid_url,invalid_secret,untrusted_certificate,timeout,unreachable"`
	// Error gives the technical details of ErrorCode for the log.
	Error string `json:"error"`
}

// webhookDeliveryListHandler returns the last deliveries to a webhook.
//
//	@Summary		List webhook deliveries
//	@Description	Returns the last deliveries to the webhook with the given id since the start, the most recent first (Admin only).
//	@Tags			webhooks
//	@Produce		json
//	@Param			id	path		string	true	"Webhook id"
//	@Success		200	{array}		WebhookDeliveryResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id}/deliveries [get]
func (a *API) webhookDeliveryListHandler(w http.ResponseWriter, r *http.Request) {
	deliveries, err := a.webhookService.ListDeliveries(r.Context(), pathParam(r, "id"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	resp := make([]WebhookDeliveryResponse, 0, len(deliveries))
	for _, delivery := range deliveries {
		resp = append(resp, toWebhookDeliveryResponse(delivery))
	}

	a.writeJSON(w, r, http.StatusOK, resp, nil)
}

// webhookPreviewGetHandler returns a message to a webhook as it would be sent now, without sending it.
//
//	@Summary		Get webhook preview
//	@Description	Returns the request a message of the given event with its current data would make to the webhook with the given id, with the signature of its secret. Nothing is sent (Admin only).
//	@Tags			webhooks
//	@Produce		json
//	@Param			id		path		string	true	"Webhook id"
//	@Param			event	query		string	true	"Event"	Enums(heat.updated, meet.reloaded)
//	@Success		200		{object}	WebhookPreviewResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/webhooks/{id}/preview [get]
func (a *API) webhookPreviewGetHandler(w http.ResponseWriter, r *http.Request) {
	p := newQueryParser(r)
	name := p.optionalString("event")
	if p.err != nil {
		a.badRequestResponse(w, r, p.err)
		return
	}
	if name == nil {
		a.badRequestResponse(w, r, errors.New("missing query parameter event"))
		return
	}
	event, ok := domain.ParseWebhookEvent(*name)
	if !ok {
		a.badRequestResponse(w, r, errors.New("invalid query parameter event"))
		return
	}

	request, err := a.webhookService.Preview(r.Context(), pathParam(r, "id"), event)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toWebhookPreviewResponse(request), nil)
}

type WebhookPreviewResponse struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	// Headers are sorted by name.
	Headers []WebhookHeaderResponse `json:"headers"`
	Body    string                  `json:"body"`
}

type WebhookHeaderResponse struct {
	// Name is in lowercase.
	Name  string `json:"name"`
	Value string `json:"value"`
}

// toWebhookResponse maps a webhook with the state of its deliveries to its response.
func toWebhookResponse(state service.WebhookState) WebhookResponse {
	events := make([]string, 0, len(state.Webhook.Events))
	for _, event := range state.Webhook.Events {
		events = append(events, string(event))
	}

	return WebhookResponse{
		ID:            state.Webhook.ID,
		Name:          state.Webhook.Name,
		URL:           state.Webhook.URL,
		Events:        events,
		Enabled:       state.Webhook.Enabled,
		SkipTLSVerify: state.Webhook.SkipTLSVerify,
		Status: WebhookStatusResponse{
			LastDeliveredAt:     optionalTime(state.Status.LastDeliveredAt),
			LastFailedAt:        optionalTime(state.Status.LastFailedAt),
			LastStatusCode:      state.Status.LastStatusCode,
			LastErrorCode:       string(state.Status.LastErrorCode),
			ConsecutiveFailures: state.Status.ConsecutiveFailures,
		},
	}
}

// toWebhookEvents maps the events of a request, which the validation limited to the known ones.
func toWebhookEvents(events []string) []domain.WebhookEvent {
	result := make([]domain.WebhookEvent, 0, len(events))
	for _, event := range events {
		result = append(result, domain.WebhookEvent(event))
	}
	return result
}

// toWebhookDeliveryResponse maps a delivery to a webhook to its response.
func toWebhookDeliveryResponse(delivery service.WebhookDelivery) WebhookDeliveryResponse {
	return WebhookDeliveryResponse{
		MessageID:    delivery.MessageID,
		Event:        delivery.Event,
		AttemptedAt:  delivery.AttemptedAt,
		Outcome:      string(delivery.Outcome),
		StatusCode:   delivery.StatusCode,
		DurationMs:   delivery.Duration.Milliseconds(),
		ResponseBody: delivery.ResponseBody,
		ErrorCode:    string(delivery.ErrorCode),
		Error:        delivery.Error,
	}
}

// toWebhookPreviewResponse maps a request to a webhook to its response.
func toWebhookPreviewResponse(request webhook.Request) WebhookPreviewResponse {
	headers := make([]WebhookHeaderResponse, 0, len(request.Header))
	for name, values := range request.Header {
		headers = append(headers, WebhookHeaderResponse{
			Name:  strings.ToLower(name),
			Value: strings.Join(values, ", "),
		})
	}
	slices.SortFunc(headers, func(a, b WebhookHeaderResponse) int { return strings.Compare(a.Name, b.Name) })

	return WebhookPreviewResponse{
		Method:  request.Method,
		URL:     request.URL,
		Headers: headers,
		Body:    string(request.Body),
	}
}
