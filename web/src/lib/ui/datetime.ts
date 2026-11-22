// The API returns every timestamp in UTC. These helpers convert it to and from the wall clock of an IANA time zone.
// Without a time zone the browser's zone applies, which suits guests.

/** Renders an ISO timestamp as a localized date with time, or a dash without a value. */
export function formatDateTime(
  value: string | null | undefined,
  locale: string,
  timeZone?: string,
): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone,
  }).format(new Date(value));
}

/** Renders an ISO timestamp as a localized date without time, or a dash without a value. */
export function formatDate(
  value: string | null | undefined,
  locale: string,
  timeZone?: string,
): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeZone,
  }).format(new Date(value));
}

/**
 * Renders an ISO timestamp as a localized date with time in digits only,
 * or a dash without a value. Every value has the same length, so a column of them lines up.
 */
export function formatDateTimeNumeric(
  value: string | null | undefined,
  locale: string,
  timeZone?: string,
): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    timeZone,
  }).format(new Date(value));
}

/**
 * Renders a calendar date in the form YYYY-MM-DD as a localized date in digits only,
 * without the year where the context names it. The value has no time, so there is no time zone to convert.
 */
export function formatPlainDate(value: string, locale: string, withYear = true): string {
  const [year = 0, month = 1, day = 1] = value.split('-').map(Number);
  return new Intl.DateTimeFormat(locale, {
    year: withYear ? 'numeric' : undefined,
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(year, month - 1, day));
}

/**
 * Returns how many minutes the time zone is ahead of UTC at the given instant.
 * It derives the offset from the wall clock Intl renders, which stays correct across DST transitions.
 */
function tzOffsetMinutes(date: Date, timeZone: string): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone,
    hourCycle: 'h23',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).formatToParts(date);

  /** Returns the number of a date part. */
  const lookup = (type: string) => Number(parts.find((p) => p.type === type)?.value);
  const asUTC = Date.UTC(
    lookup('year'),
    lookup('month') - 1,
    lookup('day'),
    lookup('hour'),
    lookup('minute'),
    lookup('second'),
  );
  return Math.round((asUTC - date.getTime()) / 60000);
}

/**
 * Converts a UTC ISO timestamp into the "YYYY-MM-DDTHH:mm" value of a datetime-local input
 * in the given time zone, or an empty string without a value.
 */
export function utcToLocalInput(value: string | null | undefined, timeZone: string): string {
  if (!value) return '';
  const date = new Date(value);
  const shifted = new Date(date.getTime() + tzOffsetMinutes(date, timeZone) * 60000);
  return shifted.toISOString().slice(0, 16);
}

/**
 * Converts a datetime-local value in the given time zone into a UTC ISO timestamp,
 * or null without a value. The offset is resolved twice, so a value near a DST transition maps to the correct instant.
 */
export function localInputToUtc(value: string | null | undefined, timeZone: string): string | null {
  if (!value) return null;
  const naive = new Date(`${value}:00Z`).getTime();
  const firstGuess = naive - tzOffsetMinutes(new Date(naive), timeZone) * 60000;
  const offset = tzOffsetMinutes(new Date(firstGuess), timeZone);
  return new Date(naive - offset * 60000).toISOString();
}

/** Returns the IANA time zones the browser knows, or none if it cannot list them. */
export function supportedTimeZones(): string[] {
  // Browsers released before 2022 lack Intl.supportedValuesOf
  if (typeof Intl.supportedValuesOf !== 'function') return [];
  return Intl.supportedValuesOf('timeZone');
}
