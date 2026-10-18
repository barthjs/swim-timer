import { describe, expect, it, vi } from 'vitest';

import { createKeyValueStore } from '@/lib/keyValueStore';

describe('createKeyValueStore', () => {
  describe('get', () => {
    it('returns null when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>('test', 'values', () => undefined);
      await store.set('key', 'value');

      const value = await store.get('key');

      expect(value).toBeNull();
    });

    it('returns null when the browser refuses IndexedDB', async () => {
      const factory = vi.fn<() => IDBFactory | undefined>(() => {
        throw new Error('denied');
      });
      const store = createKeyValueStore<string>('test', 'values', factory);

      const value = await store.get('key');

      expect(value).toBeNull();
    });

    it('tries a refused open again on the next operation', async () => {
      const factory = vi.fn<() => IDBFactory | undefined>(() => {
        throw new Error('denied');
      });
      const store = createKeyValueStore<string>('test', 'values', factory);
      await store.get('key');

      const value = await store.get('key');

      expect(value).toBeNull();
      expect(factory).toHaveBeenCalledTimes(2);
    });
  });

  describe('set', () => {
    it('keeps working without storing anything when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>('test', 'values', () => undefined);

      const result = await store.set('key', 'value');

      expect(result).toBeUndefined();
    });
  });

  describe('delete', () => {
    it('keeps working without storing anything when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>('test', 'values', () => undefined);

      const result = await store.delete('key');

      expect(result).toBeUndefined();
    });
  });
});
