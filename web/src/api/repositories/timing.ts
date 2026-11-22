import { apiFetch } from '@/api/client/client';
import type { TimingStatusResponse } from '@/api/types/timing';

export const timingRepository = {
  /** Returns the state of the connection to the timing device. */
  async getStatus(): Promise<TimingStatusResponse> {
    return apiFetch('/timing/status', {
      method: 'GET',
    });
  },
};
