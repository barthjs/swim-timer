import { describe, expect, it } from 'vitest';

import { loginLocation, redirectTarget } from '@/lib/auth/redirect';

describe('loginLocation', () => {
  it('carries the full path', () => {
    const location = loginLocation('/en/login', '/items?filter=a');

    expect(location).toEqual({
      path: '/en/login',
      query: {
        redirect: '/items?filter=a',
      },
    });
  });

  it('leads to a named login route', () => {
    const location = loginLocation(
      {
        name: 'login',
      },
      '/items',
    );

    expect(location).toEqual({
      name: 'login',
      query: {
        redirect: '/items',
      },
    });
  });

  it.each([
    {
      name: 'the start page',
      path: '/',
    },
    {
      name: 'a missing path',
      path: undefined,
    },
  ])('omits the return path for $name', ({ path }) => {
    const location = loginLocation('/en/login', path);

    expect(location).toEqual({
      path: '/en/login',
    });
  });
});

describe('redirectTarget', () => {
  const fallback = {
    name: 'dashboard',
  };

  it('returns a path within the app', () => {
    const target = redirectTarget(
      {
        redirect: '/items/1',
      },
      fallback,
    );

    expect(target).toBe('/items/1');
  });

  it('rejects a missing path', () => {
    const target = redirectTarget({}, fallback);

    expect(target).toBe(fallback);
  });

  it.each([
    null,
    ['/a', '/b'],
    'https://example.com',
    '//example.com',
    '/\\example.com',
    'profile',
  ])('rejects %j', (value) => {
    const target = redirectTarget(
      {
        redirect: value,
      },
      fallback,
    );

    expect(target).toBe(fallback);
  });
});
