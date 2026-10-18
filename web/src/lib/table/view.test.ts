import { describe, expect, it } from 'vitest';

import {
  DEFAULT_PAGE_SIZE,
  formatSort,
  pageCount,
  pageOf,
  pageRows,
  parsePage,
  parsePageSize,
  parseSort,
  sortRows,
} from '@/lib/table/view';

type Row = { name: string; number: number };

const rows: Row[] = [
  {
    name: 'b',
    number: 2,
  },
  {
    name: 'a',
    number: 10,
  },
  {
    name: 'c',
    number: 2,
  },
];

const value = (row: Row, id: string) => row[id as keyof Row];

const many = Array.from(
  {
    length: 23,
  },
  (_, index) => index,
);

describe('parseSort', () => {
  it.each([
    {
      input: '-number',
      expected: {
        id: 'number',
        desc: true,
      },
    },
    {
      input: 'number',
      expected: {
        id: 'number',
        desc: false,
      },
    },
  ])('returns the column and direction of $input', ({ input, expected }) => {
    const sort = parseSort(input, ['number']);

    expect(sort).toEqual(expected);
  });

  it.each([
    {
      name: 'an unknown column',
      input: 'other',
    },
    {
      name: 'a value that is not a string',
      input: ['number'],
    },
  ])('rejects $name', ({ input }) => {
    const sort = parseSort(input, ['number']);

    expect(sort).toBeNull();
  });
});

describe('formatSort', () => {
  it('returns the column with the direction', () => {
    const formatted = formatSort({
      id: 'number',
      desc: true,
    });

    expect(formatted).toBe('-number');
  });

  it('returns undefined without a sort', () => {
    const formatted = formatSort(null);

    expect(formatted).toBeUndefined();
  });
});

describe('parsePageSize', () => {
  it('returns a known page size', () => {
    const size = parsePageSize('50');

    expect(size).toBe(50);
  });

  it('falls back to the default for an unknown page size', () => {
    const size = parsePageSize('7');

    expect(size).toBe(DEFAULT_PAGE_SIZE);
  });
});

describe('parsePage', () => {
  it('returns the page number', () => {
    const page = parsePage('3');

    expect(page).toBe(3);
  });

  it.each([
    {
      input: '0',
    },
    {
      input: 'x',
    },
  ])('falls back to the first page for $input', ({ input }) => {
    const page = parsePage(input);

    expect(page).toBe(1);
  });
});

describe('sortRows', () => {
  it.each([
    {
      desc: false,
      expected: ['b', 'c', 'a'],
    },
    {
      desc: true,
      expected: ['a', 'b', 'c'],
    },
  ])(
    'sorts numbers numerically and keeps the order of equal values with desc $desc',
    ({ desc, expected }) => {
      const sorted = sortRows(
        rows,
        {
          id: 'number',
          desc,
        },
        value,
      );

      expect(sorted.map((row) => row.name)).toEqual(expected);
    },
  );

  it('keeps the given order without a sort', () => {
    const sorted = sortRows(rows, null, value);

    expect(sorted).toEqual(rows);
  });
});

describe('pageCount', () => {
  it('returns the number of pages', () => {
    const count = pageCount(23, 10);

    expect(count).toBe(3);
  });

  it('keeps one page for an empty table', () => {
    const count = pageCount(0, 10);

    expect(count).toBe(1);
  });
});

describe('pageRows', () => {
  it('returns the rows of the page', () => {
    const page = pageRows(many, 3, 10);

    expect(page).toEqual([20, 21, 22]);
  });

  it('returns the last page for a page beyond it', () => {
    const page = pageRows(many, 9, 10);

    expect(page).toEqual([20, 21, 22]);
  });
});

describe('pageOf', () => {
  it('returns the page of the row', () => {
    const page = pageOf(many, 10, (row) => row === 15);

    expect(page).toBe(2);
  });

  it('returns null for an unknown row', () => {
    const page = pageOf(many, 10, (row) => row === 99);

    expect(page).toBeNull();
  });
});
