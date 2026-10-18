import { describe, expect, it } from 'vitest';
import type { TableColumn } from '@nuxt/ui';

import {
  emptyTableColumnState,
  isConfigurableColumn,
  moveItem,
  parseTableColumnState,
  tableColumnId,
  tableColumnOptions,
  tableColumnOrder,
  tableColumnVisibility,
  isResizableColumn,
  tableColumnWidth,
  tableMinWidth,
  tableRenderedSizing,
  tableResizeDelta,
  tableResizePartners,
  TABLE_COLUMN_FLEXIBLE_MIN_SIZE,
} from '@/lib/table/columns';

/** Row is a row of the tables under test. */
type Row = { name: string; email: string; role: string };

/** A selection column first, three configurable columns, and an actions column last. */
const columns: TableColumn<Row>[] = [
  {
    id: 'select',
    size: 48,
    header: () => null,
  },
  {
    accessorKey: 'name',
    header: 'Name',
  },
  {
    accessorKey: 'email',
    header: 'Email',
    size: 200,
  },
  {
    accessorKey: 'role',
    header: 'Role',
    size: 120,
    minSize: 80,
  },
  {
    id: 'actions',
    size: 100,
  },
];

describe('parseTableColumnState', () => {
  it('reads a stored state', () => {
    const stored = {
      order: ['role', 'name'],
      hidden: ['email'],
      sizing: {
        name: 240,
      },
    };

    const state = parseTableColumnState(stored);

    expect(state).toEqual(stored);
  });

  it('drops values that do not fit the shape', () => {
    const stored = {
      order: ['name', 3],
      hidden: 'email',
      sizing: {
        name: -1,
        email: 'wide',
        role: 90,
      },
    };

    const state = parseTableColumnState(stored);

    expect(state).toEqual({
      order: ['name'],
      hidden: [],
      sizing: {
        role: 90,
      },
    });
  });

  it.each(['broken', null])('rejects %j as no object', (stored) => {
    const state = parseTableColumnState(stored);

    expect(state).toBeUndefined();
  });
});

describe('tableColumnId', () => {
  it('uses the id of a column, or else its accessor key', () => {
    const ids = columns.map(tableColumnId);

    expect(ids).toEqual(['select', 'name', 'email', 'role', 'actions']);
  });
});

describe('isConfigurableColumn', () => {
  it('accepts only columns with a text header', () => {
    const configurable = columns.map(isConfigurableColumn);

    expect(configurable).toEqual([false, true, true, true, false]);
  });
});

describe('tableColumnOptions', () => {
  it('lists the configurable columns in the stored order, followed by the unknown ones', () => {
    const options = tableColumnOptions(columns, ['role', 'removed', 'name']);

    expect(options).toEqual([
      {
        id: 'role',
        label: 'Role',
      },
      {
        id: 'name',
        label: 'Name',
      },
      {
        id: 'email',
        label: 'Email',
      },
    ]);
  });
});

describe('tableColumnOrder', () => {
  it('moves the configurable columns as one block and keeps the fixed ones in place', () => {
    const order = tableColumnOrder(columns, ['role', 'email', 'name']);

    expect(order).toEqual(['select', 'role', 'email', 'name', 'actions']);
  });

  it('keeps the order of the page without a stored order', () => {
    const order = tableColumnOrder(columns, []);

    expect(order).toEqual(['select', 'name', 'email', 'role', 'actions']);
  });
});

describe('tableColumnVisibility', () => {
  it('hides the hidden configurable columns only', () => {
    const visibility = tableColumnVisibility(columns, ['email', 'select']);

    expect(visibility).toEqual({
      email: false,
    });
  });
});

describe('tableColumnWidth', () => {
  it('prefers a dragged width over the size of the page', () => {
    const width = tableColumnWidth(columns[2]!, {
      email: 320,
    });

    expect(width).toBe(320);
  });

  it('uses the size of the page without a dragged width', () => {
    const width = tableColumnWidth(columns[2]!, {});

    expect(width).toBe(200);
  });

  it('leaves a column without a size flexible', () => {
    const width = tableColumnWidth(columns[1]!, {});

    expect(width).toBeUndefined();
  });
});

describe('tableMinWidth', () => {
  it('adds the widths of the visible columns with a minimum for flexible ones', () => {
    const width = tableMinWidth(columns, emptyTableColumnState());

    expect(width).toBe(48 + TABLE_COLUMN_FLEXIBLE_MIN_SIZE + 200 + 120 + 100);
  });

  it('leaves out hidden columns and counts dragged widths', () => {
    const width = tableMinWidth(columns, {
      order: [],
      hidden: ['email'],
      sizing: {
        name: 300,
      },
    });

    expect(width).toBe(48 + 300 + 120 + 100);
  });
});

describe('isResizableColumn', () => {
  it('lets configurable columns be resized', () => {
    const resizable = columns.map(isResizableColumn);

    expect(resizable).toEqual([false, true, true, true, false]);
  });

  it('rejects a column the page does not let resize', () => {
    const resizable = isResizableColumn({
      accessorKey: 'name',
      header: 'Name',
      enableResizing: false,
    });

    expect(resizable).toBe(false);
  });
});

describe('tableResizePartners', () => {
  it('pairs every visible resizable column with the next one', () => {
    const partners = tableResizePartners(columns, ['select', 'name', 'email', 'role', 'actions']);

    expect(partners).toEqual({
      name: 'email',
      email: 'role',
    });
  });

  it('skips hidden columns and follows the order of the table', () => {
    const partners = tableResizePartners(columns, ['select', 'role', 'name', 'actions']);

    expect(partners).toEqual({
      role: 'name',
    });
  });
});

describe('tableResizeDelta', () => {
  const left = {
    width: 200,
    minSize: 60,
  };
  const right = {
    width: 150,
    minSize: 100,
  };

  it.each([30, -30])('moves the border freely by %i between the minimums', (delta) => {
    const moved = tableResizeDelta(delta, left, right);

    expect(moved).toBe(delta);
  });

  it.each([
    {
      name: 'the right',
      delta: 80,
      expected: 50,
    },
    {
      name: 'the left',
      delta: -500,
      expected: -140,
    },
  ])('stops where $name side reaches its minimum', ({ delta, expected }) => {
    const moved = tableResizeDelta(delta, left, right);

    expect(moved).toBe(expected);
  });
});

describe('tableRenderedSizing', () => {
  it('stores the rendered width of every visible resizable column', () => {
    const sizing = tableRenderedSizing(
      columns,
      {
        role: 90,
        hidden: 70,
      },
      ['select', 'name', 'email', 'role', 'actions'],
      [48, 412.6, 210, 120, 100],
    );

    expect(sizing).toEqual({
      hidden: 70,
      name: 413,
      email: 210,
      role: 120,
    });
  });
});

describe('moveItem', () => {
  it.each([
    {
      name: 'backward',
      from: 3,
      to: 1,
      expected: ['a', 'd', 'b', 'c'],
    },
    {
      name: 'forward',
      from: 0,
      to: 2,
      expected: ['b', 'c', 'a', 'd'],
    },
  ])('moves an item $name', ({ from, to, expected }) => {
    const moved = moveItem(['a', 'b', 'c', 'd'], from, to);

    expect(moved).toEqual(expected);
  });

  it('leaves the list unchanged for an index outside of it', () => {
    const moved = moveItem(['a', 'b'], 5, 0);

    expect(moved).toEqual(['a', 'b']);
  });
});
