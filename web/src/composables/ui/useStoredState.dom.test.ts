import { nextTick } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useStoredState } from '@/composables/ui/useStoredState';
import { withSetup } from '@/test/withSetup';

/** Reads a stored number and rejects everything else. */
function parseCount(raw: unknown): number | undefined {
  return typeof raw === 'number' ? raw : undefined;
}

describe('useStoredState', () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('restores a stored value once mounted', () => {
    const key = faker.lorem.slug();
    const stored = faker.number.int();
    localStorage.setItem(key, JSON.stringify(stored));

    const { result: count } = withSetup(() => useStoredState(key, () => 0, parseCount));

    expect(count.value).toBe(stored);
  });

  it('stores every change, also deep ones', async () => {
    const key = faker.lorem.slug();
    const item = faker.lorem.word();
    const { result: state } = withSetup(() =>
      useStoredState(
        key,
        () => ({
          items: [] as string[],
        }),
        () => undefined,
      ),
    );

    state.value.items.push(item);
    await nextTick();

    expect(JSON.parse(localStorage.getItem(key) ?? 'null')).toEqual({
      items: [item],
    });
  });

  it('lives without storage when there is no key', async () => {
    const value = faker.number.int();
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const { result: count } = withSetup(() => useStoredState(undefined, () => 0, parseCount));

    count.value = value;
    await nextTick();

    expect(count.value).toBe(value);
    expect(setItem).not.toHaveBeenCalled();
  });

  it.each([
    {
      name: 'a value of another type',
      stored: '"many"',
    },
    {
      name: 'a value that is no JSON',
      stored: '{',
    },
  ])('keeps the fallback for $name', ({ stored }) => {
    const key = faker.lorem.slug();
    const fallback = faker.number.int();
    localStorage.setItem(key, stored);

    const { result: count } = withSetup(() => useStoredState(key, () => fallback, parseCount));

    expect(count.value).toBe(fallback);
  });

  it('keeps working when the browser denies the storage', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error(faker.lorem.sentence());
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error(faker.lorem.sentence());
    });
    const value = faker.number.int();
    const { result: count } = withSetup(() =>
      useStoredState(faker.lorem.slug(), () => 0, parseCount),
    );

    count.value = value;
    await nextTick();

    expect(count.value).toBe(value);
  });
});
