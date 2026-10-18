/**
 * measureColumnWidth returns the width in pixels a table column needs to show its header and every rendered cell
 * on one line. It copies the cells into a hidden table with an automatic layout,
 * so it measures whatever the cells render, from plain text to badges and avatars.
 */
export function measureColumnWidth(th: HTMLTableCellElement): number {
  const table = th.closest('table');
  if (!table) return Math.ceil(th.getBoundingClientRect().width);

  // UTable renders no column id on the cells, so the cells of the column are found by their index
  const index = th.cellIndex;
  const rows = Array.from(table.querySelectorAll<HTMLTableRowElement>('tbody > tr'));
  const cells: HTMLTableCellElement[] = [th];
  for (const row of rows) {
    const cell = row.children[index];
    // A row with a single cell spanning the table, such as the empty state, belongs to no column
    if (cell instanceof HTMLTableCellElement && cell.colSpan === 1) cells.push(cell);
  }

  // The classes of the real table are left out on purpose: its fixed layout and full width
  // would stretch the copy to the width of the table.
  const measureTable = document.createElement('table');
  measureTable.style.position = 'absolute';
  measureTable.style.visibility = 'hidden';
  measureTable.style.left = '-9999px';
  measureTable.style.top = '-9999px';
  measureTable.style.tableLayout = 'auto';
  const body = document.createElement('tbody');
  measureTable.append(body);

  const clones = cells.map((cell) => {
    const clone = cell.cloneNode(true) as HTMLTableCellElement;
    clone.style.width = '';
    clone.style.minWidth = '';
    clone.style.maxWidth = 'none';
    const row = document.createElement('tr');
    row.append(clone);
    body.append(row);
    return clone;
  });

  document.body.append(measureTable);
  const width = Math.max(...clones.map((clone) => clone.getBoundingClientRect().width));
  measureTable.remove();
  return Math.ceil(width);
}
