import type { TableColumn } from '@nuxt/ui';
import type { RowData } from '@tanstack/table-core';

import { isRecord } from '@/lib/types/guards';

declare module '@tanstack/table-core' {
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions -- augmentation merges into the interface of the library
  interface ColumnMeta<TData extends RowData, TValue> {
    // The column takes the width the measured columns leave
    grow?: boolean;
    // The column holds quantities, which align at the end so their digits line up
    numeric?: boolean;
  }
}

/** The order, visibility, and dragged widths in pixels a viewer gave the columns of a table. */
export type TableColumnState = {
  order: string[];
  hidden: string[];
  shown: string[];
  sizing: Record<string, number>;
};

/** The sorting model of UTable, with the sorted column first. */
export type TableSorting = { id: string; desc: boolean }[];

/** A configurable column as the column menu lists it. */
export type TableColumnOption = {
  id: string;
  label: string;
};

/** The id of the column with the row actions, which stays at the right edge. */
export const TABLE_ACTIONS_COLUMN = 'actions';

/** The narrowest width in rem a column is measured or dragged to. */
export const TABLE_COLUMN_MIN_REM = 3;

/** The widest width in rem a column is measured to. */
export const TABLE_COLUMN_MAX_REM = 20;

/** The width in rem the minimum table width reserves for a grow column. */
export const TABLE_COLUMN_GROW_MIN_REM = 10;

/** The narrowest and the widest width in pixels a measured column gets. */
export type TableWidthBounds = {
  min: number;
  max: number;
};

/** The width the visible columns need, in pixels for the sized ones and in rem for the others. */
export type TableMinWidth = {
  pixels: number;
  rem: number;
};

/** Returns the state of a table whose columns are unchanged. */
export function emptyTableColumnState(): TableColumnState {
  return {
    order: [],
    hidden: [],
    shown: [],
    sizing: {},
  };
}

/** Reads a stored column state and drops whatever does not fit its shape. */
export function parseTableColumnState(raw: unknown): TableColumnState | undefined {
  if (!isRecord(raw)) return undefined;
  const value = raw;

  const sizing: Record<string, number> = {};
  if (typeof value.sizing === 'object' && value.sizing !== null) {
    for (const [id, size] of Object.entries(value.sizing)) {
      if (typeof size === 'number' && Number.isFinite(size) && size > 0) sizing[id] = size;
    }
  }

  return {
    order: stringList(value.order),
    hidden: stringList(value.hidden),
    shown: stringList(value.shown),
    sizing,
  };
}

/** Returns the strings of a value that should be a list of strings. */
function stringList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string');
}

/** Returns the id TanStack Table gives a column: its id, or else its accessor key. */
export function tableColumnId<T>(column: TableColumn<T>): string {
  if (column.id) return column.id;
  if ('accessorKey' in column) return String(column.accessorKey);
  return '';
}

/** Reports whether a viewer may hide, move, and resize a column, which takes a text header. */
export function isConfigurableColumn<T>(column: TableColumn<T>): boolean {
  return typeof column.header === 'string';
}

/**
 * Returns the configurable columns in the order the viewer chose.
 * Columns the stored order does not know yet follow in the order of the page.
 */
export function tableColumnOptions<T>(
  columns: TableColumn<T>[],
  order: string[],
): TableColumnOption[] {
  const options = columns.filter(isConfigurableColumn).map((column) => ({
    id: tableColumnId(column),
    label: String(column.header),
  }));
  const byId = new Map(options.map((option) => [option.id, option]));

  const ordered = order
    .map((id) => byId.get(id))
    .filter((option): option is TableColumnOption => option !== undefined);
  const missing = options.filter((option) => !order.includes(option.id));
  return [...ordered, ...missing];
}

/**
 * Returns the ids of all columns with the configurable ones as one block in the order of the viewer,
 * while the other columns keep their place.
 */
export function tableColumnOrder<T>(columns: TableColumn<T>[], order: string[]): string[] {
  const configurable = tableColumnOptions(columns, order).map((option) => option.id);

  const result: string[] = [];
  let blockInserted = false;
  for (const column of columns) {
    if (!isConfigurableColumn(column)) {
      result.push(tableColumnId(column));
      continue;
    }
    if (blockInserted) continue;
    result.push(...configurable);
    blockInserted = true;
  }
  return result;
}

/**
 * Reports whether a column is shown. A column the page hides by default stays hidden until the viewer shows it,
 * every other column until the viewer hides it.
 */
export function isTableColumnVisible(
  id: string,
  state: TableColumnState,
  hiddenByDefault: string[] = [],
): boolean {
  if (hiddenByDefault.includes(id)) return state.shown.includes(id);
  return !state.hidden.includes(id);
}

/** Returns the state with a column shown or hidden. */
export function setTableColumnVisible(
  state: TableColumnState,
  id: string,
  visible: boolean,
  hiddenByDefault: string[] = [],
): TableColumnState {
  const hidden = state.hidden.filter((hiddenId) => hiddenId !== id);
  const shown = state.shown.filter((shownId) => shownId !== id);
  if (hiddenByDefault.includes(id)) {
    if (visible) shown.push(id);
  } else if (!visible) {
    hidden.push(id);
  }
  return {
    ...state,
    hidden,
    shown,
  };
}

/** Returns the visibility model of TanStack Table for the hidden columns. */
export function tableColumnVisibility<T>(
  columns: TableColumn<T>[],
  state: TableColumnState,
  hiddenByDefault: string[] = [],
): Record<string, boolean> {
  const visibility: Record<string, boolean> = {};
  for (const column of columns) {
    if (!isConfigurableColumn(column)) continue;
    const id = tableColumnId(column);
    if (!isTableColumnVisible(id, state, hiddenByDefault)) visibility[id] = false;
  }
  return visibility;
}

/** Reports whether a column takes the width the measured columns leave. */
export function isGrowColumn<T>(column: TableColumn<T>): boolean {
  return column.meta?.grow === true;
}

/** Reports whether a column holds quantities that align at the end. */
export function isNumericColumn<T>(column: TableColumn<T>): boolean {
  return column.meta?.numeric === true;
}

/**
 * Returns the dragged or else the measured width in pixels of a column,
 * or undefined for a grow column, which takes the space the others leave.
 */
export function tableColumnWidth<T>(
  column: TableColumn<T>,
  sizing: Record<string, number>,
  measured: Record<string, number>,
): number | undefined {
  if (isGrowColumn(column)) return undefined;
  const id = tableColumnId(column);
  return sizing[id] ?? measured[id];
}

/**
 * Returns the measured width of every visible column except the grow columns,
 * kept within the bounds. visibleIds and widths list the visible columns in the order of the table.
 */
export function tableMeasuredSizing<T>(
  columns: TableColumn<T>[],
  visibleIds: string[],
  widths: number[],
  bounds: TableWidthBounds,
): Record<string, number> {
  const grow = new Set(columns.filter(isGrowColumn).map(tableColumnId));
  const result: Record<string, number> = {};
  visibleIds.forEach((id, index) => {
    const width = widths[index];
    if (grow.has(id) || width === undefined) return;
    result[id] = Math.min(Math.max(Math.ceil(width), bounds.min), bounds.max);
  });
  return result;
}

/**
 * Returns the width the visible columns need, with a minimum for those not measured yet.
 * With a fixed table layout the browser lets a column without a width shrink to nothing
 * once the table gets too narrow, so the table keeps this width and scrolls sideways instead.
 */
export function tableMinWidth<T>(
  columns: TableColumn<T>[],
  state: TableColumnState,
  measured: Record<string, number>,
  hiddenByDefault: string[] = [],
): TableMinWidth {
  const result: TableMinWidth = {
    pixels: 0,
    rem: 0,
  };
  for (const column of columns) {
    if (
      isConfigurableColumn(column) &&
      !isTableColumnVisible(tableColumnId(column), state, hiddenByDefault)
    ) {
      continue;
    }
    const width = isGrowColumn(column)
      ? state.sizing[tableColumnId(column)]
      : tableColumnWidth(column, state.sizing, measured);
    if (width !== undefined) result.pixels += width;
    else result.rem += isGrowColumn(column) ? TABLE_COLUMN_GROW_MIN_REM : TABLE_COLUMN_MIN_REM;
  }
  return result;
}

/** Reports whether a viewer may drag the width of a configurable column. */
export function isResizableColumn<T>(column: TableColumn<T>): boolean {
  return column.enableResizing ?? isConfigurableColumn(column);
}

/**
 * Gives a sortable column without a value one, since UTable only offers sorting for a column with a value.
 * DataTable sorts manually, so the value is never read, and a column with a value keeps it.
 */
export function tableSortableColumn<T>(column: TableColumn<T>): TableColumn<T> {
  if (!column.enableSorting || 'accessorKey' in column || 'accessorFn' in column) return column;
  return {
    ...column,
    accessorFn: () => undefined,
  };
}

/**
 * Returns the visible columns a viewer may drag the width of, in the order of the table.
 * visibleIds lists the visible columns in the order of the table.
 */
export function tableResizableIds<T>(columns: TableColumn<T>[], visibleIds: string[]): string[] {
  const resizable = new Set(columns.filter(isResizableColumn).map(tableColumnId));
  return visibleIds.filter((id) => resizable.has(id));
}

/**
 * Maps every visible resizable column to the next one, whose shared border its handle moves,
 * for the visible columns in the order of the table.
 */
export function tableResizePartners<T>(
  columns: TableColumn<T>[],
  visibleIds: string[],
): Record<string, string> {
  const ids = tableResizableIds(columns, visibleIds);
  const partners: Record<string, string> = {};
  for (let index = 0; index < ids.length - 1; index++) {
    partners[ids[index]!] = ids[index + 1]!;
  }
  return partners;
}

/** A resizable column with the width a move of a border starts from and how narrow it may get. */
export type TableResizeColumn = {
  id: string;
  width: number;
  minSize: number;
};

/**
 * Moves the border after the column at the index by the delta, narrowing the nearest columns on the side
 * it moves to down to their minimum one after another, and returns the widths it changes.
 */
export function tableResize(
  delta: number,
  columns: TableResizeColumn[],
  index: number,
  overflow: number,
): Record<string, number> {
  const widths: Record<string, number> = {};
  const before = columns.slice(0, index + 1).reverse();
  const after = columns.slice(index + 1);
  if (delta >= 0) {
    narrow(after, delta, widths);
    const current = columns[index];
    if (current) widths[current.id] = current.width + delta;
    return widths;
  }
  const given = narrow(before, -delta, widths);
  const next = after[0];
  if (next) widths[next.id] = next.width + Math.max(0, given - Math.max(0, overflow));
  return widths;
}

/** Takes up to the amount from the columns in their order, each down to its minimum, and returns what it took. */
function narrow(
  columns: TableResizeColumn[],
  amount: number,
  widths: Record<string, number>,
): number {
  let rest = amount;
  for (const column of columns) {
    if (rest <= 0) break;
    const taken = Math.min(rest, Math.max(0, column.width - column.minSize));
    if (taken > 0) widths[column.id] = column.width - taken;
    rest -= taken;
  }
  return amount - rest;
}

/**
 * Returns the sizing with the rendered width of every visible resizable column,
 * so the stored widths match what the viewer sees before a border moves.
 */
export function tableRenderedSizing<T>(
  columns: TableColumn<T>[],
  sizing: Record<string, number>,
  visibleIds: string[],
  renderedWidths: number[],
): Record<string, number> {
  const resizable = new Set(columns.filter(isResizableColumn).map(tableColumnId));
  const result = {
    ...sizing,
  };
  visibleIds.forEach((id, index) => {
    const width = renderedWidths[index];
    if (!resizable.has(id) || width === undefined) return;
    result[id] = Math.round(width);
  });
  return result;
}

/** Returns a copy of a list with one item moved from one index to another. */
export function moveItem<T>(items: T[], from: number, to: number): T[] {
  const result = [...items];
  const [moved] = result.splice(from, 1);
  if (moved === undefined) return result;
  result.splice(to, 0, moved);
  return result;
}
