import { apiFetch } from '@/api/client/client';
import type { PresetResponse, PresetSaveRequest } from '@/api/types/preset';

export const presetRepository = {
  /** Returns the names of every preset. */
  async list(): Promise<string[]> {
    return apiFetch('/presets', {
      method: 'GET',
    });
  },

  /** Returns the preset with the given name. */
  async get(name: string): Promise<PresetResponse> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'GET',
    });
  },

  /** Creates or replaces the preset with the given name. */
  async save(name: string, payload: PresetSaveRequest): Promise<PresetResponse> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** Removes the preset with the given name. */
  async delete(name: string): Promise<void> {
    return apiFetch(`/presets/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },
};
