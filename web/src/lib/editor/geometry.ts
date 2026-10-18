/** Point is a position in the content units of an editor. */
export type Point = {
  x: number;
  y: number;
};

/** Bounds is an axis-aligned rectangle given by its edges. */
export type Bounds = {
  minX: number;
  maxX: number;
  minY: number;
  maxY: number;
};

/** Size is a width and a height, such as of the content of an editor or of an image. */
export type Size = {
  width: number;
  height: number;
};

/** pointsBounds returns the smallest bounds that contain every point. */
export function pointsBounds(points: readonly Point[]): Bounds {
  return {
    minX: Math.min(...points.map((point) => point.x)),
    maxX: Math.max(...points.map((point) => point.x)),
    minY: Math.min(...points.map((point) => point.y)),
    maxY: Math.max(...points.map((point) => point.y)),
  };
}

/** unionBounds returns the smallest bounds that contain every given bounds, or null for none. */
export function unionBounds(list: readonly Bounds[]): Bounds | null {
  if (list.length === 0) return null;
  return {
    minX: Math.min(...list.map((bounds) => bounds.minX)),
    maxX: Math.max(...list.map((bounds) => bounds.maxX)),
    minY: Math.min(...list.map((bounds) => bounds.minY)),
    maxY: Math.max(...list.map((bounds) => bounds.maxY)),
  };
}

/** sizeBounds returns the bounds of content of a size, starting at the origin. */
export function sizeBounds(size: Size): Bounds {
  return {
    minX: 0,
    maxX: size.width,
    minY: 0,
    maxY: size.height,
  };
}

/** boundsIntersect reports whether two bounds overlap. */
export function boundsIntersect(a: Bounds, b: Bounds): boolean {
  return a.minX < b.maxX && a.maxX > b.minX && a.minY < b.maxY && a.maxY > b.minY;
}

/**
 * ASPECT_RATIO_TOLERANCE is the relative difference below which two aspect ratios count as equal,
 * which absorbs sizes rounded to whole pixels.
 */
export const ASPECT_RATIO_TOLERANCE = 0.01;

/**
 * aspectRatioDiffers reports whether two sizes differ in their aspect ratio by more than the tolerance,
 * which would stretch one of them when it is drawn into the other. An empty size never differs.
 */
export function aspectRatioDiffers(a: Size, b: Size, tolerance = ASPECT_RATIO_TOLERANCE): boolean {
  if (a.width <= 0 || a.height <= 0 || b.width <= 0 || b.height <= 0) return false;
  const ratioA = a.width / a.height;
  const ratioB = b.width / b.height;
  return Math.abs(ratioA / ratioB - 1) > tolerance;
}

/** clamp limits a value to a range. A range whose maximum is below its minimum collapses to the minimum. */
export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), Math.max(min, max));
}

/**
 * clampOffset limits a shift of bounds, so they stay inside a container. Bounds larger than the container
 * keep their start at its start.
 */
export function clampOffset(bounds: Bounds, container: Bounds, offset: Point): Point {
  return {
    x: clamp(offset.x, container.minX - bounds.minX, container.maxX - bounds.maxX),
    y: clamp(offset.y, container.minY - bounds.minY, container.maxY - bounds.maxY),
  };
}

/** translateBounds returns bounds moved by an offset. */
export function translateBounds(bounds: Bounds, offset: Point): Bounds {
  return {
    minX: bounds.minX + offset.x,
    maxX: bounds.maxX + offset.x,
    minY: bounds.minY + offset.y,
    maxY: bounds.maxY + offset.y,
  };
}

/**
 * constrainToAxis keeps only the larger part of an offset, so a drag follows the horizontal or the vertical axis,
 * whichever the pointer moved along more.
 */
export function constrainToAxis(offset: Point): Point {
  if (Math.abs(offset.x) >= Math.abs(offset.y)) {
    return {
      x: offset.x,
      y: 0,
    };
  }
  return {
    x: 0,
    y: offset.y,
  };
}

/**
 * mapPointBetween moves a point from one bounds into another, keeping its relative position,
 * so a group of items scales and moves with the bounds around it. Bounds without an extent keep the relative start.
 */
export function mapPointBetween(point: Point, from: Bounds, to: Bounds): Point {
  const width = from.maxX - from.minX;
  const height = from.maxY - from.minY;
  const relativeX = width > 0 ? (point.x - from.minX) / width : 0;
  const relativeY = height > 0 ? (point.y - from.minY) / height : 0;
  return {
    x: to.minX + relativeX * (to.maxX - to.minX),
    y: to.minY + relativeY * (to.maxY - to.minY),
  };
}
