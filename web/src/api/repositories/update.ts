import { apiFetch } from '@/api/client/client';
import type { UpdateCheckResponse, UpdateStatusResponse } from '@/api/types/update';

export const updateRepository = {
  /** Compares the running version with the latest release. */
  async check(): Promise<UpdateCheckResponse> {
    return apiFetch('/update', {
      method: 'GET',
    });
  },

  /** Installs the latest release in the background. The application restarts once it is installed. */
  async start(): Promise<void> {
    return apiFetch('/update', {
      method: 'POST',
    });
  },

  /**
   * Returns the state of the last update. A missing session does not lead to the login,
   * since the server forgets the sessions when it restarts after the update.
   */
  async getStatus(): Promise<UpdateStatusResponse> {
    return apiFetch('/update/status', {
      method: 'GET',
      skipAuthRedirect: true,
    });
  },
};
