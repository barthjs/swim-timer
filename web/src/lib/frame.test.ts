import { describe, expect, it } from 'vitest';

import { FRAME_ASPECTS, closestResolution, findAspect } from '@/lib/frame';

describe('FRAME_ASPECTS', () => {
  it('offers the landscape formats with 16:9 first and no portrait format', () => {
    const keys = FRAME_ASPECTS.map((aspect) => aspect.key);

    expect(keys).toEqual(['16:9', '4:3', '1:1']);
  });
});

describe('findAspect', () => {
  it.each([
    {
      width: 1920,
      height: 1080,
      expected: '16:9',
    },
    {
      width: 1280,
      height: 720,
      expected: '16:9',
    },
    {
      width: 1024,
      height: 768,
      expected: '4:3',
    },
    {
      width: 500,
      height: 500,
      expected: '1:1',
    },
  ])('returns the aspect ratio $expected of $width x $height', ({ width, height, expected }) => {
    const aspect = findAspect({
      width,
      height,
    });

    expect(aspect?.key).toBe(expected);
  });

  it.each(
    FRAME_ASPECTS.flatMap((aspect) =>
      aspect.resolutions.map((resolution) => ({
        aspect,
        resolution,
      })),
    ),
  )(
    'returns the own aspect ratio of every listed resolution, such as $resolution.width x $resolution.height',
    ({ aspect, resolution }) => {
      const found = findAspect(resolution);

      expect(found).toBe(aspect);
    },
  );

  it.each([
    {
      width: 1920,
      height: 1200,
    },
    {
      width: 1366,
      height: 768,
    },
    {
      width: 1080,
      height: 1920,
    },
  ])('returns null for the other aspect ratio of $width x $height', ({ width, height }) => {
    const aspect = findAspect({
      width,
      height,
    });

    expect(aspect).toBeNull();
  });
});

describe('closestResolution', () => {
  it.each([
    {
      key: '16:9',
      frame: {
        width: 720,
        height: 1280,
      },
      expected: {
        width: 1280,
        height: 720,
      },
    },
    {
      key: '4:3',
      frame: {
        width: 1920,
        height: 1080,
      },
      expected: {
        width: 1600,
        height: 1200,
      },
    },
    {
      key: '1:1',
      frame: {
        width: 3840,
        height: 2160,
      },
      expected: {
        width: 2160,
        height: 2160,
      },
    },
  ])('picks the resolution of $key with the closest pixel count', ({ key, frame, expected }) => {
    const aspect = FRAME_ASPECTS.find((item) => item.key === key)!;

    const resolution = closestResolution(aspect, frame);

    expect(resolution).toEqual(expected);
  });
});
