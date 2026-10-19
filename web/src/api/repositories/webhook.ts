import { apiFetch } from '@/api/client';
import type {
  WebhookCreateRequest,
  WebhookDeliveryResponse,
  WebhookResponse,
  WebhookUpdateRequest,
} from '@/api/types';

export const webhookRepository = {
  /** list returns every webhook with the state of its deliveries. */
  list(): Promise<WebhookResponse[]> {
    return apiFetch('/webhooks', {
      method: 'GET',
    });
  },

  /** get returns the webhook with the given id. */
  get(id: string): Promise<WebhookResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'GET',
    });
  },

  /** create stores a new webhook. */
  create(payload: WebhookCreateRequest): Promise<WebhookResponse> {
    return apiFetch('/webhooks', {
      method: 'POST',
      body: payload,
    });
  },

  /** update replaces the settings of the webhook with the given id. */
  update(id: string, payload: WebhookUpdateRequest): Promise<WebhookResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** delete removes the webhook with the given id. */
  delete(id: string): Promise<void> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },

  /** test sends a test message to the webhook with the given id and returns the delivery. */
  test(id: string): Promise<WebhookDeliveryResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    });
  },
};
