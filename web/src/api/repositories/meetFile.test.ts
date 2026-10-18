import { describe, expect, it } from 'vitest';

import { meetFileRepository } from '@/api/repositories/meetFile';
import { MeetFile } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('meetFileRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the meet files', async () => {
      fetchMock.mockResolvedValue(jsonResponse(200, []));

      const result = await meetFileRepository.list();

      expect(result).toEqual([]);
      expect(lastRequest().url).toBe('/api/meet/files');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('save', () => {
    it('stores the meet file under its name as form data without a JSON content type', async () => {
      const content = new File(['data'], 'LSTRACE.TXT');
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await meetFileRepository.save(MeetFile.Events, content);

      const { url, init } = lastRequest();
      expect(url).toBe('/api/meet/files/LSTRACE');
      expect(init?.method).toBe('PUT');
      const body = init?.body;
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).get('file')).toBeInstanceOf(File);
      expect(init?.headers).toEqual({});
    });
  });

  describe('upload', () => {
    it('stores every file under the same field', async () => {
      const contents = [new File(['a'], 'LSTRACE.TXT'), new File(['b'], 'LSTSTYLE.TXT')];
      fetchMock.mockResolvedValue(jsonResponse(200, []));

      const result = await meetFileRepository.upload(contents);

      const { url, init } = lastRequest();
      expect(result).toEqual([]);
      expect(url).toBe('/api/meet/files');
      expect(init?.method).toBe('POST');
      const body = init?.body;
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).getAll('files')).toHaveLength(2);
    });
  });
});
