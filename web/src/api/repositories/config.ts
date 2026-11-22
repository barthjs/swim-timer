import { apiFetch } from '@/api/client/client';
import type {
  ConfigResponse,
  ConfigSaveRequest,
  ConfigTestResponse,
  StorageSettings,
} from '@/api/types/config';

export const configRepository = {
  /** Returns the configuration without the values of its secrets. */
  async get(): Promise<ConfigResponse> {
    return apiFetch('/config', {
      method: 'GET',
    });
  },

  /** Writes the configuration to the config file. */
  async save(payload: ConfigSaveRequest): Promise<ConfigResponse> {
    return apiFetch('/config', {
      method: 'PUT',
      body: payload,
    });
  },

  /** Checks that a storage configuration is reachable, without saving it. */
  async testStorage(settings: StorageSettings): Promise<ConfigTestResponse> {
    return apiFetch('/config/storage/test', {
      method: 'POST',
      body: settings,
    });
  },
};
