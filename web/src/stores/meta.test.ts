import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { metaRepository } from '@/api/repositories/meta';
import type { MetaResponse } from '@/api/types';
import { useMetaStore } from '@/stores/meta';

const get = vi.hoisted(() => vi.fn<typeof metaRepository.get>());

vi.mock('@/api/repositories/meta', () => ({
  metaRepository: {
    get,
  },
}));

const meta: MetaResponse = {
  name: 'Swim-Timer',
  version: '1.0.0',
  language: 'de',
  theme: {
    primary: 'blue',
    secondary: 'sky',
    success: 'green',
    info: 'blue',
    warning: 'yellow',
    error: 'red',
    neutral: 'slate',
  },
};

describe('useMetaStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    get.mockReset();
  });

  describe('load', () => {
    it('returns the metadata once and caches it', async () => {
      get.mockResolvedValue(meta);
      const store = useMetaStore();

      const [first, second] = await Promise.all([store.load(), store.load()]);
      const third = await store.load();

      expect(first).toEqual(meta);
      expect(second).toBe(first);
      expect(third).toEqual(first);
      expect(store.meta).toEqual(meta);
      expect(get).toHaveBeenCalledOnce();
    });

    it('retries after a failed load', async () => {
      get.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(meta);
      const store = useMetaStore();
      await expect(store.load()).rejects.toThrow('offline');

      const result = await store.load();

      expect(result).toEqual(meta);
      expect(get).toHaveBeenCalledTimes(2);
    });
  });

  describe('reload', () => {
    it('replaces the cached metadata', async () => {
      const changed: MetaResponse = {
        ...meta,
        language: 'en',
      };
      get.mockResolvedValueOnce(meta).mockResolvedValueOnce(changed);
      const store = useMetaStore();
      await store.load();

      const result = await store.reload();

      expect(result).toEqual(changed);
      expect(store.meta).toEqual(changed);
      await expect(store.load()).resolves.toEqual(changed);
      expect(get).toHaveBeenCalledTimes(2);
    });

    it('keeps only the result of the latest request', async () => {
      const changed: MetaResponse = {
        ...meta,
        language: 'en',
      };
      let resolveFirst: (value: MetaResponse) => void = () => {};
      get
        .mockReturnValueOnce(
          new Promise((resolve) => {
            resolveFirst = resolve;
          }),
        )
        .mockResolvedValueOnce(changed);
      const store = useMetaStore();
      const first = store.load();

      await store.reload();
      resolveFirst(meta);
      await first;

      expect(store.meta).toEqual(changed);
    });

    it('keeps the result of an earlier request when the latest request fails', async () => {
      let resolveFirst: (value: MetaResponse) => void = () => {};
      get
        .mockReturnValueOnce(
          new Promise((resolve) => {
            resolveFirst = resolve;
          }),
        )
        .mockRejectedValueOnce(new Error('offline'));
      const store = useMetaStore();
      const first = store.load();

      const reload = store.reload();
      await expect(reload).rejects.toThrow('offline');
      resolveFirst(meta);
      await first;

      expect(store.meta).toEqual(meta);
    });
  });
});
