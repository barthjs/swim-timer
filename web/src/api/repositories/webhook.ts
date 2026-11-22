import { apiFetch } from '@/api/client/client';
import type {
  WebhookCreateRequest,
  WebhookResponse,
  WebhookUpdateRequest,
} from '@/api/types/webhook';
import type { WebhookDeliveryResponse } from '@/api/types/webhookDelivery';

export const webhookRepository = {
  /** Returns every webhook with the state of its deliveries. */
  async list(): Promise<WebhookResponse[]> {
    return apiFetch('/webhooks', {
      method: 'GET',
    });
  },

  /** Returns the webhook with the given id. */
  async get(id: string): Promise<WebhookResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'GET',
    });
  },

  /** Stores a new webhook. */
  async create(payload: WebhookCreateRequest): Promise<WebhookResponse> {
    return apiFetch('/webhooks', {
      method: 'POST',
      body: payload,
    });
  },

  /** Replaces the settings of the webhook with the given id. */
  async update(id: string, payload: WebhookUpdateRequest): Promise<WebhookResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** Removes the webhook with the given id. */
  async delete(id: string): Promise<void> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },

  /** Sends a test message to the webhook with the given id and returns the delivery. */
  async test(id: string): Promise<WebhookDeliveryResponse> {
    return apiFetch(`/webhooks/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    });
  },
};
