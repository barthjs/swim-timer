import { describe, expect, it } from 'vitest';

import { presetRepository } from '@/api/repositories/preset';
import type { PresetSaveRequest } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('presetRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the presets', async () => {
      fetchMock.mockResolvedValue(jsonResponse(200, ['a', 'b']));

      const result = await presetRepository.list();

      expect(result).toEqual(['a', 'b']);
      expect(lastRequest().url).toBe('/api/presets');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('get', () => {
    it('returns the preset under its encoded name', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          name: 'name a/b',
        }),
      );

      const result = await presetRepository.get('name a/b');

      expect(result).toEqual({
        name: 'name a/b',
      });
      expect(lastRequest().url).toBe('/api/presets/name%20a%2Fb');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('save', () => {
    it('stores the preset under its encoded name', async () => {
      const payload = {
        lanes: [],
      } as unknown as PresetSaveRequest;
      fetchMock.mockResolvedValue(jsonResponse(200, payload));

      const result = await presetRepository.save('name a/b', payload);

      expect(result).toEqual(payload);
      expect(lastRequest().url).toBe('/api/presets/name%20a%2Fb');
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('delete', () => {
    it('removes the preset under its encoded name', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await presetRepository.delete('name a/b');

      expect(lastRequest().url).toBe('/api/presets/name%20a%2Fb');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });
});
