import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError } from '@/api/errors';
import { authRepository } from '@/api/repositories/auth';
import { useAuthStore } from '@/stores/auth';

vi.mock('@/api/repositories/auth', () => ({
  authRepository: {
    getStatus: vi.fn<typeof authRepository.getStatus>(),
    login: vi.fn<typeof authRepository.login>(),
    logout: vi.fn<typeof authRepository.logout>(),
  },
}));

const getStatus = vi.mocked(authRepository.getStatus);
const logout = vi.mocked(authRepository.logout);

/** failures are the errors of a session request that leave the session unknown. */
const failures = [
  {
    name: 'a server error',
    error: new ApiError({
      message: 'internal error',
      status: 500,
      code: 'internal_error',
    }),
  },
  {
    name: 'a network error',
    error: new TypeError('Failed to fetch'),
  },
];

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    getStatus.mockReset();
    logout.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('refresh', () => {
    it('shares one request between concurrent refreshes', async () => {
      getStatus.mockResolvedValue({
        authenticated: true,
      });
      const store = useAuthStore();

      await Promise.all([store.refresh(), store.ensureLoaded(), store.refresh()]);

      expect(getStatus).toHaveBeenCalledOnce();
    });

    it('treats a 401 as logged out', async () => {
      getStatus.mockRejectedValue(
        new ApiError({
          message: 'authentication required',
          status: 401,
          code: 'authentication_required',
        }),
      );
      const store = useAuthStore();

      await store.refresh();

      expect(store.authenticated).toBe(false);
      expect(store.isInitialized).toBe(true);
    });

    it('keeps a known session after a failure other than 401', async () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      getStatus.mockResolvedValueOnce({
        authenticated: true,
      });
      getStatus.mockRejectedValueOnce(new Error('offline'));
      const store = useAuthStore();
      await store.refresh();

      await store.refresh();

      expect(store.authenticated).toBe(true);
      expect(store.isInitialized).toBe(true);
    });
  });

  describe('ensureLoaded', () => {
    it('loads the session once', async () => {
      getStatus.mockResolvedValue({
        authenticated: true,
      });
      const store = useAuthStore();
      await store.ensureLoaded();

      await store.ensureLoaded();

      expect(store.authenticated).toBe(true);
      expect(store.isInitialized).toBe(true);
      expect(getStatus).toHaveBeenCalledOnce();
    });

    it.each(failures)('keeps the session unknown after $name', async ({ error }) => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      getStatus.mockRejectedValueOnce(error);
      const store = useAuthStore();

      await store.ensureLoaded();

      expect(store.authenticated).toBeNull();
      expect(store.isInitialized).toBe(false);
    });

    it.each(failures)('retries on the next call after $name', async ({ error }) => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      getStatus.mockRejectedValueOnce(error);
      getStatus.mockResolvedValueOnce({
        authenticated: true,
      });
      const store = useAuthStore();
      await store.ensureLoaded();

      await store.ensureLoaded();

      expect(store.authenticated).toBe(true);
      expect(store.isInitialized).toBe(true);
    });
  });

  describe('clearSession', () => {
    it('clears the session without a request', async () => {
      getStatus.mockResolvedValue({
        authenticated: true,
      });
      const store = useAuthStore();
      await store.refresh();

      store.clearSession();

      expect(store.authenticated).toBe(false);
      expect(store.isInitialized).toBe(true);
      expect(logout).not.toHaveBeenCalled();
    });
  });

  describe('logout', () => {
    it('clears the session even if the request fails', async () => {
      getStatus.mockResolvedValue({
        authenticated: true,
      });
      logout.mockRejectedValue(new Error('offline'));
      const store = useAuthStore();
      await store.refresh();

      const request = store.logout();

      await expect(request).rejects.toThrow('offline');
      expect(logout).toHaveBeenCalledOnce();
      expect(store.authenticated).toBe(false);
      expect(store.isInitialized).toBe(true);
    });
  });
});
