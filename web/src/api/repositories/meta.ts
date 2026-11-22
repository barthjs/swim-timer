import { apiFetch } from '@/api/client/client';
import type { MetaResponse } from '@/api/types/meta';

export const metaRepository = {
  /** Returns the public settings of the application. */
  async get(): Promise<MetaResponse> {
    return apiFetch('/meta', {
      method: 'GET',
    });
  },
};
