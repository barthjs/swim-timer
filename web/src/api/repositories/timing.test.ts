import { describe, expect, it } from 'vitest';

import { timingRepository } from '@/api/repositories/timing';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('timingRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('getStatus', () => {
    it('returns the status of the timing device', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          connected: true,
        }),
      );

      const result = await timingRepository.getStatus();

      expect(result).toEqual({
        connected: true,
      });
      expect(lastRequest().url).toBe('/api/timing/status');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });
});
