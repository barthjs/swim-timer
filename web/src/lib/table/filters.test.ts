import { describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import {
  tableDateRangeLabel,
  tableFilterPill,
  tableOptionLabel,
  withoutEmpty,
} from '@/lib/table/filters';

describe('tableFilterPill', () => {
  it('returns a pill for a set filter', () => {
    const key = faker.lorem.word();
    const label = faker.lorem.word();
    const value = faker.lorem.word();
    const remove = vi.fn<() => void>();

    const pills = tableFilterPill(key, label, value, remove);

    expect(pills).toEqual([
      {
        key,
        label,
        value,
        remove,
      },
    ]);
  });

  it.each([undefined, ''])('returns no pill for the empty value %j', (value) => {
    const pills = tableFilterPill(
      faker.lorem.word(),
      faker.lorem.word(),
      value,
      vi.fn<() => void>(),
    );

    expect(pills).toEqual([]);
  });
});

describe('tableOptionLabel', () => {
  const items = [
    {
      label: faker.lorem.word(),
      value: 1,
    },
    {
      label: faker.lorem.word(),
      value: 2,
    },
  ];

  it('returns the label of the matching option', () => {
    const label = tableOptionLabel(items, 2);

    expect(label).toBe(items[1]?.label);
  });

  it.each([
    {
      name: 'a value',
      value: undefined,
    },
    {
      name: 'a match',
      value: 3,
    },
  ])('returns undefined without $name', ({ value }) => {
    const label = tableOptionLabel(items, value);

    expect(label).toBeUndefined();
  });
});

describe('tableDateRangeLabel', () => {
  const format = (value: string, withYear: boolean) =>
    value
      .split('-')
      .reverse()
      .slice(0, withYear ? 3 : 2)
      .join('.');
  const labels = {
    from: faker.lorem.word(),
    to: faker.lorem.word(),
  };

  it('joins both ends of a range within one year and names the year once', () => {
    const range = {
      from: '2026-05-01',
      to: '2026-05-31',
    };

    const label = tableDateRangeLabel(range, format, labels);

    expect(label).toBe('01.05 – 31.05.2026');
  });

  it('names both years of a range across years', () => {
    const range = {
      from: '2025-12-01',
      to: '2026-01-31',
    };

    const label = tableDateRangeLabel(range, format, labels);

    expect(label).toBe('01.12.2025 – 31.01.2026');
  });

  it.each([
    {
      name: 'start',
      range: {
        from: '2026-05-01',
        to: '',
      },
      expected: `${labels.from} 01.05.2026`,
    },
    {
      name: 'end',
      range: {
        from: '',
        to: '2026-05-31',
      },
      expected: `${labels.to} 31.05.2026`,
    },
  ])('names the $name of an open range', ({ range, expected }) => {
    const label = tableDateRangeLabel(range, format, labels);

    expect(label).toBe(expected);
  });

  it('returns undefined without a range', () => {
    const range = {
      from: '',
      to: '',
    };

    const label = tableDateRangeLabel(range, format, labels);

    expect(label).toBeUndefined();
  });
});

describe('withoutEmpty', () => {
  it('leaves out the filters without a value', () => {
    expect(
      withoutEmpty({
        status: 'open',
        user: undefined,
        q: '',
      }),
    ).toEqual({
      status: 'open',
    });
  });
});
