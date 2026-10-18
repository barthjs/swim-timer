import { computed, isRef, toValue, type MaybeRefOrGetter, type Ref } from 'vue';
import { defineShortcuts } from '@nuxt/ui/composables';
import type { Form, FormSchema } from '@nuxt/ui';

import { useShortcutLabels, type ShortcutKey } from '@/composables/useShortcutLabels';

/** SubmitTarget is the part of a Nuxt UI form instance the save shortcut submits. */
export type SubmitTarget = Pick<Form<FormSchema>, 'submit'>;

/** SaveTarget is a form to submit or a callback that saves. */
export type SaveTarget = Readonly<Ref<SubmitTarget | null | undefined>> | (() => unknown);

/** SAVE_SHORTCUT holds the keys of the save shortcut, Ctrl+S or Cmd+S on macOS. */
export const SAVE_SHORTCUT: ShortcutKey[] = ['meta', 's'];

/**
 * useSaveShortcut saves with Ctrl+S, or Cmd+S on macOS, also while a text field has the focus.
 * A form is submitted, so the shortcut validates and saves like its submit button.
 * The shortcut always replaces the save dialog of the browser, but only saves while it is enabled.
 */
export function useSaveShortcut(target: SaveTarget, enabled: MaybeRefOrGetter<boolean> = true) {
  const { keyLabels } = useShortcutLabels();

  /** save submits the form or runs the callback while the shortcut is enabled. */
  function save(): void {
    if (!toValue(enabled)) return;
    if (isRef(target)) {
      void target.value?.submit();
      return;
    }
    void target();
  }

  defineShortcuts({
    [SAVE_SHORTCUT.join('_')]: {
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
