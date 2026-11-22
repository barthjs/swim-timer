import { ref } from 'vue';

/**
 * Holds the lock of a width to height ratio, captured when it is switched on,
 * so it does not drift through rounded values.
 */
export function useRatioLock() {
  const locked = ref(false);
  const ratio = ref(1);

  /** Locks the ratio at the given width to height ratio, or unlocks it. */
  function toggle(currentRatio: number): void {
    locked.value = !locked.value;
    if (locked.value) ratio.value = currentRatio;
  }

  /** Locks the ratio at the given width to height ratio. */
  function lock(nextRatio: number): void {
    ratio.value = nextRatio;
    locked.value = true;
  }

  /** Lets width and height change independently. */
  function unlock(): void {
    locked.value = false;
  }

  /** Returns the height that has the locked ratio with the given width. */
  function heightFor(width: number): number {
    return width / ratio.value;
  }

  /** Returns the width that has the locked ratio with the given height. */
  function widthFor(height: number): number {
    return height * ratio.value;
  }

  return {
    locked,
    ratio,
    toggle,
    lock,
    unlock,
    heightFor,
    widthFor,
  };
}
