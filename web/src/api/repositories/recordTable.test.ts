import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { recordTableRepository } from '@/api/repositories/recordTable';
import type { ImportMapping } from '@/api/types/import';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

/** Returns a mapping that reads the time from a column of the file. */
function mapping(): ImportMapping {
  return {
    format: {
      encoding: '',
      separator: '',
    },
    columns: {
      time: faker.database.column(),
    },
    values: {},
  };
}

/** Returns a file of records as a browser picks it. */
function csvFile(): File {
  return new File([faker.lorem.lines(2)], faker.system.commonFileName('csv'), {
    type: 'text/csv',
  });
}

/** Returns an id with characters the path has to encode, and the id as the path holds it. */
function fakeId(): { id: string; encoded: string } {
  const word = faker.string.alphanumeric(8);
  return {
    id: `${word} /`,
    encoded: `${word}%20%2F`,
  };
}

describe('recordTableRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the record tables', async () => {
      const list = {
        tables: [
          {
            id: faker.string.uuid(),
          },
        ],
      };
      fetchMock.mockResolvedValue(jsonResponse(200, list));

      const result = await recordTableRepository.list();

      expect(result).toEqual(list);
      expect(lastRequest().url).toBe('/api/record-tables');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('create', () => {
    it('uploads the file with the request as data', async () => {
      const payload = {
        name: faker.lorem.words(2),
        abbreviation: faker.string.alpha(2),
        enabled: faker.datatype.boolean(),
        mapping: mapping(),
      };
      const created = {
        id: faker.string.uuid(),
      };
      fetchMock.mockResolvedValue(jsonResponse(201, created));

      const result = await recordTableRepository.create(csvFile(), payload);

      expect(result).toEqual(created);
      expect(lastRequest().url).toBe('/api/record-tables');
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body as FormData;
      expect(body.get('file')).toBeInstanceOf(File);
      expect(JSON.parse(body.get('data') as string)).toEqual(payload);
    });
  });

  describe('update', () => {
    it('sends the record table as JSON under its encoded id', async () => {
      const { id, encoded } = fakeId();
      const payload = {
        name: faker.lorem.words(2),
        abbreviation: faker.string.alpha(2),
        enabled: faker.datatype.boolean(),
      };
      fetchMock.mockResolvedValue(jsonResponse(200, payload));

      await recordTableRepository.update(id, payload);

      expect(lastRequest().url).toBe(`/api/record-tables/${encoded}`);
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('delete', () => {
    it('removes the record table under its encoded id', async () => {
      const { id, encoded } = fakeId();
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordTableRepository.delete(id);

      expect(lastRequest().url).toBe(`/api/record-tables/${encoded}`);
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('preview', () => {
    it('uploads the file with the format as data', async () => {
      const preview = {
        header: [faker.database.column()],
      };
      fetchMock.mockResolvedValue(jsonResponse(200, preview));

      const result = await recordTableRepository.preview(csvFile(), {
        encoding: 'utf-8',
        separator: '',
      });

      expect(result).toEqual(preview);
      expect(lastRequest().url).toBe('/api/record-tables/preview');
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body as FormData;
      expect(JSON.parse(body.get('data') as string)).toEqual({
        format: {
          encoding: 'utf-8',
          separator: '',
        },
      });
    });

    it('passes the cancel of the caller through', async () => {
      const controller = new AbortController();
      fetchMock.mockResolvedValue(jsonResponse(200, {}));

      await recordTableRepository.preview(
        csvFile(),
        {
          encoding: '',
          separator: '',
        },
        controller.signal,
      );

      controller.abort();
      expect(lastRequest().init?.signal?.aborted).toBe(true);
    });
  });

  describe('import', () => {
    it('uploads the file with the mapping under the encoded id', async () => {
      const { id, encoded } = fakeId();
      const request = {
        mapping: mapping(),
      };
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          id,
        }),
      );

      await recordTableRepository.import(id, csvFile(), request);

      expect(lastRequest().url).toBe(`/api/record-tables/${encoded}/import`);
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body as FormData;
      expect(JSON.parse(body.get('data') as string)).toEqual(request);
    });
  });

  describe('reorder', () => {
    it('sends the ids in their new order', async () => {
      const ids = [faker.string.uuid(), faker.string.uuid()];
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordTableRepository.reorder(ids);

      expect(lastRequest().url).toBe('/api/record-tables/order');
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(
        JSON.stringify({
          ids,
        }),
      );
    });
  });

  describe('reload', () => {
    it('asks the server to read the record tables again', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordTableRepository.reload();

      expect(lastRequest().url).toBe('/api/record-tables/reload');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });

  describe('export', () => {
    it('loads the file with the format and the columns under the encoded id', async () => {
      const { id, encoded } = fakeId();
      const content = faker.lorem.lines(2);
      fetchMock.mockResolvedValue(new Response(content));

      const result = await recordTableRepository.export(id, {
        format: {
          encoding: 'windows-1252',
          separator: 'tab',
        },
        columns: ['time', 'name'],
      });

      expect(await result.text()).toBe(content);
      expect(lastRequest().url).toBe(
        `/api/record-tables/${encoded}/export?encoding=windows-1252&separator=tab&columns=time%2Cname`,
      );
      expect(lastRequest().init?.method).toBe('GET');
    });

    it('leaves out the defaults', async () => {
      const id = faker.string.alphanumeric(8);
      fetchMock.mockResolvedValue(new Response(''));

      await recordTableRepository.export(id, {
        format: {
          encoding: '',
          separator: '',
        },
        columns: [],
      });

      expect(lastRequest().url).toBe(`/api/record-tables/${id}/export`);
    });
  });
});
