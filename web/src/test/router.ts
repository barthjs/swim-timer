import { reactive } from 'vue';
import { vi } from 'vitest';
import type { LocationQuery, LocationQueryRaw } from 'vue-router';

/**
 * fakeRouter stands in for useRoute and useRouter in composables that keep their state in the URL.
 * replace writes the query synchronously, dropping undefined values as vue-router does.
 */
export function fakeRouter(initial: LocationQuery = {}) {
  const route = reactive<{ query: LocationQuery }>({
    query: {
      ...initial,
    },
  });

  const replace = vi.fn<(location: { query?: LocationQueryRaw }) => Promise<void>>(
    async ({ query }) => {
      const next: LocationQuery = {};
      for (const [key, value] of Object.entries(query ?? {})) {
        if (value === undefined || value === null) continue;
        next[key] = Array.isArray(value) ? value.map(String) : String(value);
      }
      route.query = next;
    },
  );

  return {
    route,
    router: {
      replace,
    },
  };
}
