import {
  CornerKey,
  type Fraction,
  type LiveHeat,
  type PresetLaneRequest,
  type PresetLaneResponse,
  type PresetResponse,
  type PresetSaveRequest,
  TextAnchor,
} from '@/api/types';
import { pointsBounds, type Bounds, type Point } from '@/lib/editor/geometry';
import type { PointTargets, Segment } from '@/lib/editor/snap';
import type { FrameSize } from '@/lib/frame';
import { isConvexClockwise, mapPoint, squareToQuad, toMatrix3d, type Quad } from '@/lib/homography';

/**
 * Below this absolute determinant, squareToQuad drops the perspective.
 * A valid quad, which every caller checks first with isConvexClockwise, always stays far above it.
 */
const MIN_DETERMINANT = 0.0001;

/** Size and spacing of the lanes of a new preset, in pixels of the reference resolution. */
const NEW_LANE_HEIGHT = 50;
const NEW_LANE_GAP = 10;
const NEW_LANES_TOP = 100;

/**
 * Decimal places of a stored fraction. A millionth stays below a hundredth
 * of a pixel up to a reference resolution of 10000 pixels.
 */
const FRACTION_DIGITS = 6;

/** Decimal places of a pixel coordinate in the editor, which hides the rounding of the stored fractions. */
const PIXEL_DIGITS = 2;

/** How far an imported fraction may lie outside 0 to 1 to be clamped instead of rejected. */
const FRACTION_TOLERANCE = 0.001;

export const CORNERS = Object.values(CornerKey);

/** The reference resolution of a new preset, the usual size of a browser source. */
export const DEFAULT_FRAME_WIDTH = 1920;
export const DEFAULT_FRAME_HEIGHT = 1080;

/** The range the backend accepts for the reference resolution of a preset. */
export const DIMENSION_MIN = 1;
export const DIMENSION_MAX = 100000;

/** The text size of a new preset and the range the backend accepts, in percent of the lane height. */
export const DEFAULT_TEXT_SIZE = 64;
export const TEXT_SIZE_MIN = 10;
export const TEXT_SIZE_MAX = 100;

/**
 * The lane gap of a new preset and the range the backend accepts, in percent of the lane height.
 * 19 percent gives the lanes of an 8-lane pool a gap of about 2 percent of the pool height.
 */
export const DEFAULT_LANE_GAP = 19;
export const LANE_GAP_MIN = 0;
export const LANE_GAP_MAX = 100;

/**
 * The size of the content box of a lane before it is projected onto the lane. All lanes share one box, since
 * the perspective comes from the projection alone. It is large, so the text stays sharp where the projection
 * enlarges it.
 */
export const LANE_CONTENT_WIDTH = 2000;
export const LANE_CONTENT_HEIGHT = 200;

/**
 * The maximum lengths the backend accepts for the name and the comment of a preset. The name is the file name
 * of the preset, and 60 characters keep it within the file name limit in any script.
 */
export const PRESET_NAME_MAX = 60;
export const PRESET_COMMENT_MAX = 1000;

/** The lowest lane number of the pool while no heat is known. */
export const DEFAULT_FIRST_LANE = 1;

declare const pixelBrand: unique symbol;

/** PixelPoint is a position in pixels of a frame. The brand keeps it apart from the stored fractions. */
export type PixelPoint = Point & { readonly [pixelBrand]: true };

/** LaneCorners are the corners of a lane in any coordinate space, for geometry that only reads them. */
export type LaneCorners = Record<CornerKey, Point>;

/** PixelLane is a lane with its corners in pixels of a frame. */
export type PixelLane = Record<CornerKey, PixelPoint> & {
  locked: boolean;
};

/**
 * EditorPreset is a preset as the editor edits it, with the lanes in pixels of the reference resolution
 * frameWidth x frameHeight. Only toPresetRequest turns it back into fractions.
 */
export type EditorPreset = Omit<PresetResponse, 'lanes'> & {
  lanes: Record<string, PixelLane>;
};

/** roundTo rounds a value to the given number of decimal places. */
export function roundTo(value: number, digits: number): number {
  const factor = 10 ** digits;
  return Math.round(value * factor) / factor;
}

/** toFraction keeps a share within 0 to 1 and rounds it to the precision a preset stores. */
export function toFraction(value: number): Fraction {
  return roundTo(Math.min(1, Math.max(0, value)), FRACTION_DIGITS) as Fraction;
}

/** pixelPoint creates a position in pixels of a frame. */
export function pixelPoint(x: number, y: number): PixelPoint {
  return {
    x,
    y,
  } as PixelPoint;
}

/** toPixelLane converts a stored lane into pixels of a frame of the given size. */
export function toPixelLane(lane: PresetLaneResponse, frame: FrameSize): PixelLane {
  const result = {
    locked: lane.locked,
  } as PixelLane;
  for (const corner of CORNERS) {
    result[corner] = pixelPoint(
      roundTo(lane[corner].x * frame.width, PIXEL_DIGITS),
      roundTo(lane[corner].y * frame.height, PIXEL_DIGITS),
    );
  }
  return result;
}

/** toPixelLanes converts every stored lane of a preset into pixels of a frame, see toPixelLane. */
export function toPixelLanes(
  lanes: Record<string, PresetLaneResponse>,
  frame: FrameSize,
): Record<string, PixelLane> {
  const result: Record<string, PixelLane> = {};
  for (const [key, lane] of Object.entries(lanes)) {
    result[key] = toPixelLane(lane, frame);
  }
  return result;
}

/** toFractionLane converts a lane in pixels of a frame into the fractions a preset stores. */
export function toFractionLane(
  lane: LaneCorners & { locked: boolean },
  frame: FrameSize,
): PresetLaneRequest {
  const result = {
    locked: lane.locked,
  } as PresetLaneRequest;
  for (const corner of CORNERS) {
    result[corner] = {
      x: toFraction(lane[corner].x / frame.width),
      y: toFraction(lane[corner].y / frame.height),
    };
  }
  return result;
}

/** toEditorPreset opens a stored preset in pixels of its reference resolution. */
export function toEditorPreset(preset: PresetResponse): EditorPreset {
  return {
    ...preset,
    lanes: toPixelLanes(preset.lanes, frameOf(preset)),
  };
}

/** toPresetRequest converts the preset of the editor back into the fractions the backend stores. */
export function toPresetRequest(preset: EditorPreset): PresetSaveRequest {
  const frame = frameOf(preset);
  const lanes: Record<string, PresetLaneRequest> = {};
  for (const [key, lane] of Object.entries(preset.lanes)) {
    lanes[key] = toFractionLane(lane, frame);
  }
  return {
    name: preset.name,
    comment: preset.comment,
    frameWidth: preset.frameWidth,
    frameHeight: preset.frameHeight,
    textSize: preset.textSize,
    laneGap: preset.laneGap,
    reverseLanes: preset.reverseLanes,
    textAnchor: preset.textAnchor,
    lanes,
  };
}

/** frameOf returns the reference resolution of a preset. */
export function frameOf(preset: Pick<PresetResponse, 'frameWidth' | 'frameHeight'>): FrameSize {
  return {
    width: preset.frameWidth,
    height: preset.frameHeight,
  };
}

/**
 * scaleLanes moves every corner from a frame of one size to the same relative position in a frame of another size,
 * so the lanes keep their place in the image when the reference resolution changes.
 */
export function scaleLanes(lanes: Record<string, PixelLane>, from: FrameSize, to: FrameSize): void {
  const scaleX = to.width / from.width;
  const scaleY = to.height / from.height;
  for (const lane of Object.values(lanes)) {
    for (const corner of CORNERS) {
      lane[corner].x = roundTo(lane[corner].x * scaleX, PIXEL_DIGITS);
      lane[corner].y = roundTo(lane[corner].y * scaleY, PIXEL_DIGITS);
    }
  }
}

/**
 * createPreset builds a preset with the default reference resolution and the given number of rectangular lanes,
 * stacked in the middle half of the image.
 */
export function createPreset(name: string, laneCount: number): PresetResponse {
  const frame = {
    width: DEFAULT_FRAME_WIDTH,
    height: DEFAULT_FRAME_HEIGHT,
  };
  const lanes: Record<string, PixelLane> = {};

  const left = frame.width / 4;
  const right = (frame.width * 3) / 4;

  for (let i = 1; i <= laneCount; i++) {
    const top = NEW_LANES_TOP + i * (NEW_LANE_HEIGHT + NEW_LANE_GAP);
    const bottom = top + NEW_LANE_HEIGHT;
    lanes[String(i)] = {
      [CornerKey.TopLeft]: pixelPoint(left, top),
      [CornerKey.TopRight]: pixelPoint(right, top),
      [CornerKey.BottomRight]: pixelPoint(right, bottom),
      [CornerKey.BottomLeft]: pixelPoint(left, bottom),
      locked: false,
    };
  }

  return toPresetRequest({
    name,
    comment: null,
    frameWidth: frame.width,
    frameHeight: frame.height,
    textSize: DEFAULT_TEXT_SIZE,
    laneGap: DEFAULT_LANE_GAP,
    reverseLanes: false,
    textAnchor: TextAnchor.Left,
    lanes,
  });
}

/** presetLaneCount returns the highest lane key of a preset, which numbers its lanes from 1. */
export function presetLaneCount(lanes: Record<string, unknown>): number {
  let count = 0;
  for (const key of Object.keys(lanes)) {
    const lane = Number(key);
    if (Number.isInteger(lane) && lane > count) count = lane;
  }
  return count;
}

/**
 * heatLaneNumber returns the lane of the heat a lane of a preset shows.
 * Preset lanes count from 1, top to bottom in the image, heat lanes count from firstLane,
 * and reversed lanes put the highest heat lane on top.
 */
export function heatLaneNumber(
  presetLane: number,
  laneCount: number,
  firstLane: number,
  reverseLanes: boolean,
): number {
  const offset = reverseLanes ? laneCount - presetLane : presetLane - 1;
  return firstLane + offset;
}

/**
 * liveFirstLane returns the lowest lane number of the pool from the live heat.
 * Before the first frame, the live heat carries no meet data, so the default applies.
 */
export function liveFirstLane(state: LiveHeat | null): number {
  if (!state || !state.status) return DEFAULT_FIRST_LANE;
  return state.firstLane;
}

/**
 * clampCoordinate keeps a pixel coordinate within a frame of the given length and rounds it to a whole pixel,
 * since a preset can only store positions inside the image.
 */
export function clampCoordinate(value: number, length: number): number {
  return Math.min(length, Math.max(0, Math.round(value)));
}

/** clampPoint keeps both coordinates of a point within a frame, see clampCoordinate. */
export function clampPoint(point: Point, frame: FrameSize): Point {
  return {
    x: clampCoordinate(point.x, frame.width),
    y: clampCoordinate(point.y, frame.height),
  };
}

/**
 * translateLane sets every corner of a lane to the corresponding corner of from, moved by dx and dy
 * and kept within the frame.
 */
export function translateLane(
  lane: PixelLane,
  from: LaneCorners,
  dx: number,
  dy: number,
  frame: FrameSize,
): void {
  for (const corner of CORNERS) {
    lane[corner].x = clampCoordinate(from[corner].x + dx, frame.width);
    lane[corner].y = clampCoordinate(from[corner].y + dy, frame.height);
  }
}

/** laneQuad returns the corners of a lane as a quad. */
export function laneQuad(lane: LaneCorners): Quad {
  return {
    topLeft: lane[CornerKey.TopLeft],
    topRight: lane[CornerKey.TopRight],
    bottomRight: lane[CornerKey.BottomRight],
    bottomLeft: lane[CornerKey.BottomLeft],
  };
}

/** isLaneValid reports whether a lane can be projected, see isConvexClockwise. */
export function isLaneValid(lane: LaneCorners): boolean {
  return isConvexClockwise(laneQuad(lane));
}

/** laneCenter returns the average of the four corners, where the editor labels a lane. */
export function laneCenter(lane: LaneCorners): Point {
  let x = 0;
  let y = 0;
  for (const corner of CORNERS) {
    x += lane[corner].x;
    y += lane[corner].y;
  }
  return {
    x: x / CORNERS.length,
    y: y / CORNERS.length,
  };
}

/**
 * laneStyle positions the content box of a lane onto the lane, as the AR overlay and the preview of the editor draw it.
 * The box lands in the coordinate space of the corners. A lane that cannot be projected is hidden instead of distorted.
 */
export function laneStyle(lane: LaneCorners): Record<string, string> {
  if (!isLaneValid(lane)) {
    return {
      display: 'none',
    };
  }

  const width = LANE_CONTENT_WIDTH;
  const height = LANE_CONTENT_HEIGHT;
  const { homography } = squareToQuad(laneQuad(lane), MIN_DETERMINANT);
  return {
    position: 'absolute',
    top: '0',
    left: '0',
    width: `${width}px`,
    height: `${height}px`,
    transformOrigin: '0 0',
    transform: toMatrix3d(homography, width, height),
  };
}

/** laneBounds returns the axis-aligned bounds of a lane, which the selection and the alignment work with. */
export function laneBounds(lane: LaneCorners): Bounds {
  return pointsBounds(CORNERS.map((corner) => lane[corner]));
}

/**
 * lanePointTargets collects the corners and edges of every lane except the excluded ones,
 * plus the edges of the image, so a corner can be placed against its neighbors and the image border.
 */
export function lanePointTargets(
  lanes: Record<string, LaneCorners>,
  excluded: ReadonlySet<string>,
  frame: FrameSize,
): PointTargets {
  const corners: Point[] = [];
  const edges: Segment[] = [];

  for (const [key, lane] of Object.entries(lanes)) {
    if (excluded.has(key)) continue;

    for (let i = 0; i < CORNERS.length; i++) {
      const start = lane[CORNERS[i]!];
      const end = lane[CORNERS[(i + 1) % CORNERS.length]!];
      corners.push(start);
      edges.push([start, end]);
    }
  }

  const topLeft = {
    x: 0,
    y: 0,
  };
  const topRight = {
    x: frame.width,
    y: 0,
  };
  const bottomRight = {
    x: frame.width,
    y: frame.height,
  };
  const bottomLeft = {
    x: 0,
    y: frame.height,
  };
  edges.push(
    [topLeft, topRight],
    [topRight, bottomRight],
    [bottomRight, bottomLeft],
    [bottomLeft, topLeft],
  );

  return {
    corners,
    edges,
  };
}

/** laneBoundsExcluding returns the bounds of every lane except the excluded ones, which edges align with. */
export function laneBoundsExcluding(
  lanes: Record<string, LaneCorners>,
  excluded: ReadonlySet<string>,
): Bounds[] {
  return Object.entries(lanes)
    .filter(([key]) => !excluded.has(key))
    .map(([, lane]) => laneBounds(lane));
}

/**
 * alignLanes distributes the lanes evenly between the top edge of lane 1 and the bottom edge of the last lane,
 * with a gap of laneGap percent of a lane height between two lanes. It follows the perspective of the outline,
 * so the lanes keep their proportions in an oblique view. Lanes 1 and the last lane must be positioned by hand first.
 * Locked lanes keep their position. It reports whether the lanes could be aligned, which fails if lane 1
 * or the last lane is missing or the outline they span is not a valid quad.
 */
export function alignLanes(
  lanes: Record<string, PixelLane>,
  frame: FrameSize,
  laneGap: number,
): boolean {
  const maxLane = Math.max(...Object.keys(lanes).map(Number));
  const first = lanes['1'];
  const last = lanes[String(maxLane)];
  if (!first || !last) return false;

  const pool: Quad = {
    topLeft: {
      ...first[CornerKey.TopLeft],
    },
    topRight: {
      ...first[CornerKey.TopRight],
    },
    bottomRight: {
      ...last[CornerKey.BottomRight],
    },
    bottomLeft: {
      ...last[CornerKey.BottomLeft],
    },
  };
  if (!isConvexClockwise(pool)) return false;

  const { homography } = squareToQuad(pool, MIN_DETERMINANT);

  // n lanes and n - 1 gaps of laneGap percent of a lane fill the pool height
  const gapFactor = laneGap / 100;
  const laneRatio = 1 / (maxLane + (maxLane - 1) * gapFactor);
  const gapRatio = laneRatio * gapFactor;

  for (let i = 1; i <= maxLane; i++) {
    const lane = lanes[String(i)];
    if (!lane || lane.locked) continue;

    const top = (i - 1) * (laneRatio + gapRatio);
    const bottom = top + laneRatio;

    movePoint(lane[CornerKey.TopLeft], clampPoint(mapPoint(homography, 0, top), frame));
    movePoint(lane[CornerKey.TopRight], clampPoint(mapPoint(homography, 1, top), frame));
    movePoint(lane[CornerKey.BottomLeft], clampPoint(mapPoint(homography, 0, bottom), frame));
    movePoint(lane[CornerKey.BottomRight], clampPoint(mapPoint(homography, 1, bottom), frame));
  }

  return true;
}

/** movePoint sets a point to the position of another one, keeping its coordinate space. */
function movePoint(point: PixelPoint, to: Point): void {
  point.x = to.x;
  point.y = to.y;
}

/** polygonPoints formats the corners of a lane as the points attribute of an SVG polygon. */
export function polygonPoints(lane: LaneCorners): string {
  const order: CornerKey[] = [
    CornerKey.TopLeft,
    CornerKey.TopRight,
    CornerKey.BottomRight,
    CornerKey.BottomLeft,
  ];
  return order.map((corner) => pointString(lane[corner])).join(' ');
}

/** pointString formats a point as a coordinate pair of an SVG polygon. */
function pointString(point: Point): string {
  return `${point.x},${point.y}`;
}

/**
 * parsePreset reads a preset from the JSON of an exported preset file. It only checks the shape the editor needs
 * to display it, the backend validates the remaining values on save. It returns null if the content is not a preset.
 */
export function parsePreset(json: string): PresetResponse | null {
  let value: unknown;
  try {
    value = JSON.parse(json);
  } catch {
    return null;
  }

  if (!isRecord(value) || !isRecord(value.lanes)) return null;
  if (!isDimension(value.frameWidth) || !isDimension(value.frameHeight)) return null;
  if (typeof value.textSize !== 'number') return null;
  if (!isLaneGap(value.laneGap)) return null;

  const lanes = parseLanes(value.lanes);
  if (!lanes) return null;

  return {
    name: typeof value.name === 'string' ? value.name : '',
    comment: typeof value.comment === 'string' ? value.comment : null,
    frameWidth: value.frameWidth,
    frameHeight: value.frameHeight,
    textSize: value.textSize,
    laneGap: value.laneGap,
    reverseLanes: value.reverseLanes === true,
    textAnchor: value.textAnchor === TextAnchor.Right ? TextAnchor.Right : TextAnchor.Left,
    lanes,
  };
}

/** isDimension reports whether a value is a whole number of pixels the backend accepts as a size. */
export function isDimension(value: unknown): value is number {
  return (
    typeof value === 'number' &&
    Number.isInteger(value) &&
    value >= DIMENSION_MIN &&
    value <= DIMENSION_MAX
  );
}

/** isLaneGap reports whether a value is a whole percentage the backend accepts as a lane gap. */
export function isLaneGap(value: unknown): value is number {
  return (
    typeof value === 'number' &&
    Number.isInteger(value) &&
    value >= LANE_GAP_MIN &&
    value <= LANE_GAP_MAX
  );
}

/** parseLanes reads the lanes of a preset file. It returns null if there are none or one of them is invalid. */
function parseLanes(value: Record<string, unknown>): Record<string, PresetLaneResponse> | null {
  const entries = Object.entries(value);
  if (entries.length === 0) return null;

  const lanes: Record<string, PresetLaneResponse> = {};
  for (const [key, lane] of entries) {
    if (!/^[1-9][0-9]*$/.test(key) || !isRecord(lane)) return null;

    const parsed = {
      locked: lane.locked === true,
    } as PresetLaneResponse;
    for (const corner of CORNERS) {
      const point = lane[corner];
      if (!isRecord(point)) return null;
      const x = parseFraction(point.x);
      const y = parseFraction(point.y);
      if (x === null || y === null) return null;
      parsed[corner] = {
        x,
        y,
      };
    }
    lanes[key] = parsed;
  }
  return lanes;
}

/**
 * parseFraction reads a coordinate of a preset file. Values within the tolerance around 0 to 1 are clamped,
 * as rounding in other tools may leave them, every other value is rejected with null.
 */
function parseFraction(value: unknown): Fraction | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null;
  if (value < -FRACTION_TOLERANCE || value > 1 + FRACTION_TOLERANCE) return null;
  return toFraction(value);
}

/** isRecord reports whether a value is a plain object. */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
