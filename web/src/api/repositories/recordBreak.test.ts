import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { recordBreakRepository } from '@/api/repositories/recordBreak';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('recordBreakRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the record breaks', async () => {
      const breaks = [
        {
          id: faker.string.uuid(),
        },
      ];
      fetchMock.mockResolvedValue(jsonResponse(200, breaks));

      const result = await recordBreakRepository.list();

      expect(result).toEqual(breaks);
      expect(lastRequest().url).toBe('/api/record-breaks');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('delete', () => {
    it('removes the record break under its encoded id', async () => {
      const id = faker.string.alphanumeric(8);
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordBreakRepository.delete(`${id} /`);

      expect(lastRequest().url).toBe(`/api/record-breaks/${id}%20%2F`);
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('clear', () => {
    it('removes every record break', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordBreakRepository.clear();

      expect(lastRequest().url).toBe('/api/record-breaks');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });
});
