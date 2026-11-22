import type {
  LocationQuery,
  LocationQueryRaw,
  LocationQueryValue,
  RouteLocationNormalizedLoaded,
  Router,
} from 'vue-router';

// The last change of the query each router runs, which the next change waits for
const pending = new WeakMap<Pick<Router, 'replace'>, Promise<void>>();

/** Returns the first value of a parameter of a query, or undefined for a missing or empty one. */
export function queryParam(
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): string | undefined {
  const raw = Array.isArray(value) ? value[0] : value;
  return typeof raw === 'string' && raw !== '' ? raw : undefined;
}

/**
 * Replaces the query of the current page without a history entry,
 * so changes in the same tick build on each other and a change for a page already left is dropped.
 */
export async function replaceRouteQuery(
  route: Pick<RouteLocationNormalizedLoaded, 'path' | 'query'>,
  router: Pick<Router, 'replace'>,
  change: (query: LocationQuery) => LocationQueryRaw,
): Promise<void> {
  const key = router;
  const previous = pending.get(key) ?? Promise.resolve();
  const path = route.path;
  const next = previous.then(async () => {
    // router.replace without a path targets the current page, which would carry the query of the page left
    if (route.path !== path) return;
    await router.replace({
      query: change(route.query),
      // Code outside the router may write the hash, so it is read from the page
      hash: typeof window === 'undefined' ? undefined : window.location.hash,
    });
  });
  // A failed change must not stop the ones after it
  pending.set(
    key,
    next.catch(() => undefined),
  );
  return next;
}
