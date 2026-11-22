import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import {
  OFFLINE_RETRY_SECONDS,
  offlineLanguage,
  offlinePage,
  type OfflineColors,
  type OfflineTexts,
} from '@/lib/pwa/offline';

/** Returns a random language code of two letters. */
function fakeLanguage(): string {
  return faker.string.alpha({
    length: 2,
    casing: 'lower',
  });
}

/** Returns random texts of the offline page. */
function fakeTexts(): OfflineTexts {
  return {
    title: faker.lorem.words(),
    message: faker.lorem.sentence(),
    retry: faker.lorem.word(),
  };
}

/** Returns random colors of one color mode. */
function fakeColors(): OfflineColors {
  return {
    background: faker.color.rgb(),
    text: faker.color.rgb(),
    muted: faker.color.rgb(),
    primary: faker.color.rgb(),
    onPrimary: faker.color.rgb(),
  };
}

describe('offlineLanguage', () => {
  it('returns the first preferred language of the browser', () => {
    const first = fakeLanguage();
    const second = fakeLanguage();

    const language = offlineLanguage([second, first], [first, second], fakeLanguage());

    expect(language).toBe(first);
  });

  it('matches a preferred language by its code without the region', () => {
    const preferred = fakeLanguage();

    const language = offlineLanguage(
      [preferred],
      [`${preferred}-${faker.location.countryCode()}`],
      fakeLanguage(),
    );

    expect(language).toBe(preferred);
  });

  it('skips a preferred language without texts', () => {
    const available = fakeLanguage();

    const language = offlineLanguage([available], [fakeLanguage(), available], fakeLanguage());

    expect(language).toBe(available);
  });

  it('returns the fallback without a language that has texts', () => {
    const fallback = fakeLanguage();

    const language = offlineLanguage([fallback], [], fallback);

    expect(language).toBe(fallback);
  });
});

describe('offlinePage', () => {
  it('shows the texts in the language', () => {
    const language = fakeLanguage();
    const texts = fakeTexts();

    const page = offlinePage(language, texts, {
      light: fakeColors(),
      dark: fakeColors(),
    });

    expect(page).toContain(`<html lang="${language}">`);
    expect(page).toContain(`<h1>${texts.title}</h1>`);
    expect(page).toContain(texts.message);
    expect(page).toContain(`${texts.retry}</a>`);
  });

  it('colors the page for the light and the dark mode', () => {
    const light = fakeColors();
    const dark = fakeColors();

    const page = offlinePage(fakeLanguage(), fakeTexts(), {
      light,
      dark,
    });

    const darkStart = page.indexOf('prefers-color-scheme: dark');
    expect(page.slice(0, darkStart)).toContain(`--primary: ${light.primary};`);
    expect(page.slice(darkStart)).toContain(`--primary: ${dark.primary};`);
  });

  it('loads the app again on its own', () => {
    const page = offlinePage(fakeLanguage(), fakeTexts(), {
      light: fakeColors(),
      dark: fakeColors(),
    });

    expect(page).toContain(`<meta http-equiv="refresh" content="${OFFLINE_RETRY_SECONDS}">`);
  });

  it('escapes markup in the texts', () => {
    const page = offlinePage(
      fakeLanguage(),
      {
        title: '<b>&</b>',
        message: faker.lorem.sentence(),
        retry: '"',
      },
      {
        light: fakeColors(),
        dark: fakeColors(),
      },
    );

    expect(page).toContain('<h1>&lt;b&gt;&amp;&lt;/b&gt;</h1>');
    expect(page).toContain('&quot;</a>');
  });
});
