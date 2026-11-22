import { defineStore } from 'pinia';
import { ref } from 'vue';

import { ApiError } from '@/api/client/errors';
import { authRepository } from '@/api/repositories/auth';

/** Holds whether the browser has an authenticated session, loaded once on demand. */
export const useAuthStore = defineStore('auth', () => {
  const authenticated = ref<boolean | null>(null);
  const isInitialized = ref(false);

  // Concurrent refreshes share one request
  let inflight: Promise<void> | null = null;

  /** Fetches the session status from the server, also when it is already loaded. */
  async function refresh(): Promise<void> {
    if (inflight) return inflight;

    inflight = (async () => {
      try {
        const status = await authRepository.getStatus();
        authenticated.value = status.authenticated;
        isInitialized.value = true;
      } catch (err) {
        // Only a 401 means that there is no session, any other failure keeps the known state
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

  /** Fetches the session status the first time it is called. */
  async function ensureLoaded(): Promise<void> {
    if (isInitialized.value) return;
    return refresh();
  }

  /** Forgets the session without asking the server. */
  function clearSession(): void {
    authenticated.value = false;
    isInitialized.value = true;
  }

  /** Ends the session at the server and forgets it, even if the request fails. */
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
