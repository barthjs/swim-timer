export const TextAnchor = {
  Left: 'left',
  Right: 'right',
} as const;
export type TextAnchor = (typeof TextAnchor)[keyof typeof TextAnchor];

export const CornerKey = {
  TopLeft: 'tl',
  TopRight: 'tr',
  BottomRight: 'br',
  BottomLeft: 'bl',
} as const;
export type CornerKey = (typeof CornerKey)[keyof typeof CornerKey];

declare const fractionBrand: unique symbol;

/**
 * A share of the width or height of the output image, from 0 to 1.
 * The brand keeps the stored fractions apart from pixel values, which are plain numbers.
 */
export type Fraction = number & { readonly [fractionBrand]: true };

/**
 * A position in the output image as fractions of its width and height,
 * from 0, 0 at the top left to 1, 1 at the bottom right.
 */
export type PresetPointRequest = {
  x: Fraction;
  y: Fraction;
};

export type PresetLaneRequest = Record<CornerKey, PresetPointRequest> & {
  locked: boolean;
};

/**
 * A position in the output image as fractions of its width and height,
 * from 0, 0 at the top left to 1, 1 at the bottom right.
 */
export type PresetPointResponse = {
  x: Fraction;
  y: Fraction;
};

export type PresetLaneResponse = Record<CornerKey, PresetPointResponse> & {
  locked: boolean;
};

/** A preset to create or replace. The name has to match the name in the path. */
export type PresetSaveRequest = {
  name: string;
  description: string | null;
  frameWidth: number;
  frameHeight: number;
  textSize: number;
  laneGap: number;
  reverseLanes: boolean;
  textAnchor: TextAnchor;
  lanes: Record<string, PresetLaneRequest>;
};

export type PresetResponse = {
  name: string;
  description: string | null;
  // The reference resolution of the output image. It sets the aspect ratio the editor draws the preset in
  // and the pixels it shows, while the lanes are stored as fractions.
  frameWidth: number;
  frameHeight: number;
  // The height of the text in percent of the height of a lane
  textSize: number;
  // The gap between two neighboring lanes in percent of the height of a lane, which aligning keeps
  laneGap: number;
  reverseLanes: boolean;
  textAnchor: TextAnchor;
  lanes: Record<string, PresetLaneResponse>;
};
