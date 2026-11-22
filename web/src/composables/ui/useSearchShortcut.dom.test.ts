import { ref, toValue, type MaybeRefOrGetter } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ShortcutConfig, ShortcutsConfig } from '@nuxt/ui/composables';

import { useSearchShortcut } from '@/composables/ui/useSearchShortcut';
import { withSetup } from '@/test/withSetup';

const registered: MaybeRefOrGetter<ShortcutsConfig>[] = [];
const unmounts: (() => void)[] = [];

vi.mock('@nuxt/ui/composables', () => ({
  defineShortcuts: (config: MaybeRefOrGetter<ShortcutsConfig>) => {
    registered.push(config);
  },
}));

/** Mounts a search with the shortcut and returns its focus callback and the index of its registration. */
function mountSearch(enabled: MaybeRefOrGetter<boolean> = true) {
  const focus = vi.fn<() => void>();
  const index = registered.length;
  const { unmount } = withSetup(() => useSearchShortcut(focus, enabled));
  unmounts.push(unmount);
  return {
    focus,
    index,
    unmount,
  };
}

/** Returns the shortcut a search registered, or undefined while it leaves the shortcut to others. */
function registeredShortcut(index: number): ShortcutConfig | undefined {
  const shortcut = toValue(registered[index])?.meta_f;
  if (shortcut === undefined || shortcut === null) return undefined;
  if (typeof shortcut !== 'object') throw new Error('the search shortcut is no config');
  return shortcut;
}

describe('useSearchShortcut', () => {
  beforeEach(() => {
    for (const unmount of unmounts.splice(0)) unmount();
    registered.length = 0;
  });

  it('focuses the search, also from a text field', () => {
    const search = mountSearch();

    registeredShortcut(search.index)?.handler();

    expect(search.focus).toHaveBeenCalledOnce();
    expect(registeredShortcut(search.index)?.usingInput).toBe(true);
  });

  it('leaves the shortcut to the search mounted last', () => {
    const page = mountSearch();
    const dialog = mountSearch();

    registeredShortcut(dialog.index)?.handler();

    expect(dialog.focus).toHaveBeenCalledOnce();
    expect(registeredShortcut(page.index)).toBeUndefined();
  });

  it('takes the shortcut back once the search in front unmounts', () => {
    const page = mountSearch();
    const dialog = mountSearch();
    dialog.unmount();

    registeredShortcut(page.index)?.handler();

    expect(page.focus).toHaveBeenCalledOnce();
  });

  it('leaves the shortcut to the search below while it is disabled', () => {
    const page = mountSearch();
    const dialog = mountSearch(false);

    registeredShortcut(page.index)?.handler();

    expect(page.focus).toHaveBeenCalledOnce();
    expect(registeredShortcut(dialog.index)).toBeUndefined();
  });

  it('takes the shortcut once it is enabled', () => {
    const enabled = ref(false);
    const page = mountSearch();
    const dialog = mountSearch(enabled);
    enabled.value = true;

    registeredShortcut(dialog.index)?.handler();

    expect(dialog.focus).toHaveBeenCalledOnce();
    expect(registeredShortcut(page.index)).toBeUndefined();
  });

  it('leaves the shortcut to the browser without an enabled search', () => {
    const search = mountSearch(false);

    const shortcut = registeredShortcut(search.index);

    expect(shortcut).toBeUndefined();
  });
});
