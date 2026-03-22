import { createI18n } from 'vue-i18n';

const SUPPORTED_LOCALES = ['en', 'de'];
const FALLBACK_LOCALE = 'en';

function getBrowserLocale(): string {
  const navigatorLang: string | undefined = navigator.languages
    ? navigator.languages[0]
    : navigator.language;

  if (!navigatorLang) return FALLBACK_LOCALE;

  const shortLang = navigatorLang.split('-')[0];
  if (!shortLang) return FALLBACK_LOCALE;

  const lowerShortLang = shortLang.toLowerCase();
  return SUPPORTED_LOCALES.includes(lowerShortLang) ? lowerShortLang : FALLBACK_LOCALE;
}

const initialLocale = getBrowserLocale();

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: FALLBACK_LOCALE,
  messages: {},
});

const loadedLanguages: string[] = [];

function setI18nLanguage(locale: string) {
  i18n.global.locale.value = locale;
  document.querySelector('html')!.setAttribute('lang', locale);
}

export async function loadLocaleMessages(locale: string) {
  if (loadedLanguages.includes(locale)) {
    return setI18nLanguage(locale);
  }

  try {
    const messages = await import(`./locales/${locale}.json`);

    i18n.global.setLocaleMessage(locale, messages);
    loadedLanguages.push(locale);
    return setI18nLanguage(locale);
  } catch (error) {
    console.error(`Failed to load locale messages for ${locale}:`, error);
  }
}

await loadLocaleMessages(initialLocale);
