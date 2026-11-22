import { useStoredState } from '@/composables/ui/useStoredState';
import { emptyTableColumnState, parseTableColumnState } from '@/lib/table/columns';

/** Holds the column layout a viewer gave a table, stored in the browser under the id of the table. */
export function useTableColumnState(id?: string) {
  return useStoredState(
    id ? `table.${id}.columns` : undefined,
    emptyTableColumnState,
    parseTableColumnState,
  );
}
