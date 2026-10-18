import { computed, type ComputedRef, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { useConfirm } from '@/composables/ui/useConfirm';

/**
 * useDiscardGuard asks for confirmation before unsaved changes are dropped, such as when a page is left
 * or another item is opened.
 */
export function useDiscardGuard(
  isDirty: Ref<boolean> | ComputedRef<boolean>,
  skipConfirm?: Ref<boolean> | ComputedRef<boolean>,
) {
  const { t } = useI18n();
  const { confirm } = useConfirm();

  /** confirmDiscard resolves to true once the changes may be dropped, after asking first while there are any. */
  async function confirmDiscard(): Promise<boolean> {
    if (!isDirty.value || (skipConfirm?.value ?? false)) return true;
    return confirm({
      title: t('common.discardTitle'),
      description: t('common.discardDescription'),
      confirmLabel: t('common.discardConfirm'),
      color: 'error',
      icon: 'i-lucide-triangle-alert',
    });
  }

  return {
    confirmDiscard,
  };
}

/**
 * useDiscardConfirm wraps a modal's open state, so every way of closing it asks for confirmation
 * while the form has unsaved changes.
 */
export function useDiscardConfirm(
  open: Ref<boolean>,
  isDirty: Ref<boolean> | ComputedRef<boolean>,
  skipConfirm?: Ref<boolean> | ComputedRef<boolean>,
) {
  const { confirmDiscard } = useDiscardGuard(isDirty, skipConfirm);

  /** attemptClose closes the modal, after asking first while the form has unsaved changes. */
  async function attemptClose(): Promise<void> {
    if (!(await confirmDiscard())) return;
    open.value = false;
  }

  const openProxy = computed<boolean>({
    get: () => open.value,
    set: (value) => {
      if (value) {
        open.value = true;
        return;
      }
      void attemptClose();
    },
  });

  return {
    openProxy,
    attemptClose,
  };
}
