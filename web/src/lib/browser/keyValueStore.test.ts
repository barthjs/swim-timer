import { describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { createKeyValueStore } from '@/lib/browser/keyValueStore';

describe('createKeyValueStore', () => {
  describe('get', () => {
    it('returns null when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>(
        faker.lorem.slug(),
        faker.lorem.slug(),
        () => undefined,
      );
      await store.set(faker.lorem.slug(), faker.lorem.word());

      const value = await store.get(faker.lorem.slug());

      expect(value).toBeNull();
    });

    it('returns null when the browser refuses IndexedDB', async () => {
      const factory = vi.fn<() => IDBFactory | undefined>(() => {
        throw new Error(faker.lorem.sentence());
      });
      const store = createKeyValueStore<string>(faker.lorem.slug(), faker.lorem.slug(), factory);

      const value = await store.get(faker.lorem.slug());

      expect(value).toBeNull();
    });

    it('tries a refused open again on the next operation', async () => {
      const factory = vi.fn<() => IDBFactory | undefined>(() => {
        throw new Error(faker.lorem.sentence());
      });
      const store = createKeyValueStore<string>(faker.lorem.slug(), faker.lorem.slug(), factory);
      await store.get(faker.lorem.slug());

      const value = await store.get(faker.lorem.slug());

      expect(value).toBeNull();
      expect(factory).toHaveBeenCalledTimes(2);
    });
  });

  describe('set', () => {
    it('keeps working without storing anything when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>(
        faker.lorem.slug(),
        faker.lorem.slug(),
        () => undefined,
      );

      await expect(store.set(faker.lorem.slug(), faker.lorem.word())).resolves.toBeUndefined();
    });
  });

  describe('delete', () => {
    it('keeps working without storing anything when IndexedDB is missing', async () => {
      const store = createKeyValueStore<string>(
        faker.lorem.slug(),
        faker.lorem.slug(),
        () => undefined,
      );

      await expect(store.delete(faker.lorem.slug())).resolves.toBeUndefined();
    });
  });
});
