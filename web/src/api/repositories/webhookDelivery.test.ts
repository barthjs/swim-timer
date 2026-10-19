import { describe, expect, it } from 'vitest';

import { webhookDeliveryRepository } from '@/api/repositories/webhookDelivery';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('webhookDeliveryRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the deliveries of the webhook under its encoded id', async () => {
      const deliveries = [
        {
          messageId: 'msg_a',
        },
      ];
      fetchMock.mockResolvedValue(jsonResponse(200, deliveries));

      const result = await webhookDeliveryRepository.list('id a/b');

      expect(result).toEqual(deliveries);
      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb/deliveries');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });
});
