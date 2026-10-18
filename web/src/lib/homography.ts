import type { Point } from '@/lib/editor/geometry';

/** Quad lists the corners of a quadrilateral clockwise, starting top left. */
export type Quad = {
  topLeft: Point;
  topRight: Point;
  bottomRight: Point;
  bottomLeft: Point;
};

/**
 * Homography holds the coefficients of a projective transform that maps normalized coordinates
 * (u, v in [0, 1]) to pixels: x = (a*u + b*v + c) / (g*u + h*v + 1), y = (d*u + e*v + f) / (g*u + h*v + 1).
 */
export type Homography = {
  a: number;
  b: number;
  c: number;
  d: number;
  e: number;
  f: number;
  g: number;
  h: number;
};

/**
 * squareToQuad computes the projective transform that maps the unit square onto a quad,
 * which reproduces a perspective distortion. g and h are the perspective factors.
 * If the absolute determinant does not exceed minDeterminant, the quad is degenerate
 * (for example corners collapsing into a line), the perspective factors stay 0, and the transform is affine.
 * The determinant is returned so callers can reject such quads.
 */
export function squareToQuad(
  quad: Quad,
  minDeterminant: number,
): { homography: Homography; determinant: number } {
  const { x: x0, y: y0 } = quad.topLeft;
  const { x: x1, y: y1 } = quad.topRight;
  const { x: x2, y: y2 } = quad.bottomRight;
  const { x: x3, y: y3 } = quad.bottomLeft;

  const sumX = x0 - x1 + x2 - x3;
  const sumY = y0 - y1 + y2 - y3;
  const dx1 = x1 - x2;
  const dx2 = x3 - x2;
  const dy1 = y1 - y2;
  const dy2 = y3 - y2;

  const determinant = dx1 * dy2 - dx2 * dy1;

  let g = 0;
  let h = 0;
  if (Math.abs(determinant) > minDeterminant) {
    g = (sumX * dy2 - sumY * dx2) / determinant;
    h = (dx1 * sumY - dy1 * sumX) / determinant;
  }

  return {
    homography: {
      a: x1 - x0 + g * x1,
      b: x3 - x0 + h * x3,
      c: x0,
      d: y1 - y0 + g * y1,
      e: y3 - y0 + h * y3,
      f: y0,
      g,
      h,
    },
    determinant,
  };
}

/**
 * isConvexClockwise reports whether the corners of a quad form a convex shape, clockwise on screen starting top left.
 * Only such a quad can be the image of a rectangle, for any other the denominator of the projective transform
 * becomes zero inside it.
 */
export function isConvexClockwise(quad: Quad): boolean {
  const corners = [quad.topLeft, quad.topRight, quad.bottomRight, quad.bottomLeft];

  for (let i = 0; i < corners.length; i++) {
    const current = corners[i]!;
    const next = corners[(i + 1) % corners.length]!;
    const afterNext = corners[(i + 2) % corners.length]!;

    const edgeX = next.x - current.x;
    const edgeY = next.y - current.y;
    const nextEdgeX = afterNext.x - next.x;
    const nextEdgeY = afterNext.y - next.y;

    if (edgeX * nextEdgeY - edgeY * nextEdgeX <= 0) {
      return false;
    }
  }

  return true;
}

/**
 * mapPoint maps normalized coordinates (u horizontal, v vertical, both in [0, 1]) to rounded pixel coordinates.
 */
export function mapPoint(homography: Homography, u: number, v: number): Point {
  const { a, b, c, d, e, f, g, h } = homography;
  const denominator = g * u + h * v + 1;
  return {
    x: Math.round((a * u + b * v + c) / denominator),
    y: Math.round((d * u + e * v + f) / denominator),
  };
}

/**
 * toMatrix3d expresses the transform as a CSS matrix3d for an element of the given virtual size
 * positioned at the origin, with transform-origin 0 0. The coefficients are normalized against
 * the virtual size, since the element's own pixels replace the unit square.
 */
export function toMatrix3d(homography: Homography, width: number, height: number): string {
  const { a, b, c, d, e, f, g, h } = homography;
  return `matrix3d(${a / width}, ${d / width}, 0, ${g / width}, ${b / height}, ${e / height}, 0, ${h / height}, 0, 0, 1, 0, ${c}, ${f}, 0, 1)`;
}
