import { describe, expect, it } from 'vitest';

import { sameTableFilters } from '@/lib/table/quickFilters';

describe('sameTableFilters', () => {
  it.each([
    {
      name: 'accepts the same values',
      a: {
        status: 1,
        active: true,
      },
      b: {
        active: true,
        status: 1,
      },
      expected: true,
    },
    {
      name: 'compares the values as the URL holds them',
      a: {
        status: 1,
      },
      b: {
        status: '1',
      },
      expected: true,
    },
    {
      name: 'ignores filters without a value',
      a: {
        status: 1,
        user: undefined,
        q: '',
      },
      b: {
        status: 1,
      },
      expected: true,
    },
    {
      name: 'rejects another value',
      a: {
        status: 1,
      },
      b: {
        status: 2,
      },
      expected: false,
    },
    {
      name: 'rejects a further set filter',
      a: {
        status: 1,
        active: false,
      },
      b: {
        status: 1,
      },
      expected: false,
    },
  ])('$name', ({ a, b, expected }) => {
    expect(sameTableFilters(a, b)).toBe(expected);
  });
});
