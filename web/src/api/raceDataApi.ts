import { authApiClient } from '@/api/client.ts';
import type { RaceDataUploadResult } from '@/types';

export const raceDataApi = {
  getStatus: (): Promise<Record<string, boolean>> =>
    authApiClient<Record<string, boolean>>('/race-data'),

  async uploadFile(fileId: string, file: File): Promise<void> {
    const formData = new FormData();
    formData.append('file', file);

    await authApiClient(`/race-data/${fileId}`, {
      method: 'POST',
      body: formData,
    });
  },

  async uploadFiles(files: File[]): Promise<RaceDataUploadResult[]> {
    const formData = new FormData();
    files.forEach((file) => {
      formData.append('files', file);
    });

    return authApiClient<RaceDataUploadResult[]>('/race-data', {
      method: 'POST',
      body: formData,
    });
  },

  reloadRaceData: (): Promise<void> =>
    authApiClient<void>('/actions/reload-race-data', { method: 'POST' }),
};
