import { describe, expect, it } from 'vitest';

import { Phase } from '@/api/types';
import {
  compareResults,
  finalPhaseName,
  hasRank,
  isResultRanked,
  showsHeatCount,
} from '@/lib/heat';

/** result builds the fields of a lane the result order reads. */
function result(number: number, rank: number, finished: boolean) {
  return {
    number,
    rank,
    finished,
  };
}

describe('hasRank', () => {
  it('returns true for a rank', () => {
    const ranked = hasRank({
      rank: 1,
    });

    expect(ranked).toBe(true);
  });

  it('treats rank 0 as no rank', () => {
    const ranked = hasRank({
      rank: 0,
    });

    expect(ranked).toBe(false);
  });
});

describe('isResultRanked', () => {
  it('returns true for a finished lane with a rank', () => {
    const ranked = isResultRanked(result(1, 1, true));

    expect(ranked).toBe(true);
  });

  it.each([
    {
      name: 'rejects a finished lane without a rank',
      lane: result(1, 0, true),
    },
    {
      name: 'rejects a ranked lane without a finish',
      lane: result(1, 2, false),
    },
  ])('$name', ({ lane }) => {
    const ranked = isResultRanked(lane);

    expect(ranked).toBe(false);
  });
});

describe('compareResults', () => {
  it('lists ranked lanes by rank and every other lane after them by lane number', () => {
    const lanes = [
      result(1, 0, false),
      result(2, 3, true),
      result(3, 0, true),
      result(4, 1, true),
      result(5, 1, true),
      result(6, 2, false),
    ];

    const order = [...lanes].sort(compareResults).map((lane) => lane.number);

    expect(order).toEqual([4, 5, 2, 1, 3, 6]);
  });
});

describe('finalPhaseName', () => {
  it('returns the trimmed name of a final', () => {
    const name = finalPhaseName({
      phase: Phase.Final,
      phaseName: ' Final ',
    });

    expect(name).toBe('Final');
  });

  it.each([
    {
      name: 'returns an empty string for another phase',
      phase: Phase.Heats,
      phaseName: 'Heats',
    },
    {
      name: 'returns an empty string for a final without a name',
      phase: Phase.Final,
      phaseName: '',
    },
  ])('$name', ({ phase, phaseName }) => {
    const name = finalPhaseName({
      phase,
      phaseName,
    });

    expect(name).toBe('');
  });
});

describe('showsHeatCount', () => {
  it.each([
    {
      heat: 1,
      heatCount: 5,
      expected: true,
    },
    {
      heat: 5,
      heatCount: 5,
      expected: true,
    },
    {
      heat: 6,
      heatCount: 5,
      expected: false,
    },
    {
      heat: 0,
      heatCount: 5,
      expected: false,
    },
    {
      heat: 1,
      heatCount: 0,
      expected: false,
    },
  ])('returns $expected for heat $heat of $heatCount', ({ heat, heatCount, expected }) => {
    const shown = showsHeatCount(heat, heatCount);

    expect(shown).toBe(expected);
  });
});
