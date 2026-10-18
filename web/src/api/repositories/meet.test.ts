import { describe, expect, it } from 'vitest';

import { meetRepository } from '@/api/repositories/meet';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('meetRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('get', () => {
    it('returns the meet', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          events: [],
        }),
      );

      const result = await meetRepository.get();

      expect(result).toEqual({
        events: [],
      });
      expect(lastRequest().url).toBe('/api/meet');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('reload', () => {
    it('reloads the meet files', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await meetRepository.reload();

      expect(lastRequest().url).toBe('/api/meet/reload');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });
});
