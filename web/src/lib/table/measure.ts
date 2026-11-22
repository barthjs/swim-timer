/** Returns the size of 1rem in pixels. */
export function rootFontSize(): number {
  return Number.parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
}

/**
 * Returns the width in pixels every column of a table needs to show its header
 * and every rendered cell on one line, in the order of the header cells.
 * It copies the rows into a hidden table with an automatic layout,
 * so it measures whatever the cells render, from plain text to badges and avatars.
 */
export function measureTableColumns(table: HTMLTableElement): number[] {
  const count = table.querySelector('thead > tr')?.children.length ?? 0;
  return measureColumns(
    table,
    Array.from({ length: count }, (_, index) => index),
  );
}

/** Returns the width in pixels the column of a header cell needs, as measureTableColumns does. */
export function measureColumnWidth(th: HTMLTableCellElement): number {
  const table = th.closest('table');
  const width = table ? measureColumns(table, [th.cellIndex])[0] : undefined;
  return width ?? Math.ceil(th.getBoundingClientRect().width);
}

/** Returns the widths of the columns at the indexes, copying only their cells. */
function measureColumns(table: HTMLTableElement, indexes: number[]): number[] {
  const header = table.querySelector<HTMLTableRowElement>('thead > tr');
  if (!header) return [];

  // A row with a cell spanning several columns, such as the empty state, belongs to no column
  const count = header.children.length;
  const rows = [
    header,
    ...Array.from(table.querySelectorAll<HTMLTableRowElement>('tbody > tr')).filter(
      (row) =>
        row.children.length === count &&
        Array.from(row.children).every(
          (cell) => cell instanceof HTMLTableCellElement && cell.colSpan === 1,
        ),
    ),
  ];

  // The classes of the real table are left out on purpose:
  // its fixed layout and full width would stretch the copy to the width of the table.
  const measureTable = document.createElement('table');
  measureTable.style.position = 'absolute';
  measureTable.style.visibility = 'hidden';
  measureTable.style.left = '-9999px';
  measureTable.style.top = '-9999px';
  measureTable.style.tableLayout = 'auto';
  const body = document.createElement('tbody');
  measureTable.append(body);

  const clones = rows.map((row) => {
    const clone = document.createElement('tr');
    for (const index of indexes) {
      const cell = row.children[index];
      if (!cell) continue;
      const cellClone = cell.cloneNode(true);
      if (!(cellClone instanceof HTMLTableCellElement)) continue;
      cellClone.style.width = '';
      cellClone.style.minWidth = '';
      cellClone.style.maxWidth = 'none';
      clone.append(cellClone);
    }
    body.append(clone);
    return clone;
  });

  document.body.append(measureTable);
  const widths = indexes.map((_, position) =>
    Math.ceil(
      Math.max(...clones.map((row) => row.children[position]?.getBoundingClientRect().width ?? 0)),
    ),
  );
  measureTable.remove();
  return widths;
}
