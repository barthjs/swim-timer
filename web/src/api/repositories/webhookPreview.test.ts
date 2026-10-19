import { describe, expect, it } from 'vitest';

import { webhookPreviewRepository } from '@/api/repositories/webhookPreview';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('webhookPreviewRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('get', () => {
    it('returns the message of the event to the webhook under its encoded id', async () => {
      const preview = {
        method: 'POST',
      };
      fetchMock.mockResolvedValue(jsonResponse(200, preview));

      const result = await webhookPreviewRepository.get('id a/b', 'meet.reloaded');

      expect(result).toEqual(preview);
      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb/preview?event=meet.reloaded');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });
});
