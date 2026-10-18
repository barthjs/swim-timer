import { sizeBounds, unionBounds, type Bounds, type Point, type Size } from '@/lib/editor/geometry';

/** AlignEdge is the edge or the center line that items are lined up at. */
export type AlignEdge = 'left' | 'centerX' | 'right' | 'top' | 'centerY' | 'bottom';

/** DistributeAxis is the direction in which items are spaced evenly. */
export type DistributeAxis = 'x' | 'y';

/** ALIGN_EDGES lists the edges in the order of a menu, the horizontal ones first. */
export const ALIGN_EDGES: readonly AlignEdge[] = [
  'left',
  'centerX',
  'right',
  'top',
  'centerY',
  'bottom',
];

/** DISTRIBUTE_MIN is the number of items distributing needs, since two items have no gap to even out. */
export const DISTRIBUTE_MIN = 3;

/**
 * alignReference returns what items are aligned to: the bounds of several items together,
 * or the content for a single item, so one item can be centered or moved to the border.
 */
export function alignReference(list: readonly Bounds[], content: Size): Bounds {
  if (list.length < 2) return sizeBounds(content);
  return unionBounds(list) ?? sizeBounds(content);
}

/** alignOffsets returns the shift of each bounds that lines it up with an edge of the reference. */
export function alignOffsets(list: readonly Bounds[], edge: AlignEdge, reference: Bounds): Point[] {
  return list.map((bounds) => {
    switch (edge) {
      case 'left':
        return offset(reference.minX - bounds.minX, 0);
      case 'centerX':
        return offset(center(reference.minX, reference.maxX) - center(bounds.minX, bounds.maxX), 0);
      case 'right':
        return offset(reference.maxX - bounds.maxX, 0);
      case 'top':
        return offset(0, reference.minY - bounds.minY);
      case 'centerY':
        return offset(0, center(reference.minY, reference.maxY) - center(bounds.minY, bounds.maxY));
      case 'bottom':
        return offset(0, reference.maxY - bounds.maxY);
    }
  });
}

/**
 * distributeOffsets returns the shift of each bounds that spaces them evenly along an axis. The items keep
 * the order of their start, the first and the last one stay in place, and the others move so that all gaps
 * between neighbors are equal. Fewer than DISTRIBUTE_MIN items stay in place.
 */
export function distributeOffsets(list: readonly Bounds[], axis: DistributeAxis): Point[] {
  if (list.length < DISTRIBUTE_MIN) return list.map(() => offset(0, 0));

  const spans = list.map((bounds, index) => ({
    index,
    start: axis === 'x' ? bounds.minX : bounds.minY,
    end: axis === 'x' ? bounds.maxX : bounds.maxY,
  }));
  spans.sort((a, b) => a.start - b.start);

  const first = spans[0]!;
  const last = spans[spans.length - 1]!;
  const occupied = spans.reduce((sum, span) => sum + (span.end - span.start), 0);
  const gap = (last.end - first.start - occupied) / (spans.length - 1);

  const shifts = list.map(() => 0);
  let position = first.start;
  for (const span of spans) {
    shifts[span.index] = position - span.start;
    position += span.end - span.start + gap;
  }

  return shifts.map((shift) => (axis === 'x' ? offset(shift, 0) : offset(0, shift)));
}

/** offset builds a shift by x and y. */
function offset(x: number, y: number): Point {
  return {
    x,
    y,
  };
}

/** center returns the middle between two values. */
function center(start: number, end: number): number {
  return (start + end) / 2;
}
