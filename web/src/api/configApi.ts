import { authApiClient } from '@/api/client';

import type { Config } from '@/types';

export const configApi = {
  getConfig: (): Promise<Config> => authApiClient<Config>('/config'),

  updateConfig: (config: Config): Promise<Config> =>
    authApiClient<Config>('/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    }),
};
