import { apiFetch } from '@/api/client';
import type { PresetResponse, PresetSaveRequest } from '@/api/types';

export const presetRepository = {
  /** list returns the names of every preset. */
  list(): Promise<string[]> {
    return apiFetch('/presets', {
      method: 'GET',
    });
  },

  /** get returns the preset with the given name. */
  get(name: string): Promise<PresetResponse> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'GET',
    });
  },

  /** save creates or replaces the preset with the given name. */
  save(name: string, payload: PresetSaveRequest): Promise<PresetResponse> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** delete removes the preset with the given name. */
  delete(name: string): Promise<void> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },
};
