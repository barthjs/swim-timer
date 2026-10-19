import { describe, expect, it } from 'vitest';

// Every locale file of the interface, keyed by its path
const locales = import.meta.glob<Record<string, unknown>>('./locales/*.json', {
  eager: true,
  import: 'default',
});

describe('locales', () => {
  it('has the texts of the default locale in every locale', () => {
    const english = Object.entries(locales).find(([path]) => path.endsWith('/en.json'))?.[1] ?? {};
    const expected = Object.fromEntries(
      Object.keys(locales).map((path) => [path, textKeys(english)]),
    );

    const actual = Object.fromEntries(
      Object.entries(locales).map(([path, messages]) => [path, textKeys(messages)]),
    );

    expect(Object.keys(actual).length).toBeGreaterThan(1);
    expect(actual).toEqual(expected);
  });
});

/** textKeys returns the dotted keys of every text of a locale, sorted. */
function textKeys(messages: Record<string, unknown>, prefix = ''): string[] {
  return Object.entries(messages)
    .flatMap(([key, value]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      if (typeof value === 'object' && value !== null) {
        return textKeys(value as Record<string, unknown>, path);
      }
      return [path];
    })
    .sort();
}
