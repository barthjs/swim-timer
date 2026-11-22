import { ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { configureDates, useDates } from '@/composables/ui/useDates';
import {
  formatDate,
  formatDateTime,
  formatDateTimeNumeric,
  formatPlainDate,
  localInputToUtc,
  utcToLocalInput,
} from '@/lib/ui/datetime';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: ref('de'),
  }),
}));

const instant = '2026-05-01T12:30:00Z';

describe('configureDates', () => {
  it('reads the time zone on every use', () => {
    const zone = ref('America/New_York');
    configureDates({
      timeZone: () => zone.value,
    });
    const { formatDateTime: format } = useDates();
    zone.value = 'Asia/Tokyo';

    const rendered = format(instant);

    expect(rendered).toBe(formatDateTime(instant, 'de', 'Asia/Tokyo'));
  });
});

describe('useDates', () => {
  beforeEach(() => {
    useZone(undefined);
  });

  describe('formatDateTime', () => {
    it('formats in the configured time zone and the active locale', () => {
      useZone('America/New_York');
      const { formatDateTime: format } = useDates();

      const rendered = format(instant);

      expect(rendered).toBe(formatDateTime(instant, 'de', 'America/New_York'));
    });

    it('falls back to the time zone of the browser without a configured zone', () => {
      useZone(undefined);
      const { formatDateTime: format } = useDates();

      const rendered = format(instant);

      expect(rendered).toBe(formatDateTime(instant, 'de', undefined));
    });
  });

  describe('formatDate', () => {
    it('formats in the configured time zone and the active locale', () => {
      useZone('America/New_York');
      const { formatDate: format } = useDates();

      const rendered = format(instant);

      expect(rendered).toBe(formatDate(instant, 'de', 'America/New_York'));
    });
  });

  describe('formatDateTimeNumeric', () => {
    it('formats in the configured time zone and the active locale', () => {
      useZone('America/New_York');
      const { formatDateTimeNumeric: format } = useDates();

      const rendered = format(instant);

      expect(rendered).toBe(formatDateTimeNumeric(instant, 'de', 'America/New_York'));
    });
  });

  describe('formatPlainDate', () => {
    it('formats in the active locale', () => {
      useZone('America/New_York');
      const { formatPlainDate: format } = useDates();

      const rendered = format('2026-05-01');

      expect(rendered).toBe(formatPlainDate('2026-05-01', 'de'));
    });
  });

  describe('utcToLocalInput', () => {
    it('converts in the configured time zone', () => {
      useZone('Asia/Tokyo');
      const { utcToLocalInput: convert } = useDates();

      const local = convert(instant);

      expect(local).toBe(utcToLocalInput(instant, 'Asia/Tokyo'));
    });

    it('falls back to the time zone of the browser without a configured zone', () => {
      useZone(undefined);
      const browserZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const { utcToLocalInput: convert } = useDates();

      const local = convert(instant);

      expect(local).toBe(utcToLocalInput(instant, browserZone));
    });
  });

  describe('localInputToUtc', () => {
    it('converts in the configured time zone', () => {
      useZone('Asia/Tokyo');
      const local = utcToLocalInput(instant, 'Asia/Tokyo');
      const { localInputToUtc: convert } = useDates();

      const utc = convert(local);

      expect(utc).toBe(localInputToUtc(local, 'Asia/Tokyo'));
    });
  });
});

/** Configures the time zone that the dates are shown in. */
function useZone(timeZone: string | undefined): void {
  configureDates({
    timeZone: () => timeZone,
  });
}
