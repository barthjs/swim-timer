import { ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useShortcutLabels } from '@/composables/useShortcutLabels';

const macOS = ref(false);

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => `t:${key}`,
  }),
}));

vi.mock('@nuxt/ui/composables', () => ({
  useKbd: () => ({
    macOS,
  }),
}));

describe('useShortcutLabels', () => {
  beforeEach(() => {
    macOS.value = false;
  });

  describe('keyLabels', () => {
    it('names modifier keys in the language of the UI', () => {
      const { keyLabels } = useShortcutLabels();

      const labels = keyLabels(['meta', 'shift', 'z']);

      expect(labels).toEqual(['t:keys.meta', 't:keys.shift', 'Z']);
    });

    it('names pointer keys in the language of the UI', () => {
      const { keyLabels } = useShortcutLabels();

      const labels = keyLabels(['space', 'wheel']);

      expect(labels).toEqual(['t:keys.space', 't:keys.wheel']);
    });

    it('names editing keys in the language of the UI', () => {
      const { keyLabels } = useShortcutLabels();

      const labels = keyLabels(['tab', 'delete', 'backspace']);

      expect(labels).toEqual(['t:keys.tab', 't:keys.delete', 't:keys.backspace']);
    });

    it('shows modifier symbols on macOS', () => {
      macOS.value = true;
      const { keyLabels } = useShortcutLabels();

      const labels = keyLabels(['meta', 'shift', 'alt', 'escape']);

      expect(labels).toEqual(['⌘', '⇧', '⌥', 't:keys.escape']);
    });

    it('shows editing key symbols on macOS', () => {
      macOS.value = true;
      const { keyLabels } = useShortcutLabels();

      const labels = keyLabels(['tab', 'delete', 'backspace']);

      expect(labels).toEqual(['⇥', '⌦', '⌫']);
    });
  });
});
