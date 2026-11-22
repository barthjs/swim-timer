import type { WebhookEvent, WebhookResponse } from '@/api/types/webhook';
import type { WebhookDeliveryResponse } from '@/api/types/webhookDelivery';

/** An event a webhook can receive, with its name and what it announces. */
export type WebhookEventOption = {
  value: WebhookEvent;
  label: string;
  description: string;
};

/** The number of random bytes of a generated secret, as the server generates it. */
const WEBHOOK_SECRET_BYTES = 32;

/** Summarizes how the deliveries to a webhook go. */
export type WebhookHealth = 'disabled' | 'waiting' | 'ok' | 'failing';

/** Sorts the states of the webhooks so the failing ones come first. */
export const WEBHOOK_HEALTH_ORDER: Record<WebhookHealth, number> = {
  failing: 0,
  waiting: 1,
  ok: 2,
  disabled: 3,
};

/**
 * Returns how the deliveries to a webhook go: disabled while it is off, failing after a failed delivery,
 * ok after a successful one, and waiting before the first delivery since the start.
 */
export function webhookHealth(webhook: Pick<WebhookResponse, 'enabled' | 'status'>): WebhookHealth {
  if (!webhook.enabled) return 'disabled';
  if (webhook.status.consecutiveFailures > 0) return 'failing';
  if (webhook.status.lastDeliveredAt) return 'ok';
  return 'waiting';
}

/** Reports whether the name or the URL of a webhook contains the search, ignoring letter case. */
export function matchesWebhookSearch(
  webhook: Pick<WebhookResponse, 'name' | 'url'>,
  search: string,
): boolean {
  const term = search.trim().toLowerCase();
  if (!term) return true;
  return webhook.name.toLowerCase().includes(term) || webhook.url.toLowerCase().includes(term);
}

/**
 * Reports whether the event, the outcome, the status,
 * or the answer of a delivery contains the search, ignoring letter case.
 */
export function matchesDeliverySearch(delivery: WebhookDeliveryResponse, search: string): boolean {
  const term = search.trim().toLowerCase();
  if (!term) return true;
  return [
    delivery.event,
    delivery.outcome,
    String(delivery.statusCode),
    delivery.error,
    delivery.responseBody,
  ]
    .join(' ')
    .toLowerCase()
    .includes(term);
}

/**
 * Returns a new random secret in the format of the Standard Webhooks specification.
 */
export function generateWebhookSecret(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(WEBHOOK_SECRET_BYTES));
  return `whsec_${btoa(String.fromCharCode(...bytes))}`;
}

/** Indents the JSON body of a message for reading, and returns a body that is no JSON as it is. */
export function formatMessageBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}
