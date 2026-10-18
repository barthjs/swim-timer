/** FrameSize is the width and height of an image in pixels. */
export type FrameSize = {
  width: number;
  height: number;
};

/** FrameAspect is a common aspect ratio of a browser source with its usual resolutions. */
export type FrameAspect = {
  key: '16:9' | '4:3' | '1:1';
  width: number;
  height: number;
  resolutions: readonly FrameSize[];
};

/**
 * The common aspect ratios in the order the editor offers them. 16:9 comes first, since it is the usual format
 * of a browser source and the default of a new preset.
 */
export const FRAME_ASPECTS: readonly FrameAspect[] = [
  {
    key: '16:9',
    width: 16,
    height: 9,
    resolutions: [
      {
        width: 1280,
        height: 720,
      },
      {
        width: 1920,
        height: 1080,
      },
      {
        width: 2560,
        height: 1440,
      },
      {
        width: 3840,
        height: 2160,
      },
    ],
  },
  {
    key: '4:3',
    width: 4,
    height: 3,
    resolutions: [
      {
        width: 640,
        height: 480,
      },
      {
        width: 1024,
        height: 768,
      },
      {
        width: 1440,
        height: 1080,
      },
      {
        width: 1600,
        height: 1200,
      },
    ],
  },
  {
    key: '1:1',
    width: 1,
    height: 1,
    resolutions: [
      {
        width: 720,
        height: 720,
      },
      {
        width: 1080,
        height: 1080,
      },
      {
        width: 2160,
        height: 2160,
      },
    ],
  },
];

/** findAspect returns the common aspect ratio a size has exactly, or null for any other size. */
export function findAspect(frame: FrameSize): FrameAspect | null {
  return (
    FRAME_ASPECTS.find((aspect) => frame.width * aspect.height === frame.height * aspect.width) ??
    null
  );
}

/**
 * closestResolution returns the resolution of an aspect ratio whose pixel count comes closest to the given size.
 */
export function closestResolution(aspect: FrameAspect, frame: FrameSize): FrameSize {
  const pixels = frame.width * frame.height;
  let best = aspect.resolutions[0]!;
  for (const resolution of aspect.resolutions) {
    const distance = Math.abs(resolution.width * resolution.height - pixels);
    if (distance < Math.abs(best.width * best.height - pixels)) best = resolution;
  }
  return best;
}
