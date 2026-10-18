import type { TableColumn } from '@nuxt/ui';

/**
 * TableColumnState is what a viewer changed about the columns of a table: the order and the hidden columns
 * among the configurable ones, and the widths dragged with the resize handle in pixels.
 */
export type TableColumnState = {
  order: string[];
  hidden: string[];
  sizing: Record<string, number>;
};

/** TableSorting is the sorting model of UTable, with the sorted column first. */
export type TableSorting = { id: string; desc: boolean }[];

/** TableColumnOption is a configurable column as the column menu lists it. */
export type TableColumnOption = {
  id: string;
  label: string;
};

/** TABLE_ACTIONS_COLUMN is the id of the column with the row actions, which stays at the right edge. */
export const TABLE_ACTIONS_COLUMN = 'actions';

/** TABLE_COLUMN_MIN_SIZE is the narrowest width in pixels a column can be dragged to. */
export const TABLE_COLUMN_MIN_SIZE = 60;

/**
 * TABLE_COLUMN_FLEXIBLE_MIN_SIZE is the width in pixels the minimum table width reserves for a column
 * without a size of its own.
 */
export const TABLE_COLUMN_FLEXIBLE_MIN_SIZE = 140;

/** emptyTableColumnState returns the state of a table whose columns are unchanged. */
export function emptyTableColumnState(): TableColumnState {
  return {
    order: [],
    hidden: [],
    sizing: {},
  };
}

/** parseTableColumnState reads a stored column state and drops whatever does not fit its shape. */
export function parseTableColumnState(raw: unknown): TableColumnState | undefined {
  if (typeof raw !== 'object' || raw === null) return undefined;
  const value = raw as Record<string, unknown>;

  const sizing: Record<string, number> = {};
  if (typeof value.sizing === 'object' && value.sizing !== null) {
    for (const [id, size] of Object.entries(value.sizing)) {
      if (typeof size === 'number' && Number.isFinite(size) && size > 0) sizing[id] = size;
    }
  }

  return {
    order: stringList(value.order),
    hidden: stringList(value.hidden),
    sizing,
  };
}

/** stringList returns the strings of a value that should be a list of strings. */
function stringList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string');
}

/** tableColumnId returns the id TanStack Table gives a column: its id, or else its accessor key. */
export function tableColumnId<T>(column: TableColumn<T>): string {
  if (column.id) return column.id;
  if ('accessorKey' in column && column.accessorKey !== undefined)
    return String(column.accessorKey);
  return '';
}

/**
 * isConfigurableColumn reports whether a viewer may hide, move, and resize a column.
 * These are the columns with a text header. Columns that render their own header,
 * such as the selection or the actions, stay where the page puts them.
 */
export function isConfigurableColumn<T>(column: TableColumn<T>): boolean {
  return typeof column.header === 'string';
}

/**
 * tableColumnOptions returns the configurable columns in the order the viewer chose.
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
 * tableColumnOrder returns the ids of all columns in the order of the table. The configurable columns form one block
 * in the order the viewer chose, at the place of the first configurable column of the page.
 * Every other column keeps its place, so the selection stays first and the actions last.
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

/** tableColumnVisibility returns the visibility model of TanStack Table for the hidden columns. */
export function tableColumnVisibility<T>(
  columns: TableColumn<T>[],
  hidden: string[],
): Record<string, boolean> {
  const visibility: Record<string, boolean> = {};
  for (const column of columns) {
    if (!isConfigurableColumn(column)) continue;
    const id = tableColumnId(column);
    if (hidden.includes(id)) visibility[id] = false;
  }
  return visibility;
}

/**
 * tableColumnWidth returns the width in pixels a column is shown with, or undefined for a flexible column.
 * A dragged width wins over the size of the page. A column without either shares the space the others leave.
 */
export function tableColumnWidth<T>(
  column: TableColumn<T>,
  sizing: Record<string, number>,
): number | undefined {
  return sizing[tableColumnId(column)] ?? column.size;
}

/**
 * tableMinWidth returns the width in pixels the visible columns need. With a fixed table layout the browser lets
 * a flexible column shrink to nothing once the table gets too narrow, so the table keeps this width
 * and scrolls sideways instead.
 */
export function tableMinWidth<T>(columns: TableColumn<T>[], state: TableColumnState): number {
  let total = 0;
  for (const column of columns) {
    if (state.hidden.includes(tableColumnId(column)) && isConfigurableColumn(column)) continue;
    total +=
      tableColumnWidth(column, state.sizing) ?? column.minSize ?? TABLE_COLUMN_FLEXIBLE_MIN_SIZE;
  }
  return total;
}

/**
 * isResizableColumn reports whether a viewer may drag the width of a column.
 * Configurable columns are, unless the page sets `enableResizing`.
 */
export function isResizableColumn<T>(column: TableColumn<T>): boolean {
  return column.enableResizing ?? isConfigurableColumn(column);
}

/**
 * tableResizePartners maps every visible resizable column to the next visible resizable column.
 * The handle at the right edge of a column moves the border between the two, so what one gains the other gives up
 * and the table keeps its width. A column without a partner has no handle.
 * visibleIds lists the visible columns in the order of the table.
 */
export function tableResizePartners<T>(
  columns: TableColumn<T>[],
  visibleIds: string[],
): Record<string, string> {
  const resizable = new Set(columns.filter(isResizableColumn).map(tableColumnId));
  const ids = visibleIds.filter((id) => resizable.has(id));

  const partners: Record<string, string> = {};
  for (let index = 0; index < ids.length - 1; index++) {
    partners[ids[index]!] = ids[index + 1]!;
  }
  return partners;
}

/** TableResizeSide is the width of a column on one side of a dragged border and how narrow it may get. */
export type TableResizeSide = {
  width: number;
  minSize: number;
};

/**
 * tableResizeDelta limits how far a border moves, so neither column gets narrower than its minimum.
 * A positive delta widens the left column.
 */
export function tableResizeDelta(
  delta: number,
  left: TableResizeSide,
  right: TableResizeSide,
): number {
  const shrinkLeft = Math.max(0, left.width - left.minSize);
  const shrinkRight = Math.max(0, right.width - right.minSize);
  return Math.min(Math.max(delta, -shrinkLeft), shrinkRight);
}

/**
 * tableRenderedSizing returns the sizing with the rendered width of every visible resizable column.
 * Before a border moves, the stored widths then match what the viewer sees, also where the browser
 * stretched or shrank them. visibleIds and renderedWidths list the visible columns in the order of the table.
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

/** moveItem returns a copy of a list with one item moved from one index to another. */
export function moveItem<T>(items: T[], from: number, to: number): T[] {
  const result = [...items];
  const [moved] = result.splice(from, 1);
  if (moved === undefined) return result;
  result.splice(to, 0, moved);
  return result;
}
