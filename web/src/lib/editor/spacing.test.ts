import { describe, expect, it } from 'vitest';

import { measureGaps, spacingTargets } from '@/lib/editor/spacing';

const content = {
  width: 100,
  height: 100,
};

// An item from 40 to 60 on both axes, with a neighbor on the left and one on the right.
const moved = {
  minX: 40,
  maxX: 60,
  minY: 40,
  maxY: 60,
};
const left = {
  minX: 0,
  maxX: 20,
  minY: 40,
  maxY: 60,
};
const right = {
  minX: 90,
  maxX: 100,
  minY: 45,
  maxY: 55,
};

describe('measureGaps', () => {
  it('measures the distance to the nearest neighbor on each side', () => {
    const gaps = measureGaps(moved, [left, right], content);

    expect(gaps).toEqual([
      {
        axis: 'x',
        from: 20,
        to: 40,
        at: 50,
      },
      {
        axis: 'x',
        from: 60,
        to: 90,
        at: 50,
      },
      {
        axis: 'y',
        from: 0,
        to: 40,
        at: 50,
      },
      {
        axis: 'y',
        from: 60,
        to: 100,
        at: 50,
      },
    ]);
  });

  it('leaves out a side that touches its neighbor', () => {
    const touching = {
      minX: 20,
      maxX: 40,
      minY: 40,
      maxY: 60,
    };

    const gaps = measureGaps(touching, [left], content);

    expect(gaps.filter((gap) => gap.axis === 'x' && gap.to === 20)).toEqual([]);
  });
});

describe('spacingTargets', () => {
  it('centers the bounds between the neighbors on both sides', () => {
    const targets = spacingTargets(moved, [left, right]);

    expect(targets.x).toContain(45);
  });

  it('offers the gap two other items already have', () => {
    const first = {
      minX: 0,
      maxX: 10,
      minY: 40,
      maxY: 60,
    };
    const second = {
      minX: 15,
      maxX: 25,
      minY: 40,
      maxY: 60,
    };

    const targets = spacingTargets(moved, [first, second]);

    expect(targets.x).toContain(30);
  });

  it('offers nothing without neighbors', () => {
    const targets = spacingTargets(moved, []);

    expect(targets).toEqual({
      x: [],
      y: [],
    });
  });
});
