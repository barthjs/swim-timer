import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client/client';
import type { RecordingResponse } from '@/api/types/recording';

export const recordingRepository = {
  /** Returns every stored recording. */
  async list(): Promise<RecordingResponse[]> {
    return apiFetch('/recordings', {
      method: 'GET',
    });
  },

  /** Uploads a recording. */
  async create(file: File): Promise<RecordingResponse> {
    const form = new FormData();
    form.append('file', file);
    return apiFetch('/recordings', {
      method: 'POST',
      body: form,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** Removes the recording with the given name. */
  async delete(name: string): Promise<void> {
    return apiFetch(`/recordings/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },

  /** Returns the address that downloads the recording as it is stored. */
  downloadUrl(name: string): string {
    return `/api/recordings/${encodeURIComponent(name)}`;
  },

  /** Returns the address that downloads the recording as a table with the decoded frames. */
  exportUrl(name: string): string {
    return `/api/recordings/${encodeURIComponent(name)}/csv`;
  },
};
