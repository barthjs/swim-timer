import { useStoredState } from '@/composables/useStoredState';
import { emptyTableColumnState, parseTableColumnState } from '@/lib/table/columns';

/**
 * useTableColumnState holds the order, visibility, and widths a viewer gave the columns of a table,
 * stored in the browser under the id of the table. Without an id the layout lasts until the page is left.
 * DataTable applies the state and TableColumnMenu changes it.
 */
export function useTableColumnState(id?: string) {
  return useStoredState(
    id ? `table.${id}.columns` : undefined,
    emptyTableColumnState,
    parseTableColumnState,
  );
}
