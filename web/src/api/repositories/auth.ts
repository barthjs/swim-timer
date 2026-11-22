import { apiFetch } from '@/api/client/client';
import type { AuthLoginRequest, AuthStatusResponse } from '@/api/types/auth';

export const authRepository = {
  /** Returns whether the current session is logged in. */
  async getStatus(): Promise<AuthStatusResponse> {
    return apiFetch('/auth/me', {
      method: 'GET',
      skipAuthRedirect: true,
    });
  },

  /** Starts an admin session. */
  async login(payload: AuthLoginRequest): Promise<void> {
    return apiFetch('/auth/login', {
      method: 'POST',
      body: payload,
    });
  },

  /** Ends the current session. */
  async logout(): Promise<void> {
    return apiFetch('/auth/logout', {
      method: 'POST',
    });
  },
};
