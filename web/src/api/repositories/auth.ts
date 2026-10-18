import { apiFetch } from '@/api/client';
import type { AuthLoginRequest, AuthStatusResponse } from '@/api/types';

export const authRepository = {
  /** getStatus returns whether the current session is logged in. */
  getStatus(): Promise<AuthStatusResponse> {
    return apiFetch('/auth/me', {
      method: 'GET',
      skipAuthRedirect: true,
    });
  },

  /** login starts an admin session. */
  login(payload: AuthLoginRequest): Promise<void> {
    return apiFetch('/auth/login', {
      method: 'POST',
      body: payload,
    });
  },

  /** logout ends the current session. */
  logout(): Promise<void> {
    return apiFetch('/auth/logout', {
      method: 'POST',
    });
  },
};
