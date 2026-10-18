import { ref } from 'vue';

/**
 * useRatioLock holds the lock of a width to height ratio, for a size whose other dimension follows
 * while it is locked. The ratio is captured when the lock is switched on, so it does not drift
 * through rounded values while the size changes. The lock is an editing aid and is not saved.
 */
export function useRatioLock() {
  const locked = ref(false);
  const ratio = ref(1);

  /** toggle locks the ratio at the given width to height ratio, or unlocks it. */
  function toggle(currentRatio: number): void {
    locked.value = !locked.value;
    if (locked.value) ratio.value = currentRatio;
  }

  /** lock locks the ratio at the given width to height ratio. */
  function lock(nextRatio: number): void {
    ratio.value = nextRatio;
    locked.value = true;
  }

  /** unlock lets width and height change independently. */
  function unlock(): void {
    locked.value = false;
  }

  /** heightFor returns the height that has the locked ratio with the given width. */
  function heightFor(width: number): number {
    return width / ratio.value;
  }

  /** widthFor returns the width that has the locked ratio with the given height. */
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
