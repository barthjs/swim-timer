import type { WebhookDeliveryResponse, WebhookResponse } from '@/api/types';

/** WEBHOOK_SECRET_BYTES is the number of random bytes of a generated secret, as the server generates it. */
const WEBHOOK_SECRET_BYTES = 32;

/** WebhookHealth summarizes how the deliveries to a webhook go. */
export type WebhookHealth = 'disabled' | 'waiting' | 'ok' | 'failing';

/** WEBHOOK_HEALTH_ORDER sorts the states of the webhooks so the failing ones come first. */
export const WEBHOOK_HEALTH_ORDER: Record<WebhookHealth, number> = {
  failing: 0,
  waiting: 1,
  ok: 2,
  disabled: 3,
};

/**
 * webhookHealth returns how the deliveries to a webhook go: disabled while it is off, failing after a failed
 * delivery, ok after a successful one, and waiting before the first delivery since the start.
 */
export function webhookHealth(webhook: Pick<WebhookResponse, 'enabled' | 'status'>): WebhookHealth {
  if (!webhook.enabled) return 'disabled';
  if (webhook.status.consecutiveFailures > 0) return 'failing';
  if (webhook.status.lastDeliveredAt) return 'ok';
  return 'waiting';
}

/** matchesWebhookSearch reports whether the name or the URL of a webhook contains the search, ignoring letter case. */
export function matchesWebhookSearch(
  webhook: Pick<WebhookResponse, 'name' | 'url'>,
  search: string,
): boolean {
  const term = search.trim().toLowerCase();
  if (!term) return true;
  return webhook.name.toLowerCase().includes(term) || webhook.url.toLowerCase().includes(term);
}

/**
 * matchesDeliverySearch reports whether the event, the outcome, the status, or the answer of a delivery
 * contains the search, ignoring letter case.
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
 * generateWebhookSecret returns a new random secret in the format of the Standard Webhooks specification,
 * so one secret can be copied into several webhooks before they are saved.
 */
export function generateWebhookSecret(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(WEBHOOK_SECRET_BYTES));
  return `whsec_${btoa(String.fromCharCode(...bytes))}`;
}

/** formatWebhookTime formats the time of a delivery with the date and the seconds. */
export function formatWebhookTime(iso: string, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'short',
    timeStyle: 'medium',
  }).format(new Date(iso));
}

/** formatMessageBody indents the JSON body of a message for reading, and returns a body that is no JSON as it is. */
export function formatMessageBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}
