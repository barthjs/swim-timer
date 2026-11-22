import { useEventListener } from '@vueuse/core';

/**
 * Asks the browser to warn before a reload or closing the tab while hasChanges is true.
 * The browser shows its own message, since only a genuine beforeunload listener can trigger the warning.
 */
export function useUnloadGuard(hasChanges: () => boolean): void {
  useEventListener('beforeunload', (event: BeforeUnloadEvent) => {
    if (!hasChanges()) return;
    event.preventDefault();
  });
}
