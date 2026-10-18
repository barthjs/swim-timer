import { useI18n } from 'vue-i18n';
import { useKbd } from '@nuxt/ui/composables';

// The keys a shortcut can be made of. Letters and digits are written as they are.
export type ShortcutKey =
  | 'meta'
  | 'shift'
  | 'alt'
  | 'escape'
  | 'tab'
  | 'delete'
  | 'backspace'
  | 'arrows'
  | 'space'
  | 'wheel'
  | (string & {});

/** macOS shows modifier and editing keys as symbols, the other platforms name them in the language of the UI. */
const MAC_SYMBOLS: Record<string, string> = {
  meta: '⌘',
  shift: '⇧',
  alt: '⌥',
  tab: '⇥',
  delete: '⌦',
  backspace: '⌫',
};

const LOCALIZED_KEYS = [
  'meta',
  'shift',
  'alt',
  'escape',
  'tab',
  'delete',
  'backspace',
  'arrows',
  'space',
  'wheel',
];

/**
 * useShortcutLabels writes the keys of a shortcut the way the platform and the language of the UI name them.
 * "meta" is the key Nuxt UI's defineShortcuts maps to Ctrl, or to Cmd on macOS.
 */
export function useShortcutLabels() {
  const { t } = useI18n();
  const { macOS } = useKbd();

  /** keyLabel returns the label of a key for the current platform and language. */
  function keyLabel(key: ShortcutKey): string {
    if (macOS.value && MAC_SYMBOLS[key]) {
      return MAC_SYMBOLS[key];
    }
    if (LOCALIZED_KEYS.includes(key)) {
      return t(`keys.${key}`);
    }
    return key.toUpperCase();
  }

  /** keyLabels returns one label per key, as the kbds of a UTooltip. */
  function keyLabels(keys: ShortcutKey[]): string[] {
    return keys.map(keyLabel);
  }

  return {
    keyLabel,
    keyLabels,
  };
}
