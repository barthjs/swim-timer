import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue';
import { breakpointsTailwind } from '@vueuse/core';

export type Breakpoint = keyof typeof breakpointsTailwind;

/**
 * Tracks whether the viewport reaches a Tailwind breakpoint.
 * It stays false until the component mounts, so a prerendered page hydrates without a mismatch.
 */
export function useBreakpoint(breakpoint: Breakpoint): Ref<boolean> {
  const reached = ref(false);

  let media: MediaQueryList | null = null;

  /** Reads whether the media query matches. */
  function update(): void {
    if (media) reached.value = media.matches;
  }

  onMounted(() => {
    media = window.matchMedia(`(min-width: ${breakpointsTailwind[breakpoint]}px)`);
    update();
    media.addEventListener('change', update);
  });

  onBeforeUnmount(() => {
    media?.removeEventListener('change', update);
    media = null;
  });

  return reached;
}
