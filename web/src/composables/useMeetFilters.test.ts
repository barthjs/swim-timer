import { nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { EventKind } from '@/lib/meet';
import { fakeRouter } from '@/test/router';
import { useMeetFilters } from '@/composables/useMeetFilters';

let fake = fakeRouter();

vi.mock('vue-router', () => ({
  useRoute: () => fake.route,
  useRouter: () => fake.router,
}));

describe('useMeetFilters', () => {
  beforeEach(() => {
    fake = fakeRouter();
  });

  describe('filters', () => {
    it('reads the filters from the URL', () => {
      fake = fakeRouter({
        q: 'search',
        days: ['2026-05-01', ''],
        styles: 'Breaststroke',
        kind: EventKind.Relay,
        upcoming: '1',
      });

      const { filters } = useMeetFilters();

      expect(filters.query).toBe('search');
      expect(filters.days).toEqual(['2026-05-01']);
      expect(filters.styles).toEqual(['Breaststroke']);
      expect(filters.kind).toBe(EventKind.Relay);
      expect(filters.upcomingOnly).toBe(true);
    });

    it('writes changed filters back and keeps other keys', async () => {
      fake = fakeRouter({
        tab: 'starts',
      });
      const { filters } = useMeetFilters();

      filters.query = 'search';
      filters.clubs = ['Club A'];
      filters.kind = EventKind.Individual;
      filters.upcomingOnly = true;
      await nextTick();

      expect(fake.route.query).toEqual({
        tab: 'starts',
        q: 'search',
        clubs: ['Club A'],
        kind: EventKind.Individual,
        upcoming: '1',
      });
    });

    it('ignores unknown values', () => {
      fake = fakeRouter({
        kind: 'everything',
        upcoming: 'yes',
        q: ['a', 'b'],
      });

      const { filters } = useMeetFilters();

      expect(filters.kind).toBe(EventKind.All);
      expect(filters.upcomingOnly).toBe(false);
      expect(filters.query).toBe('');
    });
  });

  describe('reset', () => {
    it('resets every filter and keeps other keys', async () => {
      fake = fakeRouter({
        q: 'search',
        styles: 'Breaststroke',
        page: '2',
      });
      const { filters, reset } = useMeetFilters();

      reset();
      await nextTick();

      expect(filters.query).toBe('');
      expect(filters.styles).toEqual([]);
      expect(fake.route.query).toEqual({
        page: '2',
      });
    });
  });
});
