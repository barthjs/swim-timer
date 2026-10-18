import { describe, expect, it } from 'vitest';

import { useRatioLock } from '@/composables/useRatioLock';

describe('useRatioLock', () => {
  it('starts unlocked', () => {
    const lock = useRatioLock();

    const state = [lock.locked.value, lock.ratio.value];

    expect(state).toEqual([false, 1]);
  });

  describe('toggle', () => {
    it('captures the ratio when it is switched on', () => {
      const lock = useRatioLock();

      lock.toggle(1.5);

      expect(lock.locked.value).toBe(true);
      expect(lock.ratio.value).toBe(1.5);
    });

    it('keeps the ratio when it is switched off', () => {
      const lock = useRatioLock();
      lock.toggle(1.5);

      lock.toggle(3);

      expect(lock.locked.value).toBe(false);
      expect(lock.ratio.value).toBe(1.5);
    });

    it('captures the new ratio when it is switched on again', () => {
      const lock = useRatioLock();
      lock.toggle(1.5);
      lock.toggle(3);

      lock.toggle(0.75);

      expect(lock.ratio.value).toBe(0.75);
    });
  });

  describe('lock', () => {
    it('locks at the given ratio', () => {
      const lock = useRatioLock();

      lock.lock(16 / 9);

      expect(lock.locked.value).toBe(true);
      expect(lock.ratio.value).toBe(16 / 9);
    });

    it('replaces the ratio of a locked lock', () => {
      const lock = useRatioLock();
      lock.lock(16 / 9);

      lock.lock(4 / 3);

      expect(lock.locked.value).toBe(true);
      expect(lock.ratio.value).toBe(4 / 3);
    });
  });

  describe('unlock', () => {
    it('unlocks a locked lock', () => {
      const lock = useRatioLock();
      lock.lock(16 / 9);

      lock.unlock();

      expect(lock.locked.value).toBe(false);
    });
  });

  describe('heightFor', () => {
    it('derives the height from the locked ratio', () => {
      const lock = useRatioLock();
      lock.lock(16 / 9);

      const height = lock.heightFor(1280);

      expect(height).toBe(720);
    });
  });

  describe('widthFor', () => {
    it('derives the width from the locked ratio', () => {
      const lock = useRatioLock();
      lock.lock(16 / 9);

      const width = lock.widthFor(1080);

      expect(width).toBe(1920);
    });
  });
});
