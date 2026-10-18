import { createPinia, setActivePinia } from 'pinia';
import type { RouteLocationNormalized, RouteMeta } from 'vue-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { authGuard } from '@/router/guard';
import { authRepository } from '@/api/repositories/auth';

vi.mock('@/api/repositories/auth', () => ({
  authRepository: {
    getStatus: vi.fn<typeof authRepository.getStatus>(),
    login: vi.fn<typeof authRepository.login>(),
    logout: vi.fn<typeof authRepository.logout>(),
  },
}));

/** route returns a target location with the given path and meta fields. */
function route(fullPath: string, meta: RouteMeta): RouteLocationNormalized {
  return {
    fullPath,
    meta,
  } as RouteLocationNormalized;
}

describe('authGuard', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.mocked(authRepository.getStatus).mockReset();
  });

  it('lets a logged-in admin open protected pages', async () => {
    vi.mocked(authRepository.getStatus).mockResolvedValue({
      authenticated: true,
    });

    const result = await authGuard(
      route('/admin', {
        requiresAuth: true,
      }),
    );

    expect(result).toBe(true);
  });

  it('lets public pages through without asking for the session', async () => {
    const result = await authGuard(route('/public', {}));

    expect(result).toBe(true);
    expect(authRepository.getStatus).not.toHaveBeenCalled();
  });

  it('shows the login page to a guest', async () => {
    vi.mocked(authRepository.getStatus).mockResolvedValue({
      authenticated: false,
    });

    const result = await authGuard(
      route('/login', {
        requiresGuest: true,
      }),
    );

    expect(result).toBe(true);
  });

  it('sends a guest to the login page and back afterward', async () => {
    vi.mocked(authRepository.getStatus).mockResolvedValue({
      authenticated: false,
    });

    const result = await authGuard(
      route('/admin?page=2', {
        requiresAuth: true,
      }),
    );

    expect(result).toEqual({
      name: 'login',
      query: {
        redirect: '/admin?page=2',
      },
    });
  });

  it('sends a logged-in admin from the login page to the dashboard', async () => {
    vi.mocked(authRepository.getStatus).mockResolvedValue({
      authenticated: true,
    });

    const result = await authGuard(
      route('/login', {
        requiresGuest: true,
      }),
    );

    expect(result).toEqual({
      name: 'dashboard',
    });
  });

  it('lets the navigation through while the session is unknown', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(authRepository.getStatus).mockRejectedValue(new Error('offline'));

    const result = await authGuard(
      route('/admin/other', {
        requiresAuth: true,
      }),
    );

    expect(result).toBe(true);
  });

  it('asks for the session only once', async () => {
    vi.mocked(authRepository.getStatus).mockResolvedValue({
      authenticated: true,
    });
    await authGuard(
      route('/admin', {
        requiresAuth: true,
      }),
    );

    await authGuard(
      route('/admin/other', {
        requiresAuth: true,
      }),
    );

    expect(authRepository.getStatus).toHaveBeenCalledTimes(1);
  });
});
