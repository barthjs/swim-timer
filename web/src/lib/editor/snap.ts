import type { Bounds, Point, Size } from '@/lib/editor/geometry';
import type { Gap } from '@/lib/editor/spacing';

/**
 * How close a dragged item has to come to a target to snap onto it, in pixels of the screen.
 * The editor divides it by its scale, so the reach stays the same at every zoom.
 */
export const SNAP_DISTANCE = 8;

/** Segment is a straight line between two points, such as the edge of an item. */
export type Segment = [start: Point, end: Point];

/** PointTargets are the corners and edges a dragged point can snap onto. */
export type PointTargets = {
  corners: Point[];
  edges: Segment[];
};

/** AxisTargets are the vertical lines at x and the horizontal lines at y that edges and centers align to. */
export type AxisTargets = {
  x: number[];
  y: number[];
};

/** SnapLine is a vertical line at a value of x or a horizontal line at a value of y. */
export type SnapLine = {
  axis: 'x' | 'y';
  value: number;
};

/**
 * SnapFeedback is what the editor shows of a drag: the point snapped onto, the lines aligned to,
 * and the distances to the neighbors.
 */
export type SnapFeedback = {
  point: Point | null;
  lines: SnapLine[];
  gaps: Gap[];
};

/** SnapResult is where a dragged point lands and what it snapped onto. */
export type SnapResult = {
  // The position the dragged point snaps to, or the point itself without a target in reach.
  point: Point;
  // The corner or the point on an edge it snapped to, or null.
  target: Point | null;
  distance: number;
};

/** noSnap returns the feedback of a drag that snapped to nothing. */
export function noSnap(): SnapFeedback {
  return {
    point: null,
    lines: [],
    gaps: [],
  };
}

/**
 * boundsTargets collects the edges and centers of the given bounds and of the content,
 * so items can be aligned with each other and with the middle and the border of the content.
 */
export function boundsTargets(list: readonly Bounds[], content: Size): AxisTargets {
  const x = new Set<number>([0, content.width / 2, content.width]);
  const y = new Set<number>([0, content.height / 2, content.height]);
  for (const bounds of list) {
    x.add(bounds.minX);
    x.add((bounds.minX + bounds.maxX) / 2);
    x.add(bounds.maxX);
    y.add(bounds.minY);
    y.add((bounds.minY + bounds.maxY) / 2);
    y.add(bounds.maxY);
  }
  return {
    x: [...x],
    y: [...y],
  };
}

/** mergeTargets combines the lines of several sources of targets. */
export function mergeTargets(...sources: readonly AxisTargets[]): AxisTargets {
  return {
    x: sources.flatMap((source) => source.x),
    y: sources.flatMap((source) => source.y),
  };
}

/** closestPointOnSegment projects a point onto a segment, clamped to its ends. */
export function closestPointOnSegment(point: Point, [start, end]: Segment): Point {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  const lengthSquared = dx * dx + dy * dy;
  if (lengthSquared === 0) return start;

  const t = ((point.x - start.x) * dx + (point.y - start.y) * dy) / lengthSquared;
  const clamped = Math.min(1, Math.max(0, t));
  return {
    x: start.x + clamped * dx,
    y: start.y + clamped * dy,
  };
}

/**
 * snapPoint moves a point onto the nearest target corner within reach. Without one, it moves the point
 * onto the nearest edge within reach. Corners take precedence, so two items meet exactly at their corners
 * instead of anywhere along an edge.
 */
export function snapPoint(point: Point, targets: PointTargets, maxDistance: number): SnapResult {
  let best: SnapResult = {
    point,
    target: null,
    distance: Infinity,
  };

  for (const corner of targets.corners) {
    const distance = distanceBetween(point, corner);
    if (distance <= maxDistance && distance < best.distance) {
      best = {
        point: corner,
        target: corner,
        distance,
      };
    }
  }
  if (best.target) return best;

  for (const edge of targets.edges) {
    const projected = closestPointOnSegment(point, edge);
    const distance = distanceBetween(point, projected);
    if (distance <= maxDistance && distance < best.distance) {
      best = {
        point: projected,
        target: projected,
        distance,
      };
    }
  }
  return best;
}

/**
 * snapOffset finds the shift that snaps the closest of several points moved together onto a target.
 * It returns no shift without a target in reach.
 */
export function snapOffset(
  points: readonly Point[],
  targets: PointTargets,
  maxDistance: number,
): { offset: Point; target: Point | null } {
  let best: SnapResult | null = null;
  let from: Point | null = null;

  for (const point of points) {
    const result = snapPoint(point, targets, maxDistance);
    if (result.target && (!best || result.distance < best.distance)) {
      best = result;
      from = point;
    }
  }

  if (!best || !from) {
    return {
      offset: {
        x: 0,
        y: 0,
      },
      target: null,
    };
  }
  return {
    offset: {
      x: best.point.x - from.x,
      y: best.point.y - from.y,
    },
    target: best.target,
  };
}

/**
 * snapValue finds the shift that moves the closest of several values moved together onto a target line.
 * It returns no shift and no line without a target in reach.
 */
export function snapValue(
  values: readonly number[],
  targets: readonly number[],
  maxDistance: number,
): { delta: number; line: number | null } {
  let delta = 0;
  let line: number | null = null;
  let distance = Infinity;

  for (const value of values) {
    for (const target of targets) {
      const current = Math.abs(target - value);
      if (current <= maxDistance && current < distance) {
        distance = current;
        delta = target - value;
        line = target;
      }
    }
  }
  return {
    delta,
    line,
  };
}

/**
 * snapBounds finds the shift that aligns the left edge, the center, or the right edge of moved bounds
 * with a vertical line, and the top edge, the middle, or the bottom edge with a horizontal line.
 * Where both axes snap, the lines cross at the point the feedback marks.
 */
export function snapBounds(
  bounds: Bounds,
  targets: AxisTargets,
  maxDistance: number,
): { offset: Point; feedback: SnapFeedback } {
  const x = snapValue(
    [bounds.minX, (bounds.minX + bounds.maxX) / 2, bounds.maxX],
    targets.x,
    maxDistance,
  );
  const y = snapValue(
    [bounds.minY, (bounds.minY + bounds.maxY) / 2, bounds.maxY],
    targets.y,
    maxDistance,
  );
  return {
    offset: {
      x: x.delta,
      y: y.delta,
    },
    feedback: axisFeedback(x.line, y.line),
  };
}

/**
 * snapPointToAxes moves a point onto the vertical and the horizontal line within reach,
 * so a point can be aligned with edges and centers it does not touch.
 */
export function snapPointToAxes(
  point: Point,
  targets: AxisTargets,
  maxDistance: number,
): { point: Point; feedback: SnapFeedback } {
  const x = snapValue([point.x], targets.x, maxDistance);
  const y = snapValue([point.y], targets.y, maxDistance);
  return {
    point: {
      x: point.x + x.delta,
      y: point.y + y.delta,
    },
    feedback: axisFeedback(x.line, y.line),
  };
}

/** axisFeedback shows the lines a drag aligned to, with the point where they cross if both axes snapped. */
export function axisFeedback(x: number | null, y: number | null): SnapFeedback {
  const lines: SnapLine[] = [];
  if (x !== null) {
    lines.push({
      axis: 'x',
      value: x,
    });
  }
  if (y !== null) {
    lines.push({
      axis: 'y',
      value: y,
    });
  }
  return {
    point:
      x !== null && y !== null
        ? {
            x,
            y,
          }
        : null,
    lines,
    gaps: [],
  };
}

/** distanceBetween returns the distance between two points. */
function distanceBetween(a: Point, b: Point): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}
