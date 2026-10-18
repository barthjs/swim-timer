import { describe, expect, it } from 'vitest';

import { getAtPath, setAtPath } from '@/lib/form/path';

describe('getAtPath', () => {
  const source = {
    app: {
      name: 'Example',
      ports: [80, 443],
    },
    empty: null,
  };

  it.each([
    {
      path: 'app',
      expected: {
        name: 'Example',
        ports: [80, 443],
      },
    },
    {
      path: 'app.name',
      expected: 'Example',
    },
    {
      path: 'app.ports.1',
      expected: 443,
    },
    {
      path: 'app.missing',
      expected: undefined,
    },
    {
      path: 'empty.name',
      expected: undefined,
    },
  ])('returns the value at $path', ({ path, expected }) => {
    const value = getAtPath(source, path);

    expect(value).toEqual(expected);
  });
});

describe('setAtPath', () => {
  it('writes a nested field', () => {
    const target = {
      app: {
        name: 'Example',
      },
    };

    setAtPath(target, 'app.name', 'Changed');

    expect(target.app.name).toBe('Changed');
  });

  it('writes a top-level field', () => {
    const target = {
      name: 'Example',
    };

    setAtPath(target, 'name', 'Changed');

    expect(target.name).toBe('Changed');
  });

  it('ignores a path whose parent is missing', () => {
    const target = {
      app: null,
    };

    setAtPath(target, 'app.name', 'Changed');

    expect(target).toEqual({
      app: null,
    });
  });
});
