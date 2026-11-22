/** How a delivery to a webhook ended. */
export const WebhookDeliveryOutcome = {
  Delivered: 'delivered',
  Retry: 'retry',
  Deferred: 'deferred',
  Gone: 'gone',
  Rejected: 'rejected',
} as const;
export type WebhookDeliveryOutcome =
  (typeof WebhookDeliveryOutcome)[keyof typeof WebhookDeliveryOutcome];

/** Names why a delivery to a webhook got no response. */
export const WebhookDeliveryErrorCode = {
  BlockedAddress: 'blocked_address',
  InvalidUrl: 'invalid_url',
  InvalidSecret: 'invalid_secret',
  UntrustedCertificate: 'untrusted_certificate',
  Timeout: 'timeout',
  Unreachable: 'unreachable',
} as const;
export type WebhookDeliveryErrorCode =
  (typeof WebhookDeliveryErrorCode)[keyof typeof WebhookDeliveryErrorCode];

export type WebhookDeliveryResponse = {
  messageId: string;
  event: string;
  attemptedAt: string;
  outcome: WebhookDeliveryOutcome;
  // The status of the response, 0 if none was received
  statusCode: number;
  durationMs: number;
  // The start of the response body
  responseBody: string;
  // Why a delivery got no response, with the technical details in error. Both are empty if the target answered.
  errorCode: WebhookDeliveryErrorCode | '';
  error: string;
};
