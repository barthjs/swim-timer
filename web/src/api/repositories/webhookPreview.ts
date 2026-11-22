import { apiFetch } from '@/api/client/client';
import type { WebhookEvent } from '@/api/types/webhook';
import type { WebhookPreviewResponse } from '@/api/types/webhookPreview';

export const webhookPreviewRepository = {
  /** Returns a message of the event to the webhook with the given id as it would be sent now. */
  async get(webhookId: string, event: WebhookEvent): Promise<WebhookPreviewResponse> {
    const query = new URLSearchParams({
      event,
    });
    return apiFetch(`/webhooks/${encodeURIComponent(webhookId)}/preview?${query.toString()}`, {
      method: 'GET',
    });
  },
};
