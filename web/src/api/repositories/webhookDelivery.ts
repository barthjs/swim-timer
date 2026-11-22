import { apiFetch } from '@/api/client/client';
import type { WebhookDeliveryResponse } from '@/api/types/webhookDelivery';

export const webhookDeliveryRepository = {
  /** Returns the last deliveries to the webhook with the given id, the most recent first. */
  async list(webhookId: string): Promise<WebhookDeliveryResponse[]> {
    return apiFetch(`/webhooks/${encodeURIComponent(webhookId)}/deliveries`, {
      method: 'GET',
    });
  },
};
