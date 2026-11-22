import {
  computed,
  onMounted,
  onUnmounted,
  shallowReactive,
  toValue,
  type MaybeRefOrGetter,
} from 'vue';
import { defineShortcuts } from '@nuxt/ui/composables';

import type { ShortcutKey } from '@/composables/ui/useShortcutLabels';
import { shortcutDefinition } from '@/lib/ui/shortcuts';

/** Holds the keys that focus a search, Ctrl+F or Cmd+F on macOS. */
export const SEARCH_SHORTCUT: ShortcutKey[] = ['meta', 'f'];

/** A mounted search that may take the shortcut. */
type SearchShortcutOwner = {
  enabled: () => boolean;
};

// The mounted searches in the order they mounted, so the last one lies in front, such as the search of a dialog
const owners = shallowReactive<SearchShortcutOwner[]>([]);

/** Focuses the search mounted last with Ctrl+F or Cmd+F while it is enabled. */
export function useSearchShortcut(focus: () => void, enabled: MaybeRefOrGetter<boolean> = true) {
  const owner: SearchShortcutOwner = {
    enabled: () => toValue(enabled),
  };

  onMounted(() => {
    owners.push(owner);
  });

  onUnmounted(() => {
    const index = owners.indexOf(owner);
    if (index !== -1) owners.splice(index, 1);
  });

  const inFront = computed(
    // oxlint-disable-next-line unicorn/prefer-array-find -- findLast is newer than the ES version of the build
    () => owners.filter((candidate) => candidate.enabled()).at(-1) === owner,
  );

  defineShortcuts(
    computed(() => {
      if (!inFront.value) return {};
      return {
        [shortcutDefinition(SEARCH_SHORTCUT)]: {
          usingInput: true,
          handler: focus,
        },
      };
    }),
  );
}
