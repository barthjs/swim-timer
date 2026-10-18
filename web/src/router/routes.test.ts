import type { RouteRecordRaw } from 'vue-router';
import { describe, expect, it, vi } from 'vitest';

import AdminLayout from '@/layouts/AdminLayout.vue';
import de from '@/locales/de.json';
import en from '@/locales/en.json';
import { routes } from '@/router/routes';

vi.mock('@/layouts/AdminLayout.vue', () => ({
  default: {},
}));
vi.mock('@/layouts/AppLayout.vue', () => ({
  default: {},
}));

/** namedRoutes collects every named route of the tree, keyed by its name. */
function namedRoutes(records: RouteRecordRaw[]): Map<string, RouteRecordRaw> {
  const result = new Map<string, RouteRecordRaw>();
  for (const record of records) {
    if (typeof record.name === 'string') result.set(record.name, record);
    if (record.children) {
      for (const [name, child] of namedRoutes(record.children)) result.set(name, child);
    }
  }
  return result;
}

/** hasKey reports whether a dotted key resolves to a string in a locale. */
function hasKey(messages: unknown, key: string): boolean {
  let node = messages;
  for (const part of key.split('.')) {
    if (typeof node !== 'object' || node === null) return false;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === 'string';
}

const byName = namedRoutes(routes);

describe('routes', () => {
  it.each([...byName])('gives %s a title in every locale', (_name, route) => {
    const titleKey = route.meta?.titleKey;

    expect(titleKey).toBeDefined();
    expect(hasKey(en, titleKey ?? '')).toBe(true);
    expect(hasKey(de, titleKey ?? '')).toBe(true);
  });

  it.each(['dashboard', 'meet', 'meetData', 'editor', 'config'])(
    'shows %s in the admin layout',
    (name) => {
      const layout = routes.find((record) => record.component === AdminLayout);

      expect(layout?.children?.some((child) => child.name === name)).toBe(true);
    },
  );

  it.each(['meetData', 'config', 'editor'])('requires a login for %s', (name) => {
    const meta = byName.get(name)?.meta;

    expect(meta?.requiresAuth).toBe(true);
  });

  it.each(['dashboard', 'meet', 'scoreboard'])('keeps %s public', (name) => {
    const meta = byName.get(name)?.meta;

    expect(meta?.requiresAuth).toBeFalsy();
    expect(meta?.requiresGuest).toBeFalsy();
  });

  it('only lets guests open the login', () => {
    const meta = byName.get('login')?.meta;

    expect(meta?.requiresGuest).toBe(true);
  });

  it('gives only the editor the full width', () => {
    for (const [name, route] of byName) {
      const expected = name === 'editor' ? true : undefined;
      expect(route.meta?.fullWidth).toBe(expected);
    }
  });

  it('gives only the editor and the meet overview the viewport height', () => {
    for (const [name, route] of byName) {
      const expected = name === 'editor' || name === 'meet' ? true : undefined;
      expect(route.meta?.fillViewport).toBe(expected);
    }
  });

  it('sends unknown paths to the dashboard', () => {
    const fallback = routes.at(-1);

    expect(fallback?.path).toBe('/:pathMatch(.*)*');
    expect(fallback?.redirect).toEqual({
      name: 'dashboard',
    });
  });
});
