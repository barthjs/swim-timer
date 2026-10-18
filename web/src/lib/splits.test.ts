import { describe, expect, it } from 'vitest';

import type { Lane } from '@/api/types';
import { createSplitTracker, heatRun } from '@/lib/splits';

/** lane builds a lane with a time. */
function lane(number: number, time: number): Lane {
  return {
    number,
    firstName: '',
    lastName: '',
    birthYear: '',
    club: '',
    lap: 1,
    meter: 25,
    time,
    rank: 0,
    finished: false,
  };
}

const firstStart = '2026-10-01T10:00:00Z';
const secondStart = '2026-10-01T10:05:00Z';

describe('heatRun', () => {
  it('returns the heat without a start', () => {
    const run = heatRun(1, 2, undefined);

    expect(run).toEqual({
      heat: '1-2',
      startTime: '',
    });
  });

  it('keeps the heat of a started heat', () => {
    const run = heatRun(1, 2, firstStart);

    expect(run.heat).toBe(heatRun(1, 2, undefined).heat);
  });

  it('distinguishes the starts of a heat', () => {
    const run = heatRun(1, 2, firstStart);

    expect(run.startTime).not.toBe(heatRun(1, 2, undefined).startTime);
  });

  it('distinguishes heats', () => {
    const run = heatRun(1, 3, firstStart);

    expect(run.heat).not.toBe(heatRun(1, 2, firstStart).heat);
  });
});

describe('createSplitTracker', () => {
  describe('update', () => {
    it('returns the lanes whose time changed', () => {
      const tracker = createSplitTracker();
      const run = heatRun(1, 1, firstStart);
      tracker.update(run, [lane(1, 15000), lane(2, 16000)]);

      const changed = tracker.update(run, [lane(1, 31000), lane(2, 16000), lane(3, 17000)]);

      expect(changed.map((item) => item.number)).toEqual([1, 3]);
    });

    it('only records the times of the first update', () => {
      const tracker = createSplitTracker();

      const changed = tracker.update(heatRun(1, 1, firstStart), [lane(1, 15000), lane(2, 16000)]);

      expect(changed).toEqual([]);
    });

    it('only records the times a heat already has when the heat changes', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 1, firstStart), [lane(1, 15000)]);
      tracker.update(heatRun(1, 1, firstStart), [lane(1, 31000)]);

      const changed = tracker.update(heatRun(1, 2, secondStart), [lane(1, 16000), lane(2, 17000)]);

      expect(changed).toEqual([]);
    });

    it('returns the changed times of a heat after the heat changed', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 1, firstStart), [lane(1, 15000)]);
      tracker.update(heatRun(1, 1, firstStart), [lane(1, 31000)]);
      tracker.update(heatRun(1, 2, secondStart), [lane(1, 16000), lane(2, 17000)]);

      const changed = tracker.update(heatRun(1, 2, secondStart), [lane(1, 33000), lane(2, 17000)]);

      expect(changed.map((item) => item.number)).toEqual([1]);
    });

    it('forgets the times when the heat changes', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 1, firstStart), [lane(1, 15000)]);
      tracker.update(heatRun(1, 2, secondStart), []);

      const changed = tracker.update(heatRun(1, 2, secondStart), [lane(1, 15000)]);

      expect(changed.map((item) => item.number)).toEqual([1]);
    });

    it('returns the first time that arrives together with the start of the heat', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 2, undefined), []);

      const changed = tracker.update(heatRun(1, 2, firstStart), [lane(4, 15000)]);

      expect(changed.map((item) => item.number)).toEqual([4]);
    });

    it('returns no times for a new start of the same heat', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 2, firstStart), []);
      tracker.update(heatRun(1, 2, firstStart), [lane(4, 15000)]);

      const changed = tracker.update(heatRun(1, 2, secondStart), []);

      expect(changed).toEqual([]);
    });

    it('returns the times of a new start of the same heat, also a time equal to one before', () => {
      const tracker = createSplitTracker();
      tracker.update(heatRun(1, 2, firstStart), []);
      tracker.update(heatRun(1, 2, firstStart), [lane(4, 15000)]);
      tracker.update(heatRun(1, 2, secondStart), []);

      const changed = tracker.update(heatRun(1, 2, secondStart), [lane(4, 15000)]);

      expect(changed.map((item) => item.number)).toEqual([4]);
    });
  });
});
