import { shallowReactive } from 'vue';
import { tryOnScopeDispose } from '@vueuse/core';

// The holds of every component, reactive so a watcher of isReloadHeld follows them
const holds = shallowReactive(new Set<() => boolean>());

/** Keeps the page from reloading on its own while isHolding is true and the calling scope lives. */
export function useReloadHold(isHolding: () => boolean): void {
  holds.add(isHolding);
  tryOnScopeDispose(() => holds.delete(isHolding));
}

/** Reports whether a reload would lose something on the page. */
export function isReloadHeld(): boolean {
  for (const isHolding of holds) {
    if (isHolding()) return true;
  }
  return false;
}
