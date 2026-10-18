import { ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ShortcutConfig, ShortcutsConfig } from '@nuxt/ui/composables';

import { useSaveShortcut, type SubmitTarget } from '@/composables/form/useSaveShortcut';

const registered: ShortcutsConfig[] = [];

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => `t:${key}`,
  }),
}));

vi.mock('@nuxt/ui/composables', () => ({
  useKbd: () => ({
    macOS: ref(false),
  }),
  defineShortcuts: (config: ShortcutsConfig) => {
    registered.push(config);
  },
}));

/** registeredShortcut returns the save shortcut the last call of the composable registered. */
function registeredShortcut(): ShortcutConfig {
  const shortcut = registered.at(-1)?.meta_s;
  if (!shortcut || typeof shortcut !== 'object') {
    throw new Error('the save shortcut is not registered');
  }
  return shortcut;
}

describe('useSaveShortcut', () => {
  beforeEach(() => {
    registered.length = 0;
  });

  it('registers the shortcut for text fields as well', () => {
    useSaveShortcut(() => undefined);

    expect(registeredShortcut().usingInput).toBe(true);
  });

  describe('save', () => {
    it('submits the form', () => {
      const submit = vi.fn<SubmitTarget['submit']>();
      useSaveShortcut(
        ref<SubmitTarget>({
          submit,
        }),
      );

      registeredShortcut().handler();

      expect(submit).toHaveBeenCalledOnce();
    });

    it('runs the callback', () => {
      const callback = vi.fn<() => void>();
      useSaveShortcut(callback);

      registeredShortcut().handler();

      expect(callback).toHaveBeenCalledOnce();
    });

    it('saves once it is enabled', () => {
      const callback = vi.fn<() => void>();
      const enabled = ref(false);
      useSaveShortcut(callback, enabled);
      enabled.value = true;

      registeredShortcut().handler();

      expect(callback).toHaveBeenCalledOnce();
    });

    it('ignores a missing form', () => {
      useSaveShortcut(ref(null));

      const save = () => registeredShortcut().handler();

      expect(save).not.toThrow();
    });

    it('rejects a save while it is disabled', () => {
      const callback = vi.fn<() => void>();
      const enabled = ref(false);
      useSaveShortcut(callback, enabled);

      registeredShortcut().handler();

      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('kbds', () => {
    it('returns the keys for the tooltip of the save button', () => {
      const { kbds } = useSaveShortcut(() => undefined);

      const keys = kbds.value;

      expect(keys).toEqual(['t:keys.meta', 'S']);
    });
  });
});
