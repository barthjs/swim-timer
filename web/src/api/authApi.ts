import { authApiClient } from '@/api/client.ts';

export const authApi = {
  login: (key: string): Promise<void> =>
    authApiClient<void>(`/auth/login`, {
      method: 'POST',
      body: JSON.stringify({ key: key }),
    }),
};
