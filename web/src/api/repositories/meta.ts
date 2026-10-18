import { apiFetch } from '@/api/client';
import type { MetaResponse } from '@/api/types';

export const metaRepository = {
  /** get returns the public settings of the application. */
  get(): Promise<MetaResponse> {
    return apiFetch('/meta', {
      method: 'GET',
    });
  },
};
