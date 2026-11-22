import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { changedPaths, getAtPath, setAtPath } from '@/lib/form/path';
import { fakeDistinct } from '@/test/fake';

describe('getAtPath', () => {
  const source = {
    app: {
      name: faker.lorem.word(),
      ports: [faker.internet.port(), faker.internet.port()],
    },
    empty: null,
  };

  it.each([
    {
      path: 'app',
      expected: source.app,
    },
    {
      path: 'app.name',
      expected: source.app.name,
    },
    {
      path: 'app.ports.1',
      expected: source.app.ports[1],
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
  const [name, changed] = fakeDistinct(() => faker.lorem.word(), 2);

  it('writes a nested field', () => {
    const target = {
      app: {
        name,
      },
    };

    setAtPath(target, 'app.name', changed);

    expect(target.app.name).toBe(changed);
  });

  it('writes a top-level field', () => {
    const target = {
      name,
    };

    setAtPath(target, 'name', changed);

    expect(target.name).toBe(changed);
  });

  it('ignores a path whose parent is missing', () => {
    const target = {
      app: null,
    };

    setAtPath(target, 'app.name', changed);

    expect(target).toEqual({
      app: null,
    });
  });
});

describe('changedPaths', () => {
  const [name, changed] = fakeDistinct(() => faker.lorem.word(), 2);

  it('returns no path for equal values', () => {
    const paths = changedPaths(
      {
        app: {
          name,
        },
      },
      {
        app: {
          name,
        },
      },
    );

    expect(paths).toEqual([]);
  });

  it('returns the path of a changed nested field', () => {
    const paths = changedPaths(
      {
        app: {
          name,
        },
        ports: [1],
      },
      {
        app: {
          name: changed,
        },
        ports: [1],
      },
    );

    expect(paths).toEqual(['app.name']);
  });

  it('returns the paths of added and removed entries', () => {
    const paths = changedPaths(
      {
        tags: [name],
      },
      {
        tags: [name, changed],
        extra: changed,
      },
    );

    expect(paths).toEqual(['tags.1', 'extra']);
  });

  it('returns the path of a branch replaced by a leaf', () => {
    const paths = changedPaths(
      {
        app: {
          name,
        },
      },
      {
        app: null,
      },
    );

    expect(paths).toEqual(['app.name']);
  });
});
