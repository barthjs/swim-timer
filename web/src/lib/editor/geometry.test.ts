import { describe, expect, it } from 'vitest';

import {
  aspectRatioDiffers,
  boundsIntersect,
  clamp,
  clampOffset,
  constrainToAxis,
  mapPointBetween,
  pointsBounds,
  sizeBounds,
  translateBounds,
  unionBounds,
} from '@/lib/editor/geometry';

const bounds = {
  minX: 10,
  maxX: 30,
  minY: 20,
  maxY: 40,
};

describe('pointsBounds', () => {
  it('spans every point', () => {
    const points = [
      {
        x: 30,
        y: 5,
      },
      {
        x: -10,
        y: 25,
      },
      {
        x: 0,
        y: 0,
      },
    ];

    const result = pointsBounds(points);

    expect(result).toEqual({
      minX: -10,
      maxX: 30,
      minY: 0,
      maxY: 25,
    });
  });
});

describe('unionBounds', () => {
  it('spans every bounds', () => {
    const other = {
      minX: 0,
      maxX: 20,
      minY: 30,
      maxY: 60,
    };

    const result = unionBounds([bounds, other]);

    expect(result).toEqual({
      minX: 0,
      maxX: 30,
      minY: 20,
      maxY: 60,
    });
  });

  it('returns null for no bounds', () => {
    const result = unionBounds([]);

    expect(result).toBeNull();
  });
});

describe('sizeBounds', () => {
  it('starts at the origin', () => {
    const result = sizeBounds({
      width: 200,
      height: 100,
    });

    expect(result).toEqual({
      minX: 0,
      maxX: 200,
      minY: 0,
      maxY: 100,
    });
  });
});

describe('boundsIntersect', () => {
  it('detects overlapping bounds', () => {
    const other = {
      minX: 25,
      maxX: 50,
      minY: 0,
      maxY: 25,
    };

    const intersects = boundsIntersect(bounds, other);

    expect(intersects).toBe(true);
  });

  it('keeps separate bounds apart', () => {
    const other = {
      minX: 40,
      maxX: 50,
      minY: 0,
      maxY: 100,
    };

    const intersects = boundsIntersect(bounds, other);

    expect(intersects).toBe(false);
  });
});

describe('aspectRatioDiffers', () => {
  it.each([
    {
      name: 'treats the same ratio at another size as equal',
      a: {
        width: 1920,
        height: 1080,
      },
      b: {
        width: 1280,
        height: 720,
      },
      expected: false,
    },
    {
      name: 'tolerates sizes rounded to whole pixels',
      a: {
        width: 1920,
        height: 1080,
      },
      b: {
        width: 1366,
        height: 768,
      },
      expected: false,
    },
    {
      name: 'detects another landscape ratio',
      a: {
        width: 1920,
        height: 1080,
      },
      b: {
        width: 1920,
        height: 1200,
      },
      expected: true,
    },
    {
      name: 'detects a portrait size',
      a: {
        width: 1920,
        height: 1080,
      },
      b: {
        width: 1080,
        height: 1920,
      },
      expected: true,
    },
    {
      name: 'never reports an empty size as different',
      a: {
        width: 1920,
        height: 1080,
      },
      b: {
        width: 0,
        height: 0,
      },
      expected: false,
    },
  ])('$name', ({ a, b, expected }) => {
    const differs = aspectRatioDiffers(a, b);

    expect(differs).toBe(expected);
  });
});

describe('clamp', () => {
  it.each([
    {
      name: 'keeps a value inside the range',
      value: 5,
      expected: 5,
    },
    {
      name: 'raises a value below the range to the minimum',
      value: -1,
      expected: 0,
    },
    {
      name: 'lowers a value above the range to the maximum',
      value: 11,
      expected: 10,
    },
  ])('$name', ({ value, expected }) => {
    const result = clamp(value, 0, 10);

    expect(result).toBe(expected);
  });

  it('collapses a range whose maximum is below its minimum to the minimum', () => {
    const result = clamp(5, 3, 1);

    expect(result).toBe(3);
  });
});

describe('clampOffset', () => {
  const container = sizeBounds({
    width: 100,
    height: 100,
  });

  it('keeps a shift that stays inside the container', () => {
    const offset = clampOffset(bounds, container, {
      x: 20,
      y: -10,
    });

    expect(offset).toEqual({
      x: 20,
      y: -10,
    });
  });

  it('stops the bounds at the border of the container', () => {
    const offset = clampOffset(bounds, container, {
      x: 500,
      y: -500,
    });

    expect(offset).toEqual({
      x: 70,
      y: -20,
    });
  });

  it('keeps the start of bounds larger than the container at its start', () => {
    const wide = {
      minX: 10,
      maxX: 210,
      minY: 0,
      maxY: 10,
    };

    const offset = clampOffset(wide, container, {
      x: 50,
      y: 0,
    });

    expect(offset).toEqual({
      x: -10,
      y: 0,
    });
  });
});

describe('translateBounds', () => {
  it('moves the bounds by the offset', () => {
    const moved = translateBounds(bounds, {
      x: 5,
      y: -5,
    });

    expect(moved).toEqual({
      minX: 15,
      maxX: 35,
      minY: 15,
      maxY: 35,
    });
  });
});

describe('constrainToAxis', () => {
  it('keeps the horizontal part of a mostly horizontal offset', () => {
    const offset = constrainToAxis({
      x: 10,
      y: -3,
    });

    expect(offset).toEqual({
      x: 10,
      y: 0,
    });
  });

  it('keeps the vertical part of a mostly vertical offset', () => {
    const offset = constrainToAxis({
      x: 2,
      y: -8,
    });

    expect(offset).toEqual({
      x: 0,
      y: -8,
    });
  });
});

describe('mapPointBetween', () => {
  it('keeps the relative position of the point', () => {
    const point = mapPointBetween(
      {
        x: 20,
        y: 30,
      },
      bounds,
      {
        minX: 0,
        maxX: 40,
        minY: 0,
        maxY: 40,
      },
    );

    expect(point).toEqual({
      x: 20,
      y: 20,
    });
  });

  it('keeps the start of bounds without an extent', () => {
    const point = mapPointBetween(
      {
        x: 10,
        y: 20,
      },
      {
        minX: 10,
        maxX: 10,
        minY: 20,
        maxY: 20,
      },
      bounds,
    );

    expect(point).toEqual({
      x: 10,
      y: 20,
    });
  });
});
