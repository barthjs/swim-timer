import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  formatDate,
  formatDateTime,
  formatDateTimeNumeric,
  formatPlainDate,
  localInputToUtc,
  utcToLocalInput,
} from '@/lib/ui/datetime';

/** Describe where the dates of the app are shown. */
export type DatesOptions = {
  // The time zone to show dates in, read on every use so it follows the session, undefined for the browser zone
  timeZone: () => string | undefined;
};

// Until the app configures a zone, dates follow the browser
const options: DatesOptions = {
  timeZone: () => undefined,
};

/** Sets where the dates of the app are shown. The app calls it once at startup. */
export function configureDates(changes: Partial<DatesOptions>): void {
  Object.assign(options, changes);
}

/** Returns date helpers bound to the active locale and the configured time zone. */
export function useDates() {
  const { locale } = useI18n();

  // An undefined zone lets Intl use the browser zone
  const displayZone = computed(() => options.timeZone() || undefined);

  // Conversions need an explicit zone, so resolve the browser zone without one
  const inputZone = computed(
    () => options.timeZone() || Intl.DateTimeFormat().resolvedOptions().timeZone,
  );

  return {
    formatDateTime: (value: string | null | undefined) =>
      formatDateTime(value, locale.value, displayZone.value),
    formatDate: (value: string | null | undefined) =>
      formatDate(value, locale.value, displayZone.value),
    formatDateTimeNumeric: (value: string | null | undefined) =>
      formatDateTimeNumeric(value, locale.value, displayZone.value),
    formatPlainDate: (value: string) => formatPlainDate(value, locale.value),
    utcToLocalInput: (value: string | null | undefined) => utcToLocalInput(value, inputZone.value),
    localInputToUtc: (value: string | null | undefined) => localInputToUtc(value, inputZone.value),
  };
}
