import { describe, expect, it, vi } from 'vitest';

import type { WebhookEvent } from '@/api/types/webhook';
import { useWebhookTexts } from '@/composables/webhook/useWebhookTexts';
import type { WebhookEventOption } from '@/lib/webhook/webhook';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: { status: number }) => (values ? `${key}:${values.status}` : key),
    te: (key: string) => key === 'webhook.errors.timeout',
  }),
}));

const events: WebhookEventOption[] = [
  {
    value: 'item.updated' as WebhookEvent,
    label: 'Item updated',
    description: 'An item changed.',
  },
];

describe('useWebhookTexts', () => {
  describe('eventLabel', () => {
    it('names an event by its option', () => {
      const { eventLabel } = useWebhookTexts(events);

      const label = eventLabel('item.updated');

      expect(label).toBe('Item updated');
    });

    it('translates the test message', () => {
      const { eventLabel } = useWebhookTexts(events);

      const label = eventLabel('webhook.test');

      expect(label).toBe('webhook.test');
    });

    it('passes an unknown event through', () => {
      const { eventLabel } = useWebhookTexts(events);

      const label = eventLabel('item.removed');

      expect(label).toBe('item.removed');
    });
  });

  describe('problem', () => {
    it('describes a delivery without a response by its reason', () => {
      const { problem } = useWebhookTexts(events);

      const text = problem('timeout', 0);

      expect(text).toBe('webhook.errors.timeout');
    });

    it('describes a rejected delivery by the status of the response', () => {
      const { problem } = useWebhookTexts(events);

      const text = problem('', 503);

      expect(text).toBe('webhook.errors.status:503');
    });

    it('falls back to the status for a reason it does not know', () => {
      const { problem } = useWebhookTexts(events);

      const text = problem('unreachable', 0);

      expect(text).toBe('webhook.errors.status:0');
    });
  });
});
