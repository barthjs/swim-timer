import { describe, expect, it } from 'vitest';

import { updateRepository } from '@/api/repositories/update';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('updateRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('check', () => {
    it('returns the latest release', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          newer: false,
        }),
      );

      const release = await updateRepository.check();

      expect(lastRequest().url).toBe('/api/update');
      expect(lastRequest().init?.method).toBe('GET');
      expect(release).toEqual({
        newer: false,
      });
    });
  });

  describe('start', () => {
    it('starts the update', async () => {
      fetchMock.mockResolvedValue(jsonResponse(202, undefined));

      await updateRepository.start();

      expect(lastRequest().url).toBe('/api/update');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });

  describe('getStatus', () => {
    it('returns the state of the last update', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          state: 'idle',
        }),
      );

      const status = await updateRepository.getStatus();

      expect(lastRequest().url).toBe('/api/update/status');
      expect(lastRequest().init?.method).toBe('GET');
      expect(status).toEqual({
        state: 'idle',
      });
    });
  });
});
