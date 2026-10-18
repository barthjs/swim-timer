import { describe, expect, it } from 'vitest';

import { alignOffsets, alignReference, distributeOffsets } from '@/lib/editor/align';

const content = {
  width: 200,
  height: 100,
};

const left = {
  minX: 10,
  maxX: 30,
  minY: 10,
  maxY: 20,
};

const right = {
  minX: 50,
  maxX: 90,
  minY: 40,
  maxY: 80,
};

describe('alignReference', () => {
  it('uses the bounds of several items together', () => {
    const reference = alignReference([left, right], content);

    expect(reference).toEqual({
      minX: 10,
      maxX: 90,
      minY: 10,
      maxY: 80,
    });
  });

  it('uses the content for a single item', () => {
    const reference = alignReference([left], content);

    expect(reference).toEqual({
      minX: 0,
      maxX: 200,
      minY: 0,
      maxY: 100,
    });
  });
});

describe('alignOffsets', () => {
  const reference = alignReference([left, right], content);

  it.each([
    {
      edge: 'left' as const,
      expected: [
        {
          x: 0,
          y: 0,
        },
        {
          x: -40,
          y: 0,
        },
      ],
    },
    {
      edge: 'centerX' as const,
      expected: [
        {
          x: 30,
          y: 0,
        },
        {
          x: -20,
          y: 0,
        },
      ],
    },
    {
      edge: 'right' as const,
      expected: [
        {
          x: 60,
          y: 0,
        },
        {
          x: 0,
          y: 0,
        },
      ],
    },
    {
      edge: 'top' as const,
      expected: [
        {
          x: 0,
          y: 0,
        },
        {
          x: 0,
          y: -30,
        },
      ],
    },
    {
      edge: 'centerY' as const,
      expected: [
        {
          x: 0,
          y: 30,
        },
        {
          x: 0,
          y: -15,
        },
      ],
    },
    {
      edge: 'bottom' as const,
      expected: [
        {
          x: 0,
          y: 60,
        },
        {
          x: 0,
          y: 0,
        },
      ],
    },
  ])('lines the items up at $edge', ({ edge, expected }) => {
    const offsets = alignOffsets([left, right], edge, reference);

    expect(offsets).toEqual(expected);
  });
});

describe('distributeOffsets', () => {
  it('evens out the gaps between the first and the last item', () => {
    const middle = {
      minX: 31,
      maxX: 41,
      minY: 0,
      maxY: 10,
    };

    const offsets = distributeOffsets([right, middle, left], 'x');

    expect(offsets).toEqual([
      {
        x: 0,
        y: 0,
      },
      {
        x: 4,
        y: 0,
      },
      {
        x: 0,
        y: 0,
      },
    ]);
  });

  it('moves the inner items in the order of their start', () => {
    const middle = {
      minX: 0,
      maxX: 10,
      minY: 22,
      maxY: 32,
    };

    const offsets = distributeOffsets([left, right, middle], 'y');

    expect(offsets).toEqual([
      {
        x: 0,
        y: 0,
      },
      {
        x: 0,
        y: 0,
      },
      {
        x: 0,
        y: 3,
      },
    ]);
  });

  it('keeps too few items in place', () => {
    const offsets = distributeOffsets([left, right], 'x');

    expect(offsets).toEqual([
      {
        x: 0,
        y: 0,
      },
      {
        x: 0,
        y: 0,
      },
    ]);
  });
});
