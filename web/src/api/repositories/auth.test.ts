import { afterEach, describe, expect, it } from 'vitest';

import { setAuthenticationRequiredHandler } from '@/api/client';
import { authRepository } from '@/api/repositories/auth';
import type { AuthLoginRequest } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('authRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  afterEach(() => {
    setAuthenticationRequiredHandler(() => {});
  });

  describe('getStatus', () => {
    it('returns the session status', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          authenticated: true,
        }),
      );

      const status = await authRepository.getStatus();

      expect(lastRequest().url).toBe('/api/auth/me');
      expect(lastRequest().init?.method).toBe('GET');
      expect(status).toEqual({
        authenticated: true,
      });
    });

    it('skips the login redirect for a guest', async () => {
      let redirected = false;
      setAuthenticationRequiredHandler(() => {
        redirected = true;
      });
      fetchMock.mockResolvedValue(
        jsonResponse(401, {
          code: 'authentication_required',
          message: 'login',
        }),
      );

      const request = authRepository.getStatus();

      await expect(request).rejects.toThrow('login');
      expect(redirected).toBe(false);
    });
  });

  describe('login', () => {
    it('sends the credentials', async () => {
      const payload: AuthLoginRequest = {
        password: 'password1',
      };
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await authRepository.login(payload);

      expect(lastRequest().url).toBe('/api/auth/login');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('logout', () => {
    it('ends the session', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await authRepository.logout();

      expect(lastRequest().url).toBe('/api/auth/logout');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });
});
