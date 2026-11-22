import { apiFetch } from '@/api/client/client';
import type { RecordListResponse } from '@/api/types/record';

export const recordRepository = {
  /** Returns every record of every record table. */
  async list(): Promise<RecordListResponse> {
    return apiFetch('/records', {
      method: 'GET',
    });
  },
};
