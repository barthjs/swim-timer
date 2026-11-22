import { useEventListener } from '@vueuse/core';

import { useReloadHold } from '@/composables/pwa/useReloadHold';

/**
 * Asks the browser to warn before a reload or closing the tab while hasChanges is true,
 * and keeps the page from reloading on its own meanwhile. The browser shows its own message,
 * since only a genuine beforeunload listener can trigger the warning.
 */
export function useUnloadGuard(hasChanges: () => boolean): void {
  useReloadHold(hasChanges);
  useEventListener('beforeunload', (event: BeforeUnloadEvent) => {
    if (!hasChanges()) return;
    event.preventDefault();
  });
}
