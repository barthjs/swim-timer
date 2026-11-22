import { createI18n } from 'vue-i18n';

import en from '@/locales/en.json';
import de from '@/locales/de.json';

export const supportedLocales = ['en', 'de'] as const;
export type SupportedLocale = (typeof supportedLocales)[number];

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: {
    en,
    de,
  },
});

/** Reports whether the app has translations for a language. */
function isSupportedLocale(value: string): value is SupportedLocale {
  return (supportedLocales as readonly string[]).includes(value);
}

/**
 * Switches the active locale and the language of the document
 * to the backend-configured app-wide language.
 */
export function setLocaleFromBackend(language: string): void {
  if (isSupportedLocale(language)) {
    i18n.global.locale.value = language;
    document.documentElement.lang = language;
  }
}
