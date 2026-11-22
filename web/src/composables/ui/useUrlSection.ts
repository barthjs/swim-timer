import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { replaceRouteQuery } from '@/lib/ui/routeQuery';
import { isOneOf } from '@/lib/types/guards';

/** The query parameter that names the open section of a page. */
const URL_SECTION_KEY = 'section';

/**
 * Keeps the open section of a page in the URL and drops the query of the other sections on a switch,
 * apart from the shared keys every section reads.
 */
export function useUrlSection<T extends string>(
  sections: MaybeRefOrGetter<readonly T[]>,
  defaultSection: T,
  sharedKeys: readonly string[] = [],
) {
  const route = useRoute();
  const router = useRouter();

  /** Returns the section the URL names, or the default for a missing or unknown one. */
  const current = (): T => {
    const raw = route.query[URL_SECTION_KEY];
    const value = Array.isArray(raw) ? raw[0] : raw;
    return isOneOf(toValue(sections), value) ? value : defaultSection;
  };

  return computed<T>({
    get: current,
    set: (value) => {
      if (value === current()) return;
      void replaceRouteQuery(route, router, (query) => {
        const shared = Object.fromEntries(
          Object.entries(query).filter(([key]) => sharedKeys.includes(key)),
        );
        return value === defaultSection
          ? shared
          : {
              ...shared,
              [URL_SECTION_KEY]: value,
            };
      });
    },
  });
}
