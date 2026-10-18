import { describe, expect, it } from 'vitest';

import { recordingRepository } from '@/api/repositories/recording';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('recordingRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the recordings', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, [
          {
            name: 'recording.jsonl',
            path: 'recordings/recording.jsonl',
            current: true,
          },
        ]),
      );

      const result = await recordingRepository.list();

      expect(result).toEqual([
        {
          name: 'recording.jsonl',
          path: 'recordings/recording.jsonl',
          current: true,
        },
      ]);
      expect(lastRequest().url).toBe('/api/recordings');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('create', () => {
    it('uploads the recording as a file', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(201, {
          name: 'recording.jsonl',
          path: 'recordings/recording.jsonl',
          current: false,
        }),
      );
      const file = new File(['{}'], 'recording.jsonl', {
        type: 'application/jsonl',
      });

      const result = await recordingRepository.create(file);

      expect(result).toEqual({
        name: 'recording.jsonl',
        path: 'recordings/recording.jsonl',
        current: false,
      });
      expect(lastRequest().url).toBe('/api/recordings');
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body;
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).get('file')).toBeInstanceOf(File);
    });
  });

  describe('delete', () => {
    it('removes the recording under its encoded name', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await recordingRepository.delete('a recording.jsonl');

      expect(lastRequest().url).toBe('/api/recordings/a%20recording.jsonl');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('downloadUrl', () => {
    it('returns the download address under the encoded name', () => {
      const result = recordingRepository.downloadUrl('a recording.jsonl');

      expect(result).toBe('/api/recordings/a%20recording.jsonl');
    });
  });

  describe('exportUrl', () => {
    it('returns the address of the table under the encoded name', () => {
      const result = recordingRepository.exportUrl('a recording.jsonl');

      expect(result).toBe('/api/recordings/a%20recording.jsonl/csv');
    });
  });
});
