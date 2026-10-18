import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import type { ConfigTestResponse } from '@/api/types';

/**
 * useConnectionTestReport reports the outcome of a connection test as a toast.
 * Unlike save, a failed check is not an ApiError, since the request itself succeeds
 * and the result carries ok and message.
 */
export function useConnectionTestReport() {
  const { t } = useI18n();
  const toast = useToast();

  /** reportTestConnection shows whether the tested connection works. */
  function reportTestConnection(result: ConfigTestResponse): void {
    if (result.ok) {
      toast.add({
        title: t('pages.config.testConnectionSuccess'),
        icon: 'i-lucide-check-circle',
        color: 'success',
      });
      return;
    }
    toast.add({
      title: t('pages.config.testConnectionFailure'),
      description: result.message,
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  }

  return {
    reportTestConnection,
  };
}
