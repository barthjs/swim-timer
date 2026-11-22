import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { recordRepository } from '@/api/repositories/record';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('recordRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the records of every table', async () => {
      const list = {
        records: [
          {
            tableId: faker.string.uuid(),
          },
        ],
      };
      fetchMock.mockResolvedValue(jsonResponse(200, list));

      const result = await recordRepository.list();

      expect(result).toEqual(list);
      expect(lastRequest().url).toBe('/api/records');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });
});
