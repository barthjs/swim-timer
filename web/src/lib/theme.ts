const TINTS: Array<[shade: number, mixWhitePercent: number]> = [
  [50, 95],
  [100, 90],
  [200, 75],
  [300, 60],
  [400, 30],
];

const SHADES: Array<[shade: number, mixBlackPercent: number]> = [
  [600, 15],
  [700, 30],
  [800, 45],
  [900, 60],
  [950, 75],
];

const COLOR_KEYS = [
  'primary',
  'secondary',
  'success',
  'info',
  'warning',
  'error',
  'neutral',
] as const;

/** ThemeColors maps every semantic color of Nuxt UI to a hex value, empty for the built-in default. */
export type ThemeColors = Record<(typeof COLOR_KEYS)[number], string>;

/**
 * applyThemeColors sets the configured colors on the document. An empty color keeps the default.
 */
export function applyThemeColors(theme: ThemeColors): void {
  const root = document.documentElement.style;

  for (const name of COLOR_KEYS) {
    const hex = theme[name];
    if (!hex) continue;

    root.setProperty(`--ui-color-${name}-500`, hex);

    for (const [shade, percent] of TINTS) {
      root.setProperty(
        `--ui-color-${name}-${shade}`,
        `color-mix(in oklch, white ${percent}%, ${hex} ${100 - percent}%)`,
      );
    }

    for (const [shade, percent] of SHADES) {
      root.setProperty(
        `--ui-color-${name}-${shade}`,
        `color-mix(in oklch, black ${percent}%, ${hex} ${100 - percent}%)`,
      );
    }
  }
}
