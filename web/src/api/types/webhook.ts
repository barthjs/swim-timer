import type { WebhookDeliveryErrorCode } from '@/api/types/webhookDelivery';

/** Names what a webhook message announces. */
export const WebhookEvent = {
  HeatUpdated: 'heat.updated',
  MeetReloaded: 'meet.reloaded',
  RecordSet: 'record.set',
} as const;
export type WebhookEvent = (typeof WebhookEvent)[keyof typeof WebhookEvent];

/**
 * The state of the deliveries to a webhook since the start of the application.
 * The times are missing before the first delivery of their kind, and lastStatusCode and lastErrorCode describe
 * the last failed delivery by the status of its response or by the reason it got none.
 */
export type WebhookStatusResponse = {
  lastDeliveredAt?: string;
  lastFailedAt?: string;
  lastStatusCode?: number;
  lastErrorCode?: WebhookDeliveryErrorCode;
  consecutiveFailures: number;
};

/** A new webhook. An empty secret is generated. */
export type WebhookCreateRequest = {
  name: string;
  url: string;
  events: WebhookEvent[];
  enabled: boolean;
  secret: string;
  // Accepts any certificate of the endpoint, such as a self-signed one
  skipTlsVerify: boolean;
};

/** Replaces the settings of a webhook. An empty secret keeps the stored one. */
export type WebhookUpdateRequest = {
  name: string;
  url: string;
  events: WebhookEvent[];
  enabled: boolean;
  secret: string;
  // Accepts any certificate of the endpoint, such as a self-signed one
  skipTlsVerify: boolean;
};

export type WebhookResponse = {
  id: string;
  name: string;
  url: string;
  events: WebhookEvent[];
  enabled: boolean;
  // Only set in the response that created the webhook, since the secret is never shown again
  secret?: string;
  // Accepts any certificate of the endpoint, such as a self-signed one
  skipTlsVerify: boolean;
  status: WebhookStatusResponse;
};
