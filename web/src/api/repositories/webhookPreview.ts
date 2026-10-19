import { apiFetch } from '@/api/client';
import type { WebhookEvent, WebhookPreviewResponse } from '@/api/types';

export const webhookPreviewRepository = {
  /** get returns a message of the event to the webhook with the given id as it would be sent now. */
  get(webhookId: string, event: WebhookEvent): Promise<WebhookPreviewResponse> {
    const query = new URLSearchParams({
      event,
    });
    return apiFetch(`/webhooks/${encodeURIComponent(webhookId)}/preview?${query.toString()}`, {
      method: 'GET',
    });
  },
};
