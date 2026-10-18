import { afterEach, describe, expect, it, vi } from 'vitest';

import { applyThemeColors, type ThemeColors } from '@/lib/theme';

const emptyTheme: ThemeColors = {
  primary: '',
  secondary: '',
  success: '',
  info: '',
  warning: '',
  error: '',
  neutral: '',
};

/** stubRootStyle replaces the document with one that records the custom properties set on its root. */
function stubRootStyle(): Map<string, string> {
  const properties = new Map<string, string>();
  vi.stubGlobal('document', {
    documentElement: {
      style: {
        setProperty: (name: string, value: string) => properties.set(name, value),
      },
    },
  });
  return properties;
}

describe('applyThemeColors', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('derives the full shade scale from a configured color', () => {
    const properties = stubRootStyle();

    applyThemeColors({
      ...emptyTheme,
      primary: '#336699',
    });

    expect(properties.get('--ui-color-primary-500')).toBe('#336699');
    expect(properties.get('--ui-color-primary-50')).toBe(
      'color-mix(in oklch, white 95%, #336699 5%)',
    );
    expect(properties.get('--ui-color-primary-950')).toBe(
      'color-mix(in oklch, black 75%, #336699 25%)',
    );
    expect(properties.size).toBe(11);
  });

  it('keeps the built-in defaults of unset colors', () => {
    const properties = stubRootStyle();

    applyThemeColors(emptyTheme);

    expect(properties.size).toBe(0);
  });
});
