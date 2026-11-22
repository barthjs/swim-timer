import { describe, expect, it } from 'vitest';
import type { TableColumn } from '@nuxt/ui';

import {
  emptyTableColumnState,
  isConfigurableColumn,
  isTableColumnVisible,
  setTableColumnVisible,
  moveItem,
  parseTableColumnState,
  tableColumnId,
  tableColumnOptions,
  tableColumnOrder,
  tableColumnVisibility,
  isGrowColumn,
  isNumericColumn,
  isResizableColumn,
  tableColumnWidth,
  tableMeasuredSizing,
  tableMinWidth,
  tableRenderedSizing,
  tableResizableIds,
  tableResize,
  tableResizePartners,
  tableSortableColumn,
  TABLE_COLUMN_GROW_MIN_REM,
  TABLE_COLUMN_MIN_REM,
} from '@/lib/table/columns';

type Row = { name: string; email: string; role: string };

/** A selection column first, three configurable columns with a grow column, and an actions column last. */
const columns: TableColumn<Row>[] = [
  {
    id: 'select',
    header: () => null,
  },
  {
    accessorKey: 'name',
    header: 'Name',
    meta: {
      grow: true,
    },
  },
  {
    accessorKey: 'email',
    header: 'Email',
  },
  {
    accessorKey: 'role',
    header: 'Role',
    minSize: 80,
  },
  {
    id: 'actions',
  },
];

describe('parseTableColumnState', () => {
  it('reads a stored state', () => {
    const stored = {
      order: ['role', 'name'],
      hidden: ['email'],
      shown: ['role'],
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
      shown: [],
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

describe('isTableColumnVisible', () => {
  it('shows a column the viewer did not hide', () => {
    const visible = isTableColumnVisible('name', emptyTableColumnState());

    expect(visible).toBe(true);
  });

  it('hides a column the viewer hid', () => {
    const visible = isTableColumnVisible('email', {
      ...emptyTableColumnState(),
      hidden: ['email'],
    });

    expect(visible).toBe(false);
  });

  it('hides a column hidden by default', () => {
    const visible = isTableColumnVisible('role', emptyTableColumnState(), ['role']);

    expect(visible).toBe(false);
  });

  it('shows a column hidden by default that the viewer showed', () => {
    const visible = isTableColumnVisible(
      'role',
      {
        ...emptyTableColumnState(),
        shown: ['role'],
      },
      ['role'],
    );

    expect(visible).toBe(true);
  });
});

describe('setTableColumnVisible', () => {
  it('hides a column', () => {
    const state = setTableColumnVisible(emptyTableColumnState(), 'email', false);

    expect(state.hidden).toEqual(['email']);
  });

  it('shows a hidden column', () => {
    const state = setTableColumnVisible(
      {
        ...emptyTableColumnState(),
        hidden: ['email'],
      },
      'email',
      true,
    );

    expect(state.hidden).toEqual([]);
  });

  it('records a shown column that is hidden by default', () => {
    const state = setTableColumnVisible(emptyTableColumnState(), 'role', true, ['role']);

    expect(state).toEqual({
      ...emptyTableColumnState(),
      shown: ['role'],
    });
  });

  it('forgets a column hidden by default when the viewer hides it again', () => {
    const state = setTableColumnVisible(
      {
        ...emptyTableColumnState(),
        shown: ['role'],
      },
      'role',
      false,
      ['role'],
    );

    expect(state).toEqual(emptyTableColumnState());
  });
});

describe('tableColumnVisibility', () => {
  it('hides the hidden configurable columns only', () => {
    const visibility = tableColumnVisibility(columns, {
      ...emptyTableColumnState(),
      hidden: ['email', 'select'],
    });

    expect(visibility).toEqual({
      email: false,
    });
  });

  it('hides the columns hidden by default that the viewer did not show', () => {
    const visibility = tableColumnVisibility(columns, emptyTableColumnState(), ['role']);

    expect(visibility).toEqual({
      role: false,
    });
  });
});

describe('isGrowColumn', () => {
  it('reports a column marked to grow', () => {
    const grow = columns.map(isGrowColumn);

    expect(grow).toEqual([false, true, false, false, false]);
  });
});

describe('isNumericColumn', () => {
  it('reports a column marked as numeric', () => {
    const numericColumn: TableColumn<Row> = {
      accessorKey: 'role',
      header: 'Role',
      meta: {
        numeric: true,
      },
    };

    const numeric = [numericColumn, ...columns].map(isNumericColumn);

    expect(numeric).toEqual([true, false, false, false, false, false]);
  });
});

describe('tableColumnWidth', () => {
  /** The widths the content of the columns needs. */
  const measured = {
    name: 180,
    email: 200,
  };

  it('prefers a dragged width over the measured one', () => {
    const width = tableColumnWidth(
      columns[2]!,
      {
        email: 320,
      },
      measured,
    );

    expect(width).toBe(320);
  });

  it('uses the measured width without a dragged width', () => {
    const width = tableColumnWidth(columns[2]!, {}, measured);

    expect(width).toBe(200);
  });

  it('leaves a grow column without a width', () => {
    const width = tableColumnWidth(columns[1]!, {}, measured);

    expect(width).toBeUndefined();
  });

  it('leaves a grow column without a width after a drag as well', () => {
    const width = tableColumnWidth(
      columns[1]!,
      {
        name: 260,
      },
      measured,
    );

    expect(width).toBeUndefined();
  });
});

describe('tableMeasuredSizing', () => {
  /** Keep the widths between 50 and 300 pixels. */
  const bounds = {
    min: 50,
    max: 300,
  };

  it('maps the widths to the visible columns without the grow column', () => {
    const sizing = tableMeasuredSizing(
      columns,
      ['select', 'name', 'email', 'actions'],
      [52, 400, 180.4, 90],
      bounds,
    );

    expect(sizing).toEqual({
      select: 52,
      email: 181,
      actions: 90,
    });
  });

  it('keeps the widths within the bounds', () => {
    const sizing = tableMeasuredSizing(columns, ['select', 'email'], [20, 640], bounds);

    expect(sizing).toEqual({
      select: 50,
      email: 300,
    });
  });
});

describe('tableMinWidth', () => {
  /** The widths the content of the columns needs. */
  const measured = {
    select: 48,
    email: 200,
    role: 120,
    actions: 100,
  };

  it('adds the widths of the visible columns with a minimum for those without one', () => {
    const width = tableMinWidth(columns, emptyTableColumnState(), measured);

    expect(width).toEqual({
      pixels: 48 + 200 + 120 + 100,
      rem: TABLE_COLUMN_GROW_MIN_REM,
    });
  });

  it('leaves out hidden columns and counts dragged widths', () => {
    const width = tableMinWidth(
      columns,
      {
        order: [],
        hidden: ['email'],
        shown: [],
        sizing: {
          name: 300,
        },
      },
      measured,
    );

    expect(width).toEqual({
      pixels: 48 + 300 + 120 + 100,
      rem: 0,
    });
  });

  it('leaves out columns hidden by default', () => {
    const width = tableMinWidth(columns, emptyTableColumnState(), measured, ['role']);

    expect(width).toEqual({
      pixels: 48 + 200 + 100,
      rem: TABLE_COLUMN_GROW_MIN_REM,
    });
  });

  it('counts the narrowest width for a column not measured yet', () => {
    const width = tableMinWidth(columns, emptyTableColumnState(), {});

    expect(width).toEqual({
      pixels: 0,
      rem: TABLE_COLUMN_GROW_MIN_REM + 4 * TABLE_COLUMN_MIN_REM,
    });
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

describe('tableSortableColumn', () => {
  it('gives a sortable column without a value one', () => {
    const column = tableSortableColumn({
      id: 'status',
      header: 'Status',
      enableSorting: true,
    });

    expect('accessorFn' in column).toBe(true);
  });

  it('keeps the value of a column with an accessor', () => {
    const accessorFn = () => 'value';

    const column = tableSortableColumn({
      id: 'status',
      header: 'Status',
      enableSorting: true,
      accessorFn,
    });

    expect('accessorFn' in column && column.accessorFn).toBe(accessorFn);
  });

  it('leaves a column without sorting as it is', () => {
    const original = {
      id: 'status',
      header: 'Status',
    };

    const column = tableSortableColumn(original);

    expect(column).toBe(original);
  });
});

describe('tableResizableIds', () => {
  it('lists the visible resizable columns in the order of the table', () => {
    const ids = tableResizableIds(columns, ['select', 'role', 'name', 'actions']);

    expect(ids).toEqual(['role', 'name']);
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

describe('tableResize', () => {
  /** Four columns with the border after the second one, each 50 pixels above its minimum. */
  const row = [
    {
      id: 'a',
      width: 150,
      minSize: 100,
    },
    {
      id: 'b',
      width: 150,
      minSize: 100,
    },
    {
      id: 'c',
      width: 150,
      minSize: 100,
    },
    {
      id: 'd',
      width: 150,
      minSize: 100,
    },
  ];

  it('narrows the column after the border while it is above its minimum', () => {
    const widths = tableResize(30, row, 1, 0);

    expect(widths).toEqual({
      b: 180,
      c: 120,
    });
  });

  it('narrows the columns further right once the next one reached its minimum', () => {
    const widths = tableResize(80, row, 1, 0);

    expect(widths).toEqual({
      b: 230,
      c: 100,
      d: 120,
    });
  });

  it('grows the table once every column after the border reached its minimum', () => {
    const widths = tableResize(200, row, 1, 0);

    expect(widths).toEqual({
      b: 350,
      c: 100,
      d: 100,
    });
  });

  it('narrows the columns further left once the column before the border reached its minimum', () => {
    const widths = tableResize(-80, row, 1, 0);

    expect(widths).toEqual({
      a: 120,
      b: 100,
      c: 230,
    });
  });

  it('stops once every column before the border reached its minimum', () => {
    const widths = tableResize(-500, row, 1, 0);

    expect(widths).toEqual({
      a: 100,
      b: 100,
      c: 250,
    });
  });

  it('first gives back the width the table has beyond its container', () => {
    const widths = tableResize(-30, row, 1, 20);

    expect(widths).toEqual({
      b: 120,
      c: 160,
    });
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
