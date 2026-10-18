import { apiFetch } from '@/api/client';
import type { MeetResponse } from '@/api/types';

export const meetRepository = {
  /** get returns every event, start entry, and competitor of the loaded meet. */
  get(): Promise<MeetResponse> {
    return apiFetch('/meet', {
      method: 'GET',
    });
  },

  /** reload reads the stored meet files again. */
  reload(): Promise<void> {
    return apiFetch('/meet/reload', {
      method: 'POST',
    });
  },
};
