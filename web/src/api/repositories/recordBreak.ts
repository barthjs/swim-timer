import { apiFetch } from '@/api/client/client';
import type { RecordBreakResponse } from '@/api/types/recordBreak';

export const recordBreakRepository = {
  /** Returns every record set at the meet, the most recent first. */
  async list(): Promise<RecordBreakResponse[]> {
    return apiFetch('/record-breaks', {
      method: 'GET',
    });
  },

  /** Removes the record break with the given id. */
  async delete(id: string): Promise<void> {
    return apiFetch(`/record-breaks/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },

  /** Removes every record break. */
  async clear(): Promise<void> {
    return apiFetch('/record-breaks', {
      method: 'DELETE',
    });
  },
};
