import { describe, expect, it, vi } from 'vitest';

import { tableDateRangeLabel, tableFilterPill, tableOptionLabel } from '@/lib/table/filters';

describe('tableFilterPill', () => {
  it('returns a pill for a set filter', () => {
    const remove = vi.fn<() => void>();

    const pills = tableFilterPill('status', 'Status', 'Active', remove);

    expect(pills).toEqual([
      {
        key: 'status',
        label: 'Status',
        value: 'Active',
        remove,
      },
    ]);
  });

  it.each([undefined, ''])('returns no pill for the empty value %j', (value) => {
    const pills = tableFilterPill('status', 'Status', value, vi.fn<() => void>());

    expect(pills).toEqual([]);
  });
});

describe('tableOptionLabel', () => {
  const items = [
    {
      label: 'Active',
      value: 1,
    },
    {
      label: 'Pending',
      value: 2,
    },
  ];

  it('returns the label of the matching option', () => {
    const label = tableOptionLabel(items, 2);

    expect(label).toBe('Pending');
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
  const format = (value: string) => value.split('-').reverse().join('.');
  const labels = {
    from: 'from',
    to: 'until',
  };

  it('joins both ends of a closed range', () => {
    const range = {
      from: '2026-05-01',
      to: '2026-05-31',
    };

    const label = tableDateRangeLabel(range, format, labels);

    expect(label).toBe('01.05.2026 – 31.05.2026');
  });

  it.each([
    {
      name: 'start',
      range: {
        from: '2026-05-01',
        to: '',
      },
      expected: 'from 01.05.2026',
    },
    {
      name: 'end',
      range: {
        from: '',
        to: '2026-05-31',
      },
      expected: 'until 31.05.2026',
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
