import { describe, expect, it } from 'vitest';

import { CornerKey, type LiveHeat, type PresetResponse } from '@/api/types';
import {
  DEFAULT_FIRST_LANE,
  DEFAULT_FRAME_HEIGHT,
  DEFAULT_FRAME_WIDTH,
  DEFAULT_LANE_GAP,
  DEFAULT_TEXT_SIZE,
  LANE_CONTENT_HEIGHT,
  LANE_CONTENT_WIDTH,
  alignLanes,
  clampCoordinate,
  createPreset,
  heatLaneNumber,
  isLaneValid,
  laneBounds,
  laneBoundsExcluding,
  laneCenter,
  lanePointTargets,
  laneStyle,
  liveFirstLane,
  parsePreset,
  pixelPoint,
  presetLaneCount,
  scaleLanes,
  toEditorPreset,
  toFraction,
  toPresetRequest,
  type EditorPreset,
} from '@/lib/lanes';

const FRAME = {
  width: DEFAULT_FRAME_WIDTH,
  height: DEFAULT_FRAME_HEIGHT,
};

/** editorPreset opens a new preset in pixels of the default reference resolution. */
function editorPreset(laneCount: number): EditorPreset {
  return toEditorPreset(createPreset('test', laneCount));
}

describe('toFraction', () => {
  it('rounds a share to six decimal places', () => {
    const fraction = toFraction(0.1234567);

    expect(fraction).toBe(0.123457);
  });

  it.each([
    {
      value: -0.2,
      expected: 0,
    },
    {
      value: 1.3,
      expected: 1,
    },
  ])('keeps the share $value within 0 and 1', ({ value, expected }) => {
    const fraction = toFraction(value);

    expect(fraction).toBe(expected);
  });
});

describe('toEditorPreset', () => {
  it('converts the fractions into pixels of the reference resolution', () => {
    const stored = createPreset('test', 2);

    const preset = toEditorPreset(stored);

    expect(preset.lanes['1']?.[CornerKey.TopLeft]).toEqual({
      x: 480,
      y: 160,
    });
    expect(preset.lanes['2']?.[CornerKey.BottomRight]).toEqual({
      x: 1440,
      y: 270,
    });
  });

  it('round-trips through toPresetRequest', () => {
    const stored = createPreset('test', 3);

    const preset = toEditorPreset(stored);

    expect(toPresetRequest(preset)).toEqual(stored);
  });
});

describe('toPresetRequest', () => {
  it('stores the corners as fractions rounded to six decimal places', () => {
    const preset = editorPreset(1);
    preset.lanes['1']![CornerKey.TopLeft] = pixelPoint(100, 1000);

    const request = toPresetRequest(preset);

    expect(request.lanes['1']?.[CornerKey.TopLeft]).toEqual({
      x: 0.052083,
      y: 0.925926,
    });
  });
});

describe('scaleLanes', () => {
  it('keeps the relative position of the lanes for another resolution', () => {
    const preset = editorPreset(1);

    scaleLanes(preset.lanes, FRAME, {
      width: 1280,
      height: 720,
    });

    expect(preset.lanes['1']?.[CornerKey.TopLeft]).toEqual({
      x: 320,
      y: 106.67,
    });
    expect(preset.lanes['1']?.[CornerKey.BottomRight]).toEqual({
      x: 960,
      y: 140,
    });
  });
});

describe('createPreset', () => {
  it('uses the default reference resolution, text size, and lane gap', () => {
    const preset = createPreset('test', 2);

    expect(preset.frameWidth).toBe(DEFAULT_FRAME_WIDTH);
    expect(preset.frameHeight).toBe(DEFAULT_FRAME_HEIGHT);
    expect(preset.textSize).toBe(DEFAULT_TEXT_SIZE);
    expect(preset.laneGap).toBe(DEFAULT_LANE_GAP);
  });

  it('stacks the lanes in the middle half of the image as fractions', () => {
    const preset = createPreset('test', 2);

    expect(Object.keys(preset.lanes)).toEqual(['1', '2']);
    expect(preset.lanes['1']?.[CornerKey.TopLeft]).toEqual({
      x: 0.25,
      y: 0.148148,
    });
    expect(preset.lanes['2']?.[CornerKey.BottomRight]).toEqual({
      x: 0.75,
      y: 0.25,
    });
  });
});

describe('presetLaneCount', () => {
  it('returns the highest lane key', () => {
    const lanes = {
      '1': {},
      '2': {},
      '10': {},
    };

    const count = presetLaneCount(lanes);

    expect(count).toBe(10);
  });

  it('returns zero without lanes', () => {
    const count = presetLaneCount({});

    expect(count).toBe(0);
  });
});

describe('heatLaneNumber', () => {
  it.each([
    {
      presetLane: 1,
      firstLane: 1,
      reverse: false,
      expected: 1,
    },
    {
      presetLane: 8,
      firstLane: 1,
      reverse: false,
      expected: 8,
    },
    {
      presetLane: 1,
      firstLane: 0,
      reverse: false,
      expected: 0,
    },
    {
      presetLane: 8,
      firstLane: 0,
      reverse: false,
      expected: 7,
    },
    {
      presetLane: 1,
      firstLane: 1,
      reverse: true,
      expected: 8,
    },
    {
      presetLane: 8,
      firstLane: 1,
      reverse: true,
      expected: 1,
    },
    {
      presetLane: 1,
      firstLane: 0,
      reverse: true,
      expected: 7,
    },
    {
      presetLane: 8,
      firstLane: 0,
      reverse: true,
      expected: 0,
    },
  ])(
    'maps preset lane $presetLane to its heat lane from first lane $firstLane, reversed: $reverse',
    ({ presetLane, firstLane, reverse, expected }) => {
      const lane = heatLaneNumber(presetLane, 8, firstLane, reverse);

      expect(lane).toBe(expected);
    },
  );
});

describe('liveFirstLane', () => {
  const heat: LiveHeat = {
    event: 1,
    phase: 0,
    phaseName: '',
    heat: 1,
    heatCount: 1,
    lapCount: 2,
    name: '',
    status: 'load',
    firstLane: 0,
    lanes: {},
  };

  it('returns the first lane of a loaded heat', () => {
    const firstLane = liveFirstLane(heat);

    expect(firstLane).toBe(0);
  });

  it.each<{ name: string; state: LiveHeat | null }>([
    {
      name: 'without a live heat',
      state: null,
    },
    {
      name: 'before the first frame',
      state: {
        ...heat,
        status: '',
      },
    },
  ])('falls back to the default $name', ({ state }) => {
    const firstLane = liveFirstLane(state);

    expect(firstLane).toBe(DEFAULT_FIRST_LANE);
  });
});

describe('clampCoordinate', () => {
  it('rounds a coordinate to whole pixels', () => {
    const coordinate = clampCoordinate(12.6, 1920);

    expect(coordinate).toBe(13);
  });

  it.each([
    {
      value: -4,
      expected: 0,
    },
    {
      value: 2000,
      expected: 1920,
    },
  ])('keeps the coordinate $value within the frame', ({ value, expected }) => {
    const coordinate = clampCoordinate(value, 1920);

    expect(coordinate).toBe(expected);
  });
});

describe('isLaneValid', () => {
  it('accepts a lane that can be projected', () => {
    const lane = editorPreset(1).lanes['1']!;

    const valid = isLaneValid(lane);

    expect(valid).toBe(true);
  });

  it('rejects a lane that cannot be projected', () => {
    const lane = editorPreset(1).lanes['1']!;
    lane[CornerKey.TopLeft] = pixelPoint(1500, 200);

    const valid = isLaneValid(lane);

    expect(valid).toBe(false);
  });
});

describe('laneCenter', () => {
  it('averages the corners', () => {
    const lane = editorPreset(1).lanes['1']!;

    const center = laneCenter(lane);

    expect(center).toEqual({
      x: 960,
      y: 185,
    });
  });
});

describe('laneStyle', () => {
  it('projects the content box onto the lane', () => {
    const lane = editorPreset(1).lanes['1']!;

    const style = laneStyle(lane);

    expect(style.width).toBe(`${LANE_CONTENT_WIDTH}px`);
    expect(style.height).toBe(`${LANE_CONTENT_HEIGHT}px`);
    expect(style.transformOrigin).toBe('0 0');
    expect(style.transform).toMatch(/^matrix3d\(/);
  });

  it('hides a lane that cannot be projected', () => {
    const lane = editorPreset(1).lanes['1']!;
    lane[CornerKey.TopLeft] = pixelPoint(1500, 200);

    const style = laneStyle(lane);

    expect(style).toEqual({
      display: 'none',
    });
  });
});

describe('laneBounds', () => {
  it('spans every corner of the lane', () => {
    const lane = editorPreset(1).lanes['1']!;
    lane[CornerKey.BottomLeft] = pixelPoint(400, 300);

    const bounds = laneBounds(lane);

    expect(bounds).toEqual({
      minX: 400,
      maxX: 1440,
      minY: 160,
      maxY: 300,
    });
  });
});

describe('lanePointTargets', () => {
  it('collects the corners and edges of the other lanes and the image border', () => {
    const lanes = editorPreset(2).lanes;

    const targets = lanePointTargets(lanes, new Set(['2']), FRAME);

    expect(targets.corners).toHaveLength(4);
    expect(targets.edges).toHaveLength(8);
    expect(targets.corners).toContainEqual(lanes['1']![CornerKey.BottomRight]);
  });
});

describe('laneBoundsExcluding', () => {
  it('returns the bounds of the other lanes', () => {
    const lanes = editorPreset(2).lanes;

    const bounds = laneBoundsExcluding(lanes, new Set(['2']));

    expect(bounds).toEqual([laneBounds(lanes['1']!)]);
  });
});

describe('alignLanes', () => {
  /** squarePool places lane 1 and the last lane so they span a square outline. */
  function squarePool(laneCount: number) {
    const preset = editorPreset(laneCount);
    const first = preset.lanes['1']!;
    const last = preset.lanes[String(laneCount)]!;
    first[CornerKey.TopLeft] = pixelPoint(100, 100);
    first[CornerKey.TopRight] = pixelPoint(1100, 100);
    last[CornerKey.BottomRight] = pixelPoint(1100, 1100);
    last[CornerKey.BottomLeft] = pixelPoint(100, 1100);
    return preset;
  }

  it('distributes rectangular lanes evenly with gaps of a share of the lane height', () => {
    const preset = squarePool(3);

    const aligned = alignLanes(
      preset.lanes,
      {
        width: 2000,
        height: 2000,
      },
      25,
    );

    expect(aligned).toBe(true);
    // 1 / (3 + 2 * 0.25) = 0.2857 per lane, a quarter of it per gap
    expect(preset.lanes['2']?.[CornerKey.TopLeft]).toEqual({
      x: 100,
      y: 457,
    });
    expect(preset.lanes['2']?.[CornerKey.BottomRight]).toEqual({
      x: 1100,
      y: 743,
    });
  });

  it('lets the lanes touch without a gap', () => {
    const preset = squarePool(3);

    const aligned = alignLanes(
      preset.lanes,
      {
        width: 2000,
        height: 2000,
      },
      0,
    );

    expect(aligned).toBe(true);
    expect(preset.lanes['1']?.[CornerKey.BottomLeft].y).toBe(433);
    expect(preset.lanes['2']?.[CornerKey.TopLeft].y).toBe(433);
    expect(preset.lanes['2']?.[CornerKey.BottomLeft].y).toBe(767);
    expect(preset.lanes['3']?.[CornerKey.TopLeft].y).toBe(767);
  });

  it('keeps the default gap small relative to the outline for a full set of lanes', () => {
    const preset = squarePool(8);

    const aligned = alignLanes(
      preset.lanes,
      {
        width: 2000,
        height: 2000,
      },
      DEFAULT_LANE_GAP,
    );

    expect(aligned).toBe(true);
    // A fixed gap of 0.02 of the outline leaves (1 - 0.14) / 8 = 0.1075 per lane
    expect(preset.lanes['2']?.[CornerKey.TopLeft].y).toBe(228);
    expect(preset.lanes['8']?.[CornerKey.TopLeft].y).toBeCloseTo(992.5, -0.5);
  });

  it('keeps the aligned lanes within the frame', () => {
    const preset = editorPreset(3);
    preset.lanes['3']![CornerKey.BottomRight] = pixelPoint(1920, 1080);
    preset.lanes['3']![CornerKey.BottomLeft] = pixelPoint(0, 1080);

    const aligned = alignLanes(
      preset.lanes,
      {
        width: 1000,
        height: 500,
      },
      DEFAULT_LANE_GAP,
    );

    expect(aligned).toBe(true);
    for (const lane of Object.values(preset.lanes)) {
      expect(lane[CornerKey.BottomRight].x).toBeLessThanOrEqual(1920);
    }
    expect(preset.lanes['2']![CornerKey.BottomRight].y).toBeLessThanOrEqual(500);
  });

  it('keeps locked lanes in place', () => {
    const preset = editorPreset(3);
    const locked = structuredClone(preset.lanes['2']!);
    preset.lanes['2']!.locked = true;

    alignLanes(preset.lanes, FRAME, DEFAULT_LANE_GAP);

    expect(preset.lanes['2']?.[CornerKey.TopLeft]).toEqual(locked[CornerKey.TopLeft]);
  });

  it('rejects an outline that is not a valid quad', () => {
    const preset = editorPreset(3);
    const before = structuredClone(preset.lanes['2']!);
    preset.lanes['3']![CornerKey.BottomRight] = pixelPoint(100, 50);

    const aligned = alignLanes(preset.lanes, FRAME, DEFAULT_LANE_GAP);

    expect(aligned).toBe(false);
    expect(preset.lanes['2']).toEqual(before);
  });
});

describe('parsePreset', () => {
  const preset: PresetResponse = createPreset('preset', 1);
  const lane = preset.lanes['1']!;

  it('returns an exported preset', () => {
    const exported = createPreset('preset', 2);

    const parsed = parsePreset(JSON.stringify(exported));

    expect(parsed).toEqual(exported);
  });

  it('clamps fractions slightly outside of the image', () => {
    const json = JSON.stringify({
      ...preset,
      lanes: {
        '1': {
          ...lane,
          tl: {
            x: -0.0004,
            y: 1.0004,
          },
        },
      },
    });

    const parsed = parsePreset(json);

    expect(parsed?.lanes['1']?.[CornerKey.TopLeft]).toEqual({
      x: 0,
      y: 1,
    });
  });

  it.each([
    {
      name: 'content that is not JSON',
      json: 'not json',
    },
    {
      name: 'content that is not a preset',
      json: '{"name":"preset"}',
    },
    {
      name: 'a preset without lanes',
      json: JSON.stringify({
        ...preset,
        lanes: {},
      }),
    },
    {
      name: 'a lane key that is not a number',
      json: JSON.stringify({
        ...preset,
        lanes: {
          first: lane,
        },
      }),
    },
    {
      name: 'an empty frame',
      json: JSON.stringify({
        ...preset,
        frameWidth: 0,
      }),
    },
    {
      name: 'a missing text size',
      json: JSON.stringify({
        ...preset,
        textSize: undefined,
      }),
    },
    {
      name: 'a missing lane gap',
      json: JSON.stringify({
        ...preset,
        laneGap: undefined,
      }),
    },
    {
      name: 'a lane gap above the maximum',
      json: JSON.stringify({
        ...preset,
        laneGap: 101,
      }),
    },
    {
      name: 'a lane gap that is not a whole number',
      json: JSON.stringify({
        ...preset,
        laneGap: 2.5,
      }),
    },
    {
      name: 'a corner without both coordinates',
      json: JSON.stringify({
        ...preset,
        lanes: {
          '1': {
            ...lane,
            tl: {
              x: 1,
            },
          },
        },
      }),
    },
    {
      name: 'a preset with pixel coordinates',
      json: JSON.stringify({
        ...preset,
        lanes: {
          '1': {
            ...lane,
            tl: {
              x: 480,
              y: 160,
            },
          },
        },
      }),
    },
  ])('rejects $name', ({ json }) => {
    const parsed = parsePreset(json);

    expect(parsed).toBeNull();
  });
});
