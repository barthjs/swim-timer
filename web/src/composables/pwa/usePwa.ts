import { shallowRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { tryOnMounted, tryOnScopeDispose, useDocumentVisibility } from '@vueuse/core';
import { useToast } from '@nuxt/ui/composables';

import { isReloadHeld } from '@/composables/pwa/useReloadHold';
import { outdatedCheck, reloadOnce } from '@/lib/pwa/outdated';
import { HELLO_MESSAGE, type Hello, type HelloMessage, useRealtimeStore } from '@/stores/realtime';

/**
 * Keeps the app at the version and the settings the server names on every connect,
 * reloading once a reload loses nothing and nobody reads the page.
 */
export function usePwa(version: string, isUnattended: () => boolean = () => false): void {
  const { t } = useI18n();
  const toast = useToast();
  const router = useRouter();
  const realtime = useRealtimeStore();
  const visibility = useDocumentVisibility();
  const isOutdated = outdatedCheck(version);
  // The greeting that showed the page behind the server, null while it is current
  const outdatedBy = shallowRef<Hello | null>(null);

  /** Reloads a page behind the server once it loses nothing and nobody reads it, and reports whether it does. */
  function reloadIfFree(): boolean {
    const hello = outdatedBy.value;
    if (!hello || isReloadHeld()) return false;
    if (visibility.value === 'visible' && !isUnattended()) return false;

    return reloadOnce(hello);
  }

  tryOnScopeDispose(
    realtime.onState<HelloMessage>(HELLO_MESSAGE, (message) => {
      if (outdatedBy.value || !isOutdated(message.data)) return;

      outdatedBy.value = message.data;
      if (reloadIfFree() || isUnattended()) return;

      toast.add({
        title: t('pwa.update.title'),
        duration: Number.POSITIVE_INFINITY,
        actions: [
          {
            label: t('pwa.reload'),
            onClick: () => window.location.reload(),
          },
        ],
      });
    }),
  );

  watch([visibility, isReloadHeld, isUnattended], reloadIfFree);

  // The guard is added on the mounted page, so a page rendered on the server adds none
  let removeGuard: (() => void) | undefined;
  tryOnScopeDispose(() => removeGuard?.());
  tryOnMounted(() => {
    removeGuard = router.beforeEach((to) => {
      if (!outdatedBy.value || isReloadHeld()) return true;

      window.location.assign(to.fullPath);
      return false;
    });
  });
}
