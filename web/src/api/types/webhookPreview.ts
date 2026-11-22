export type WebhookHeaderResponse = {
  // The name of the header in lowercase
  name: string;
  value: string;
};

/** A message to a webhook as it would be sent now. The headers are sorted by name. */
export type WebhookPreviewResponse = {
  method: string;
  url: string;
  headers: WebhookHeaderResponse[];
  body: string;
};
