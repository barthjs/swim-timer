import { afterEach, describe, expect, it, vi } from 'vitest';

import { createClockSync } from '@/lib/clock';

describe('createClockSync', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  /** at sets the local clock to the given time. */
  function at(now: number): void {
    vi.spyOn(performance, 'now').mockReturnValue(now);
  }

  describe('toLocal', () => {
    it('maps server time onto the local clock', () => {
      const clock = createClockSync();
      at(100);
      clock.sample(5100);

      const local = clock.toLocal(5000);

      expect(local).toBe(0);
    });

    it('ignores the delay of late messages', () => {
      const clock = createClockSync();
      at(100);
      clock.sample(5100);
      at(400);
      clock.sample(5200);

      const local = clock.toLocal(5100);

      expect(local).toBe(100);
    });

    it('forgets samples beyond the window', () => {
      const clock = createClockSync(2);
      at(0);
      clock.sample(5000);
      at(100);
      clock.sample(4900);
      at(200);
      clock.sample(5000);

      const local = clock.toLocal(5000);

      expect(local).toBe(200);
    });

    it('returns null before the first sample', () => {
      const clock = createClockSync();

      const local = clock.toLocal(1000);

      expect(local).toBeNull();
    });
  });

  describe('reset', () => {
    it('removes the samples', () => {
      const clock = createClockSync();
      at(0);
      clock.sample(5000);

      clock.reset();

      expect(clock.toLocal(5000)).toBeNull();
    });
  });
});
