import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  formatDate,
  formatDateTime,
  formatDateTimeNumeric,
  formatPlainDate,
  localInputToUtc,
  supportedTimeZones,
  utcToLocalInput,
} from '@/lib/ui/datetime';

describe('formatDateTime', () => {
  it('includes the year and the time', () => {
    const value = '2026-03-01T12:00:00Z';

    const rendered = formatDateTime(value, 'en', 'UTC');

    expect(rendered).toContain('2026');
    expect(rendered).toMatch(/12:00|12.00/);
  });

  it.each([null, undefined])('renders a dash for %s', (value) => {
    const rendered = formatDateTime(value, 'en', 'UTC');

    expect(rendered).toBe('—');
  });
});

describe('formatDate', () => {
  // 23:30 UTC on the 15th is already the 16th in UTC+2
  it.each([
    {
      timeZone: 'UTC',
      day: '15',
    },
    {
      timeZone: 'Europe/Berlin',
      day: '16',
    },
  ])('renders the day in $timeZone, not the day of the raw UTC instant', ({ timeZone, day }) => {
    const value = '2026-06-15T23:30:00Z';

    const rendered = formatDate(value, 'en', timeZone);

    expect(rendered).toContain(day);
  });

  it('renders a dash for a missing value', () => {
    const rendered = formatDate(null, 'en', 'UTC');

    expect(rendered).toBe('—');
  });
});

describe('formatDateTimeNumeric', () => {
  it('renders every timestamp with the same length in digits', () => {
    const lateValue = '2026-12-24T23:59:00Z';
    const late = formatDateTimeNumeric(lateValue, 'de', 'UTC');

    const early = formatDateTimeNumeric('2026-01-02T03:04:00Z', 'de', 'UTC');

    expect(early).toBe('02.01.2026, 03:04');
    expect(early).toHaveLength(late.length);
  });

  it('renders the time in the given time zone', () => {
    const value = '2026-06-15T23:30:00Z';

    const rendered = formatDateTimeNumeric(value, 'de', 'Europe/Berlin');

    expect(rendered).toBe('16.06.2026, 01:30');
  });

  it('renders a dash for a missing value', () => {
    const rendered = formatDateTimeNumeric(null, 'de', 'UTC');

    expect(rendered).toBe('—');
  });
});

describe('formatPlainDate', () => {
  it('renders a calendar date in digits without shifting it to another day', () => {
    const value = '2026-03-01';

    const rendered = formatPlainDate(value, 'de');

    expect(rendered).toBe('01.03.2026');
  });

  it('leaves out the year on request', () => {
    const rendered = formatPlainDate('2026-03-01', 'de', false);

    expect(rendered).toBe('01.03.');
  });
});

/** Europe/Berlin is UTC+1 in January and UTC+2 in July, America/New_York is UTC-5 in January. */
const conversions = [
  {
    name: 'in winter without DST',
    timeZone: 'Europe/Berlin',
    utc: '2026-01-15T10:00:00.000Z',
    local: '2026-01-15T11:00',
  },
  {
    name: 'in summer with DST',
    timeZone: 'Europe/Berlin',
    utc: '2026-07-15T10:00:00.000Z',
    local: '2026-07-15T12:00',
  },
  {
    name: 'in a time zone west of UTC',
    timeZone: 'America/New_York',
    utc: '2026-01-15T10:00:00.000Z',
    local: '2026-01-15T05:00',
  },
];

describe('utcToLocalInput', () => {
  it.each(conversions)(
    'converts a UTC instant to the wall-clock value $name',
    ({ timeZone, utc, local }) => {
      const value = utc.replace('.000Z', 'Z');

      const input = utcToLocalInput(value, timeZone);

      expect(input).toBe(local);
    },
  );

  it.each([null, undefined])('returns an empty value for %s', (value) => {
    const input = utcToLocalInput(value, 'UTC');

    expect(input).toBe('');
  });
});

describe('localInputToUtc', () => {
  it.each(conversions)(
    'converts a wall-clock value back to its UTC instant $name',
    ({ timeZone, utc, local }) => {
      const instant = localInputToUtc(local, timeZone);

      expect(instant).toBe(utc);
    },
  );

  // Europe/Berlin moves from UTC+1 to UTC+2 at 2026-03-29 01:00 UTC and back at 2026-10-25 01:00 UTC.
  // The points of the fall-back avoid the repeated local hour, whose wall-clock values are ambiguous.
  it.each([
    {
      name: 'before the spring-forward',
      utc: '2026-03-29T00:30:00Z',
    },
    {
      name: 'after the spring-forward',
      utc: '2026-03-29T01:30:00Z',
    },
    {
      name: 'before the fall-back',
      utc: '2026-10-24T20:00:00Z',
    },
    {
      name: 'after the fall-back',
      utc: '2026-10-25T04:00:00Z',
    },
  ])('round-trips an instant $name', ({ utc }) => {
    const input = utcToLocalInput(utc, 'Europe/Berlin');

    const instant = localInputToUtc(input, 'Europe/Berlin');

    expect(instant).toBe(new Date(utc).toISOString());
  });

  it.each([null, ''])('returns null for %j', (value) => {
    const instant = localInputToUtc(value, 'UTC');

    expect(instant).toBeNull();
  });
});

describe('supportedTimeZones', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('returns the IANA time zones of the browser', () => {
    const zones = supportedTimeZones();

    expect(zones).toContain('Europe/Berlin');
  });

  it('returns no time zone if the browser cannot list them', () => {
    vi.stubGlobal('Intl', {
      supportedValuesOf: undefined,
    });

    const zones = supportedTimeZones();

    expect(zones).toEqual([]);
  });
});
