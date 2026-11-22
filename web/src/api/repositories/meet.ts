import { apiFetch } from '@/api/client/client';
import type { MeetResponse } from '@/api/types/meet';

export const meetRepository = {
  /** Returns every event, start entry, and competitor of the loaded meet. */
  async get(): Promise<MeetResponse> {
    return apiFetch('/meet', {
      method: 'GET',
    });
  },

  /** Reads the stored meet files again. */
  async reload(): Promise<void> {
    return apiFetch('/meet/reload', {
      method: 'POST',
    });
  },
};
