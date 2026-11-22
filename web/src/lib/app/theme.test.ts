import { afterEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { applyThemeColors, type ThemeColors } from '@/lib/app/theme';

const emptyTheme: ThemeColors = {
  primary: '',
  secondary: '',
  success: '',
  info: '',
  warning: '',
  error: '',
  neutral: '',
};

/** Replaces the document with one that records the custom properties set on its root. */
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
    const color = faker.color.rgb();
    const properties = stubRootStyle();

    applyThemeColors({
      ...emptyTheme,
      primary: color,
    });

    expect(properties.get('--ui-color-primary-500')).toBe(color);
    expect(properties.get('--ui-color-primary-50')).toBe(
      `color-mix(in oklch, white 95%, ${color} 5%)`,
    );
    expect(properties.get('--ui-color-primary-950')).toBe(
      `color-mix(in oklch, black 75%, ${color} 25%)`,
    );
    expect(properties.size).toBe(11);
  });

  it('keeps the built-in defaults of unset colors', () => {
    const properties = stubRootStyle();

    applyThemeColors(emptyTheme);

    expect(properties.size).toBe(0);
  });
});
