import { defineStore } from 'pinia';
import { ref } from 'vue';

import { ApiError } from '@/api/errors';
import { authRepository } from '@/api/repositories/auth';

/**
 * useAuthStore holds whether the browser has an authenticated session. The status is loaded once on demand
 * and refreshed after a login. The status stays unknown while it could not be loaded.
 */
export const useAuthStore = defineStore('auth', () => {
  const authenticated = ref<boolean | null>(null);
  const isInitialized = ref(false);

  // Concurrent refreshes share one request
  let inflight: Promise<void> | null = null;

  /**
   * refresh fetches the current session status from the server, even if it is already loaded.
   * Only a 401 means that there is no session. Any other failure, such as a network error, a timeout,
   * or a server error, says nothing about the session: the known status stays,
   * and a store that has not been initialized yet tries again on the next call of ensureLoaded.
   */
  async function refresh(): Promise<void> {
    if (inflight) return inflight;

    inflight = (async () => {
      try {
        const status = await authRepository.getStatus();
        authenticated.value = status.authenticated;
        isInitialized.value = true;
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          authenticated.value = false;
          isInitialized.value = true;
          return;
        }
        console.error('[auth] loading the session status failed', err);
      } finally {
        inflight = null;
      }
    })();

    return inflight;
  }

  /** ensureLoaded fetches the session status the first time it is called. */
  async function ensureLoaded(): Promise<void> {
    if (isInitialized.value) return;
    return refresh();
  }

  /** clearSession forgets the session without asking the server. */
  function clearSession(): void {
    authenticated.value = false;
    isInitialized.value = true;
  }

  /** logout ends the session at the server and forgets it, even if the request fails. */
  async function logout(): Promise<void> {
    try {
      await authRepository.logout();
    } finally {
      clearSession();
    }
  }

  return {
    authenticated,
    isInitialized,
    ensureLoaded,
    refresh,
    clearSession,
    logout,
  };
});
