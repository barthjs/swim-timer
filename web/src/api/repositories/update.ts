import { apiFetch } from '@/api/client';
import type { UpdateCheckResponse, UpdateStatusResponse } from '@/api/types';

export const updateRepository = {
  /** check compares the running version with the latest release. */
  check(): Promise<UpdateCheckResponse> {
    return apiFetch('/update', {
      method: 'GET',
    });
  },

  /** start installs the latest release in the background. The application restarts once it is installed. */
  start(): Promise<void> {
    return apiFetch('/update', {
      method: 'POST',
    });
  },

  /**
   * getStatus returns the state of the last update. A missing session does not lead to the login,
   * since the server forgets the sessions when it restarts after the update.
   */
  getStatus(): Promise<UpdateStatusResponse> {
    return apiFetch('/update/status', {
      method: 'GET',
      skipAuthRedirect: true,
    });
  },
};
