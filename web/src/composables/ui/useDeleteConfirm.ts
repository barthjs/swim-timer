import { useI18n } from 'vue-i18n';

import { type ConfirmOptions, useConfirm } from '@/composables/ui/useConfirm';

/**
 * Asks for confirmation before something is deleted,
 * with the label, the color, and the icon every deletion shares.
 */
export function useDeleteConfirm() {
  const { t } = useI18n();
  const { confirm } = useConfirm();

  /** Opens the dialog of a deletion and resolves to true once it is confirmed. */
  async function confirmDelete(
    options: Pick<ConfirmOptions, 'title' | 'description'>,
  ): Promise<boolean> {
    return confirm({
      ...options,
      confirmLabel: t('common.delete'),
      color: 'error',
      icon: 'i-lucide-trash-2',
    });
  }

  return {
    confirmDelete,
  };
}
