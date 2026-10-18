import type { Bounds, Size } from '@/lib/editor/geometry';

/**
 * Gap is a distance between two edges along an axis, drawn at a position across it:
 * along x from `from` to `to` at height `at`, or along y at width `at`.
 */
export type Gap = {
  axis: 'x' | 'y';
  from: number;
  to: number;
  at: number;
};

/**
 * measureGaps returns the distance from moved bounds to the nearest other bounds on each side, among those
 * that face it across the axis, or to the border of the content where nothing faces it.
 */
export function measureGaps(moved: Bounds, others: readonly Bounds[], content: Size): Gap[] {
  const facingX = others.filter((other) =>
    overlaps(other.minY, other.maxY, moved.minY, moved.maxY),
  );
  const facingY = others.filter((other) =>
    overlaps(other.minX, other.maxX, moved.minX, moved.maxX),
  );
  const middleY = (moved.minY + moved.maxY) / 2;
  const middleX = (moved.minX + moved.maxX) / 2;

  const left = Math.max(
    0,
    ...facingX.filter((other) => other.maxX <= moved.minX).map((other) => other.maxX),
  );
  const right = Math.min(
    content.width,
    ...facingX.filter((other) => other.minX >= moved.maxX).map((other) => other.minX),
  );
  const top = Math.max(
    0,
    ...facingY.filter((other) => other.maxY <= moved.minY).map((other) => other.maxY),
  );
  const bottom = Math.min(
    content.height,
    ...facingY.filter((other) => other.minY >= moved.maxY).map((other) => other.minY),
  );

  const gaps: Gap[] = [
    {
      axis: 'x',
      from: left,
      to: moved.minX,
      at: middleY,
    },
    {
      axis: 'x',
      from: moved.maxX,
      to: right,
      at: middleY,
    },
    {
      axis: 'y',
      from: top,
      to: moved.minY,
      at: middleX,
    },
    {
      axis: 'y',
      from: moved.maxY,
      to: bottom,
      at: middleX,
    },
  ];
  return gaps.filter((gap) => gap.to > gap.from);
}

/**
 * spacingTargets returns where moved bounds start when their gaps to the neighbors are even: centered between
 * the nearest neighbors on both sides, or next to a neighbor at a gap two other items already have.
 * The values are positions of the left edge for x and of the top edge for y.
 */
export function spacingTargets(
  moved: Bounds,
  others: readonly Bounds[],
): { x: number[]; y: number[] } {
  return {
    x: axisSpacing(
      moved.minX,
      moved.maxX,
      others
        .filter((other) => overlaps(other.minY, other.maxY, moved.minY, moved.maxY))
        .map((other) => [other.minX, other.maxX]),
    ),
    y: axisSpacing(
      moved.minY,
      moved.maxY,
      others
        .filter((other) => overlaps(other.minX, other.maxX, moved.minX, moved.maxX))
        .map((other) => [other.minY, other.maxY]),
    ),
  };
}

/** axisSpacing collects the even starts of an extent among the extents that face it along one axis. */
function axisSpacing(start: number, end: number, extents: Array<[number, number]>): number[] {
  const size = end - start;
  const before = extents.filter(([, otherEnd]) => otherEnd <= start + size / 2);
  const after = extents.filter(([otherStart]) => otherStart >= end - size / 2);
  const nearestBefore = Math.max(...before.map(([, otherEnd]) => otherEnd));
  const nearestAfter = Math.min(...after.map(([otherStart]) => otherStart));

  const targets: number[] = [];
  if (Number.isFinite(nearestBefore) && Number.isFinite(nearestAfter)) {
    targets.push((nearestBefore + nearestAfter - size) / 2);
  }

  const sorted = [...extents].sort((a, b) => a[0] - b[0]);
  for (let i = 1; i < sorted.length; i++) {
    const gap = sorted[i]![0] - sorted[i - 1]![1];
    if (gap <= 0) continue;
    if (Number.isFinite(nearestBefore)) targets.push(nearestBefore + gap);
    if (Number.isFinite(nearestAfter)) targets.push(nearestAfter - gap - size);
  }
  return targets;
}

/** overlaps reports whether two ranges share a part. */
function overlaps(startA: number, endA: number, startB: number, endB: number): boolean {
  return startA < endB && endA > startB;
}
