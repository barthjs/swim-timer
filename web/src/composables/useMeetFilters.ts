import { reactive, watch } from 'vue';
import { useRoute, useRouter, type LocationQuery, type LocationQueryRaw } from 'vue-router';

import { EventKind, emptyFilters, type MeetFilters } from '@/lib/meet';

/** LIST_KEYS are the query parameters of the filters that hold several values. */
const LIST_KEYS = ['days', 'styles', 'distances', 'categories', 'clubs'] as const;

/** readList reads a query parameter that may occur several times. */
function readList(query: LocationQuery, key: string): string[] {
  const value = query[key];
  const values = Array.isArray(value) ? value : [value];
  return values.filter((item): item is string => typeof item === 'string' && item !== '');
}

/** readFilters reads the filters from the URL. */
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

/** writeFilters writes the filters to the URL, leaving out the ones that match everything. */
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

/**
 * useMeetFilters keeps the filters of the meet overview in the URL,
 * so a filtered view can be shared as a link and survives a reload.
 * Keys other than the filters, such as the tab, are kept.
 */
export function useMeetFilters() {
  const route = useRoute();
  const router = useRouter();

  const filters = reactive<MeetFilters>(readFilters(route.query));

  watch(
    filters,
    () => {
      const others = Object.fromEntries(
        Object.entries(route.query).filter(
          ([key]) => !['q', 'kind', 'upcoming', ...LIST_KEYS].includes(key),
        ),
      );
      void router.replace({
        query: {
          ...others,
          ...writeFilters(filters),
        },
      });
    },
    {
      deep: true,
    },
  );

  /** reset clears every filter. */
  function reset(): void {
    Object.assign(filters, emptyFilters());
  }

  return {
    filters,
    reset,
  };
}
