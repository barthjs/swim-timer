import { apiClient, authApiClient } from './client';
import type { Preset } from '@/types';

export const presetsApi = {
  getPresetNames: (): Promise<string[]> => apiClient<string[]>('/presets'),

  getPreset: (name: string): Promise<Preset> => apiClient<Preset>(`/presets/${name}`),

  savePreset: (name: string, data: Preset): Promise<void> =>
    authApiClient<void>(`/presets/${name}`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  deletePreset: (name: string): Promise<void> =>
    authApiClient<void>(`/presets/${name}`, {
      method: 'DELETE',
    }),
};
