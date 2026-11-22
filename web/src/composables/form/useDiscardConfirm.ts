import { computed, type ComputedRef, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { useReloadHold } from '@/composables/pwa/useReloadHold';
import { useConfirm } from '@/composables/ui/useConfirm';

/** Asks for confirmation before unsaved changes are dropped. */
export function useDiscardGuard(
  isDirty: Ref<boolean> | ComputedRef<boolean>,
  skipConfirm?: Ref<boolean> | ComputedRef<boolean>,
) {
  const { t } = useI18n();
  const { confirm } = useConfirm();

  /** Resolves to true once the changes may be dropped, after asking first while there are any. */
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
 * Wraps a modal's open state, so every way of closing it asks for confirmation
 * while the form has unsaved changes, and keeps the page from reloading on its own meanwhile.
 */
export function useDiscardConfirm(
  open: Ref<boolean>,
  isDirty: Ref<boolean> | ComputedRef<boolean>,
  skipConfirm?: Ref<boolean> | ComputedRef<boolean>,
) {
  const { confirmDiscard } = useDiscardGuard(isDirty, skipConfirm);
  useReloadHold(() => open.value && isDirty.value);

  /** Closes the modal, after asking first while the form has unsaved changes. */
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
