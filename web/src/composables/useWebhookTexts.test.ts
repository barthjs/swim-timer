import { describe, expect, it, vi } from 'vitest';

import { useWebhookTexts } from '@/composables/useWebhookTexts';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: { status: number }) => (values ? `${key}:${values.status}` : key),
  }),
}));

describe('useWebhookTexts', () => {
  describe('eventLabel', () => {
    it('translates a known event, including the test message', () => {
      const { eventLabel } = useWebhookTexts();

      const label = eventLabel('webhook.test');

      expect(label).toBe('pages.config.webhook.events.test');
    });

    it('passes an unknown event through', () => {
      const { eventLabel } = useWebhookTexts();

      const label = eventLabel('heat.started');

      expect(label).toBe('heat.started');
    });
  });

  describe('problem', () => {
    it('describes a delivery without a response by its reason', () => {
      const { problem } = useWebhookTexts();

      const text = problem('timeout', 0);

      expect(text).toBe('pages.config.webhook.errors.timeout');
    });

    it('describes a rejected delivery by the status of the response', () => {
      const { problem } = useWebhookTexts();

      const text = problem('', 503);

      expect(text).toBe('pages.config.webhook.errors.status:503');
    });
  });
});
