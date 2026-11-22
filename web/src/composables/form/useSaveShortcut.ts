import { computed, isRef, toValue, type MaybeRefOrGetter, type Ref } from 'vue';
import { defineShortcuts } from '@nuxt/ui/composables';
import type { Form, FormSchema } from '@nuxt/ui';

import { useShortcutLabels, type ShortcutKey } from '@/composables/ui/useShortcutLabels';
import { shortcutDefinition } from '@/lib/ui/shortcuts';

/** The part of a Nuxt UI form instance the save shortcut submits. */
export type SubmitTarget = Pick<Form<FormSchema>, 'submit'>;

/** A form to submit or a callback that saves. */
export type SaveTarget = Readonly<Ref<SubmitTarget | null | undefined>> | (() => unknown);

/** Holds the keys of the save shortcut, Ctrl+S or Cmd+S on macOS. */
export const SAVE_SHORTCUT: ShortcutKey[] = ['meta', 's'];

/** Saves with Ctrl+S or Cmd+S like the submit button while it is enabled, instead of the browser. */
export function useSaveShortcut(target: SaveTarget, enabled: MaybeRefOrGetter<boolean> = true) {
  const { keyLabels } = useShortcutLabels();

  /** Submits the form or runs the callback while the shortcut is enabled. */
  function save(): void {
    if (!toValue(enabled)) return;
    if (isRef(target)) {
      void target.value?.submit();
      return;
    }
    void target();
  }

  defineShortcuts({
    [shortcutDefinition(SAVE_SHORTCUT)]: {
      usingInput: true,
      handler: save,
    },
  });

  // The labels of the keys for the tooltip of the save button
  const kbds = computed(() => keyLabels(SAVE_SHORTCUT));

  return {
    kbds,
    save,
  };
}
