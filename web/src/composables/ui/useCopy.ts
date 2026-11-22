import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { copyText } from '@/lib/browser/clipboard';

/** Copies text to the clipboard and confirms it with a toast, or tells that it failed. */
export function useCopy() {
  const { t } = useI18n();
  const toast = useToast();

  /** Copies the text and shows the title once it is on the clipboard. */
  async function copy(text: string, title: string): Promise<void> {
    if (await copyText(text)) {
      toast.add({
        title,
        icon: 'i-lucide-clipboard-check',
        color: 'success',
      });
      return;
    }
    toast.add({
      title: t('common.copyFailed'),
      icon: 'i-lucide-triangle-alert',
      color: 'error',
    });
  }

  return {
    copy,
  };
}
