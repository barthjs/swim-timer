import { describe, expect, it } from 'vitest';

import { isConvexClockwise, mapPoint, squareToQuad, toMatrix3d } from '@/lib/homography';

const rectangle = {
  topLeft: {
    x: 100,
    y: 50,
  },
  topRight: {
    x: 500,
    y: 50,
  },
  bottomRight: {
    x: 500,
    y: 250,
  },
  bottomLeft: {
    x: 100,
    y: 250,
  },
};

const trapezoid = {
  topLeft: {
    x: 200,
    y: 100,
  },
  topRight: {
    x: 600,
    y: 100,
  },
  bottomRight: {
    x: 800,
    y: 400,
  },
  bottomLeft: {
    x: 0,
    y: 400,
  },
};

describe('squareToQuad', () => {
  it.each([
    {
      name: 'rectangle',
      quad: rectangle,
    },
    {
      name: 'trapezoid',
      quad: trapezoid,
    },
  ])('maps the unit square corners onto the corners of a $name', ({ quad }) => {
    const { homography } = squareToQuad(quad, 1e-4);

    expect(mapPoint(homography, 0, 0)).toEqual(quad.topLeft);
    expect(mapPoint(homography, 1, 0)).toEqual(quad.topRight);
    expect(mapPoint(homography, 1, 1)).toEqual(quad.bottomRight);
    expect(mapPoint(homography, 0, 1)).toEqual(quad.bottomLeft);
  });

  it('is affine for a rectangle', () => {
    const { homography } = squareToQuad(rectangle, 1e-4);

    expect(Math.abs(homography.g)).toBe(0);
    expect(Math.abs(homography.h)).toBe(0);
    expect(mapPoint(homography, 0.5, 0.5)).toEqual({
      x: 300,
      y: 150,
    });
  });

  it('reports a zero determinant for a collapsed quad', () => {
    const line = {
      topLeft: {
        x: 0,
        y: 0,
      },
      topRight: {
        x: 10,
        y: 0,
      },
      bottomRight: {
        x: 20,
        y: 0,
      },
      bottomLeft: {
        x: 30,
        y: 0,
      },
    };

    const { determinant } = squareToQuad(line, 1e-6);

    expect(Math.abs(determinant)).toBe(0);
  });
});

describe('isConvexClockwise', () => {
  it.each([
    {
      name: 'rectangle',
      quad: rectangle,
    },
    {
      name: 'trapezoid',
      quad: trapezoid,
    },
  ])('accepts a $name', ({ quad }) => {
    const convex = isConvexClockwise(quad);

    expect(convex).toBe(true);
  });

  it.each([
    {
      name: 'rejects a dent at the top left corner',
      quad: {
        ...rectangle,
        topLeft: {
          x: 300,
          y: 150,
        },
      },
    },
    {
      name: 'rejects a dent at the bottom right corner',
      quad: {
        ...rectangle,
        bottomRight: {
          x: 300,
          y: 150,
        },
      },
    },
    {
      name: 'rejects a self-intersecting quad',
      quad: {
        ...rectangle,
        topRight: rectangle.bottomRight,
        bottomRight: rectangle.topRight,
      },
    },
    {
      name: 'rejects a mirrored quad',
      quad: {
        topLeft: rectangle.topRight,
        topRight: rectangle.topLeft,
        bottomRight: rectangle.bottomLeft,
        bottomLeft: rectangle.bottomRight,
      },
    },
    {
      name: 'rejects three corners on a line',
      quad: {
        ...rectangle,
        bottomRight: {
          x: 300,
          y: 50,
        },
      },
    },
  ])('$name', ({ quad }) => {
    const convex = isConvexClockwise(quad);

    expect(convex).toBe(false);
  });
});

describe('mapPoint', () => {
  it('maps the center of a trapezoid closer to its narrow far edge than the midpoint', () => {
    const { homography } = squareToQuad(trapezoid, 1e-4);

    const center = mapPoint(homography, 0.5, 0.5);

    expect(center.x).toBe(400);
    expect(center.y).toBeLessThan(250);
  });
});

describe('toMatrix3d', () => {
  it('normalizes the coefficients against the virtual size in column-major order', () => {
    const homography = {
      a: 400,
      b: 0,
      c: 100,
      d: 0,
      e: 200,
      f: 50,
      g: 0,
      h: 0,
    };

    const matrix = toMatrix3d(homography, 2000, 200);

    expect(matrix).toBe('matrix3d(0.2, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 100, 50, 0, 1)');
  });

  it('places the corners of the virtual element onto the quad', () => {
    const width = 2000;
    const height = 200;
    const { homography } = squareToQuad(trapezoid, 1e-4);

    const values = toMatrix3d(homography, width, height)
      .replace(/^matrix3d\(|\)$/g, '')
      .split(',')
      .map(Number);

    /**
     * project maps a point like CSS does, which multiplies the column vector (x, y, 0, 1)
     * and divides by the fourth component.
     */
    function project(x: number, y: number): { x: number; y: number } {
      const column = (index: number) => values[index] ?? 0;
      const w = column(3) * x + column(7) * y + column(15);
      return {
        x: Math.round((column(0) * x + column(4) * y + column(12)) / w),
        y: Math.round((column(1) * x + column(5) * y + column(13)) / w),
      };
    }

    expect(project(0, 0)).toEqual(trapezoid.topLeft);
    expect(project(width, 0)).toEqual(trapezoid.topRight);
    expect(project(width, height)).toEqual(trapezoid.bottomRight);
    expect(project(0, height)).toEqual(trapezoid.bottomLeft);
  });
});
