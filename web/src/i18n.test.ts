import { afterEach, describe, expect, it } from 'vitest';

import { i18n, setLocaleFromBackend } from '@/i18n';

describe('setLocaleFromBackend', () => {
  afterEach(() => {
    i18n.global.locale.value = 'en';
  });

  it('switches to a supported language', () => {
    setLocaleFromBackend('de');

    expect(i18n.global.locale.value).toBe('de');
  });

  it('keeps the locale for an unsupported language', () => {
    setLocaleFromBackend('fr');

    expect(i18n.global.locale.value).toBe('en');
  });
});
