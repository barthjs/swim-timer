import { apiFetch } from '@/api/client';
import type { WebhookDeliveryResponse } from '@/api/types';

export const webhookDeliveryRepository = {
  /** list returns the last deliveries to the webhook with the given id, the most recent first. */
  list(webhookId: string): Promise<WebhookDeliveryResponse[]> {
    return apiFetch(`/webhooks/${encodeURIComponent(webhookId)}/deliveries`, {
      method: 'GET',
    });
  },
};
