import { apiFetch } from '@/api/client';
import type { TimingStatusResponse } from '@/api/types';

export const timingRepository = {
  /** getStatus returns the state of the connection to the timing device. */
  getStatus(): Promise<TimingStatusResponse> {
    return apiFetch('/timing/status', {
      method: 'GET',
    });
  },
};
