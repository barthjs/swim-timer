import { describe, expect, it } from 'vitest';

import { metaRepository } from '@/api/repositories/meta';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('metaRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('get', () => {
    it('returns the public settings', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          version: '1.0.0',
        }),
      );

      const meta = await metaRepository.get();

      expect(lastRequest().url).toBe('/api/meta');
      expect(lastRequest().init?.method).toBe('GET');
      expect(meta).toEqual({
        version: '1.0.0',
      });
    });
  });
});
