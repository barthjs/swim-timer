import { nextTick } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useStoredState } from '@/composables/useStoredState';
import { withSetup } from '@/test/withSetup';

/** parseCount reads a stored number and rejects everything else. */
function parseCount(raw: unknown): number | undefined {
  return typeof raw === 'number' ? raw : undefined;
}

describe('useStoredState', () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('restores a stored value once mounted', () => {
    localStorage.setItem('count', '5');

    const { result: count } = withSetup(() => useStoredState('count', () => 0, parseCount));

    expect(count.value).toBe(5);
  });

  it('stores every change, also deep ones', async () => {
    const { result: state } = withSetup(() =>
      useStoredState(
        'state',
        () => ({
          items: [] as string[],
        }),
        () => undefined,
      ),
    );

    state.value.items.push('first');
    await nextTick();

    expect(JSON.parse(localStorage.getItem('state') ?? 'null')).toEqual({
      items: ['first'],
    });
  });

  it('lives without storage when there is no key', async () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const { result: count } = withSetup(() => useStoredState(undefined, () => 0, parseCount));

    count.value = 3;
    await nextTick();

    expect(count.value).toBe(3);
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
    localStorage.setItem('count', stored);

    const { result: count } = withSetup(() => useStoredState('count', () => 1, parseCount));

    expect(count.value).toBe(1);
  });

  it('keeps working when the browser denies the storage', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied');
    });
    const { result: count } = withSetup(() => useStoredState('count', () => 0, parseCount));

    count.value = 2;
    await nextTick();

    expect(count.value).toBe(2);
  });
});
