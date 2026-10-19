import { describe, expect, it } from 'vitest';

import {
  formatMessageBody,
  formatWebhookTime,
  generateWebhookSecret,
  matchesDeliverySearch,
  matchesWebhookSearch,
  webhookHealth,
} from '@/lib/webhook';

describe('webhookHealth', () => {
  it('reports a delivered webhook as ok', () => {
    const webhook = {
      enabled: true,
      status: {
        lastDeliveredAt: '2026-10-19T10:00:00Z',
        consecutiveFailures: 0,
      },
    };

    const health = webhookHealth(webhook);

    expect(health).toBe('ok');
  });

  it('reports a webhook without deliveries as waiting', () => {
    const webhook = {
      enabled: true,
      status: {
        consecutiveFailures: 0,
      },
    };

    const health = webhookHealth(webhook);

    expect(health).toBe('waiting');
  });

  it('reports a failed delivery as failing, even after an earlier success', () => {
    const webhook = {
      enabled: true,
      status: {
        lastDeliveredAt: '2026-10-19T10:00:00Z',
        lastFailedAt: '2026-10-19T10:01:00Z',
        lastStatusCode: 500,
        consecutiveFailures: 2,
      },
    };

    const health = webhookHealth(webhook);

    expect(health).toBe('failing');
  });

  it('reports a disabled webhook as disabled', () => {
    const webhook = {
      enabled: false,
      status: {
        lastStatusCode: 500,
        consecutiveFailures: 1,
      },
    };

    const health = webhookHealth(webhook);

    expect(health).toBe('disabled');
  });
});

describe('matchesWebhookSearch', () => {
  const webhook = {
    name: 'Club Website',
    url: 'https://example.com/live',
  };

  it('matches the name without letter case', () => {
    const matches = matchesWebhookSearch(webhook, ' club web ');

    expect(matches).toBe(true);
  });

  it('matches the url', () => {
    const matches = matchesWebhookSearch(webhook, 'example.com');

    expect(matches).toBe(true);
  });

  it('matches every webhook without a search', () => {
    const matches = matchesWebhookSearch(webhook, '');

    expect(matches).toBe(true);
  });

  it('rejects a webhook that contains the search nowhere', () => {
    const matches = matchesWebhookSearch(webhook, 'results');

    expect(matches).toBe(false);
  });
});

describe('matchesDeliverySearch', () => {
  const delivery = {
    messageId: 'msg_a',
    event: 'heat.updated',
    attemptedAt: '2026-10-19T10:15:30Z',
    outcome: 'retry' as const,
    statusCode: 503,
    durationMs: 12,
    responseBody: 'Service Unavailable',
    errorCode: '' as const,
    error: '',
  };

  it('matches the answer without letter case', () => {
    const matches = matchesDeliverySearch(delivery, 'unavailable');

    expect(matches).toBe(true);
  });

  it('matches the status', () => {
    const matches = matchesDeliverySearch(delivery, '503');

    expect(matches).toBe(true);
  });

  it('matches every delivery without a search', () => {
    const matches = matchesDeliverySearch(delivery, ' ');

    expect(matches).toBe(true);
  });

  it('rejects a delivery that contains the search nowhere', () => {
    const matches = matchesDeliverySearch(delivery, 'meet');

    expect(matches).toBe(false);
  });
});

describe('generateWebhookSecret', () => {
  it('returns 32 random bytes in base64 after the prefix', () => {
    const secret = generateWebhookSecret();

    expect(secret.startsWith('whsec_')).toBe(true);
    expect(atob(secret.slice('whsec_'.length))).toHaveLength(32);
  });

  it('returns a new secret every time', () => {
    const first = generateWebhookSecret();

    const second = generateWebhookSecret();

    expect(second).not.toBe(first);
  });
});

describe('formatWebhookTime', () => {
  it('formats the date and the time with seconds', () => {
    const formatted = formatWebhookTime('2026-10-19T10:15:30Z', 'en-GB');

    expect(formatted).toContain('19/10/2026');
    expect(formatted).toMatch(/\d{2}:15:30/);
  });
});

describe('formatMessageBody', () => {
  it('indents a JSON body', () => {
    const formatted = formatMessageBody('{"type":"meet.reloaded","data":{"events":[]}}');

    expect(formatted).toBe('{\n  "type": "meet.reloaded",\n  "data": {\n    "events": []\n  }\n}');
  });

  it('returns a body that is no JSON as it is', () => {
    const formatted = formatMessageBody('payload=1');

    expect(formatted).toBe('payload=1');
  });
});
