/**
 * tableRowClass returns the classes of a table row, which stripe every second row and shade the row under the pointer.
 * The row is a named group, so a pinned cell with its own background can follow the hover of its row.
 */
export function tableRowClass(row: { index: number }): string {
  const hover = 'group/row hover:bg-table-hover';
  return row.index % 2 === 1 ? `${hover} bg-table-stripe` : hover;
}
