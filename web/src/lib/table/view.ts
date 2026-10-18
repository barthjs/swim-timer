/** The sorting of a table: the column and the direction. Only one column sorts at a time. */
export type TableSort = {
  id: string;
  desc: boolean;
};

export const PAGE_SIZES = [10, 25, 50, 100] as const;
export const DEFAULT_PAGE_SIZE = 25;

/**
 * parseSort reads a sort from the URL, written as the column id with a leading "-" for descending.
 */
export function parseSort(value: unknown, allowed: readonly string[]): TableSort | null {
  if (typeof value !== 'string' || !value) return null;

  const desc = value.startsWith('-');
  const id = desc ? value.slice(1) : value;
  return allowed.includes(id)
    ? {
        id,
        desc,
      }
    : null;
}

/** formatSort writes a sort for the URL, see parseSort. */
export function formatSort(sort: TableSort | null): string | undefined {
  if (!sort) return undefined;
  return sort.desc ? `-${sort.id}` : sort.id;
}

/** parsePageSize reads a page size from the URL, falling back to the default for unknown values. */
export function parsePageSize(value: unknown): number {
  const size = Number(value);
  return (PAGE_SIZES as readonly number[]).includes(size) ? size : DEFAULT_PAGE_SIZE;
}

/** parsePage reads a page number counted from 1, falling back to the first page. */
export function parsePage(value: unknown): number {
  const page = Number(value);
  return Number.isInteger(page) && page > 0 ? page : 1;
}

/**
 * compareValues orders numbers by value and everything else as text, with numbers inside text compared naturally.
 */
function compareValues(a: unknown, b: unknown): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b;
  return String(a ?? '').localeCompare(String(b ?? ''), undefined, {
    numeric: true,
    sensitivity: 'base',
  });
}

/**
 * sortRows returns the rows sorted by the value of the sorted column,
 * keeping the given order for equal values and without a sort.
 */
export function sortRows<T>(
  rows: readonly T[],
  sort: TableSort | null,
  value: (row: T, id: string) => unknown,
): T[] {
  if (!sort) return [...rows];

  const direction = sort.desc ? -1 : 1;
  return rows
    .map((row, index) => ({
      row,
      index,
    }))
    .sort(
      (a, b) =>
        direction * compareValues(value(a.row, sort.id), value(b.row, sort.id)) ||
        a.index - b.index,
    )
    .map((item) => item.row);
}

/** pageCount returns the number of pages, at least 1 so an empty table still has a page. */
export function pageCount(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

/**
 * pageRows returns the rows of a page counted from 1. A page beyond the last returns the last page.
 */
export function pageRows<T>(rows: readonly T[], page: number, pageSize: number): T[] {
  const last = pageCount(rows.length, pageSize);
  const start = (Math.min(page, last) - 1) * pageSize;
  return rows.slice(start, start + pageSize);
}

/**
 * pageOf returns the page counted from 1 that holds the first row matching the predicate, or null.
 */
export function pageOf<T>(
  rows: readonly T[],
  pageSize: number,
  predicate: (row: T) => boolean,
): number | null {
  const index = rows.findIndex(predicate);
  return index < 0 ? null : Math.floor(index / pageSize) + 1;
}
