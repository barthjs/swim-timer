import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client';
import type { MeetFile, MeetFileResponse, MeetFileUploadResponse } from '@/api/types';

export const meetFileRepository = {
  /** list returns every meet file and whether it is stored. */
  list(): Promise<MeetFileResponse[]> {
    return apiFetch('/meet/files', {
      method: 'GET',
    });
  },

  /** save stores the content of one meet file. */
  save(file: MeetFile, content: File): Promise<void> {
    const body = new FormData();
    body.append('file', content);
    return apiFetch(`/meet/files/${encodeURIComponent(file)}`, {
      method: 'PUT',
      body,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** upload stores every file whose name matches a meet file and reports each one. */
  upload(contents: File[]): Promise<MeetFileUploadResponse[]> {
    const body = new FormData();
    for (const content of contents) {
      body.append('files', content);
    }
    return apiFetch('/meet/files', {
      method: 'POST',
      body,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },
};
