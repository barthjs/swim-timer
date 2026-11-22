import { reactive, watch } from 'vue';
import { useRoute, useRouter, type LocationQuery, type LocationQueryRaw } from 'vue-router';

import { EventKind, emptyFilters, type MeetFilters } from '@/lib/meet/meet';
import { replaceRouteQuery } from '@/lib/ui/routeQuery';

/** The query parameters of the filters that hold several values. */
const LIST_KEYS = ['days', 'styles', 'distances', 'categories', 'clubs'] as const;

/** The query parameters of all filters, which every section of the meet overview reads. */
export const MEET_FILTER_KEYS: readonly string[] = ['q', 'kind', 'upcoming', ...LIST_KEYS];

/** Reads a query parameter that may occur several times. */
function readList(query: LocationQuery, key: string): string[] {
  const value = query[key];
  const values = Array.isArray(value) ? value : [value];
  return values.filter((item): item is string => typeof item === 'string' && item !== '');
}

/** Reads the filters from the URL. */
function readFilters(query: LocationQuery): MeetFilters {
  const filters = emptyFilters();
  filters.query = typeof query.q === 'string' ? query.q : '';
  for (const key of LIST_KEYS) {
    filters[key] = readList(query, key);
  }
  const kind = query.kind;
  if (kind === EventKind.Individual || kind === EventKind.Relay) {
    filters.kind = kind;
  }
  filters.upcomingOnly = query.upcoming === '1';
  return filters;
}

/** Writes the filters to the URL, leaving out the ones that match everything. */
function writeFilters(filters: MeetFilters): LocationQueryRaw {
  const query: LocationQueryRaw = {};
  if (filters.query) query.q = filters.query;
  for (const key of LIST_KEYS) {
    if (filters[key].length) query[key] = [...filters[key]];
  }
  if (filters.kind !== EventKind.All) query.kind = filters.kind;
  if (filters.upcomingOnly) query.upcoming = '1';
  return query;
}

/** Keeps the filters of the meet overview in the URL next to its other keys. */
export function useMeetFilters() {
  const route = useRoute();
  const router = useRouter();

  const filters = reactive<MeetFilters>(readFilters(route.query));

  watch(
    filters,
    () => {
      void replaceRouteQuery(route, router, (query) => ({
        ...Object.fromEntries(
          Object.entries(query).filter(([key]) => !MEET_FILTER_KEYS.includes(key)),
        ),
        ...writeFilters(filters),
      }));
    },
    {
      deep: true,
    },
  );

  /** Clears every filter. */
  function reset(): void {
    Object.assign(filters, emptyFilters());
  }

  return {
    filters,
    reset,
  };
}
