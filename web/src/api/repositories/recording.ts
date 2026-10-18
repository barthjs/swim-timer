import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client';
import type { RecordingResponse } from '@/api/types';

export const recordingRepository = {
  /** list returns every stored recording. */
  list(): Promise<RecordingResponse[]> {
    return apiFetch('/recordings', {
      method: 'GET',
    });
  },

  /** create uploads a recording. */
  create(file: File): Promise<RecordingResponse> {
    const form = new FormData();
    form.append('file', file);
    return apiFetch('/recordings', {
      method: 'POST',
      body: form,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** delete removes the recording with the given name. */
  delete(name: string): Promise<void> {
    return apiFetch(`/recordings/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },

  /** downloadUrl returns the address that downloads the recording as it is stored. */
  downloadUrl(name: string): string {
    return `/api/recordings/${encodeURIComponent(name)}`;
  },

  /** exportUrl returns the address that downloads the recording as a table with the decoded frames. */
  exportUrl(name: string): string {
    return `/api/recordings/${encodeURIComponent(name)}/csv`;
  },
};
