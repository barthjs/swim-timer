import { describe, expect, it } from 'vitest';

import { configRepository } from '@/api/repositories/config';
import type { ConfigSaveRequest, StorageSettings } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('configRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('get', () => {
    it('returns the configuration', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          secrets: {},
        }),
      );

      const config = await configRepository.get();

      expect(lastRequest().url).toBe('/api/config');
      expect(lastRequest().init?.method).toBe('GET');
      expect(config).toEqual({
        secrets: {},
      });
    });
  });

  describe('save', () => {
    it('replaces the configuration with the secrets to clear', async () => {
      const payload = {
        clearSecrets: ['app.secret'],
      } as unknown as ConfigSaveRequest;
      fetchMock.mockResolvedValue(jsonResponse(200, {}));

      const config = await configRepository.save(payload);

      expect(lastRequest().url).toBe('/api/config');
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
      expect(config).toEqual({});
    });
  });

  describe('testStorage', () => {
    it('returns the result of the storage test', async () => {
      const settings = {
        driver: 'local',
        path: 'data',
      } as unknown as StorageSettings;
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          ok: true,
        }),
      );

      const result = await configRepository.testStorage(settings);

      expect(lastRequest().url).toBe('/api/config/storage/test');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(settings));
      expect(result).toEqual({
        ok: true,
      });
    });
  });
});
