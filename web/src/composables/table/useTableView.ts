import { computed, reactive, toValue, watch, type MaybeRefOrGetter, type WatchSource } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import {
  DEFAULT_PAGE_SIZE,
  formatSort,
  pageCount,
  pageOf,
  pageRows,
  parsePage,
  parsePageSize,
  parseSort,
  sortRows,
  type TableSort,
} from '@/lib/table/view';
import type { TableSorting } from '@/lib/table/columns';

export type TableViewOptions<T> = {
  // The ids of the columns that can be sorted.
  sortable: readonly string[];
  // The value a row is sorted by for a column id.
  value: (row: T, id: string) => unknown;
  // A change of these, for example the filters, returns to the first page.
  resetOn?: WatchSource;
  // Whether the view is kept in the URL, true by default. A table in a dialog keeps it in memory,
  // so it leaves the view of the table on the page below alone.
  inUrl?: boolean;
};

/**
 * useTableView sorts and pages the rows of a table the client holds as a whole and keeps the sorting, the page,
 * and the page size in the URL, so a view can be shared as a link and survives a reload. Sorting and paging happen
 * here instead of in the table component, so a caller can find the page of a row and jump to it.
 */
export function useTableView<T>(
  rows: MaybeRefOrGetter<readonly T[]>,
  options: TableViewOptions<T>,
) {
  const route = useRoute();
  const router = useRouter();
  const inUrl = options.inUrl ?? true;
  const memory = reactive<Record<string, string | undefined>>({});

  /** readQuery returns a parameter of the view from the URL, or from memory for a view outside the URL. */
  function readQuery(key: string): unknown {
    return inUrl ? route.query[key] : memory[key];
  }

  /** setQuery replaces parameters of the view, in the URL without adding a history entry. */
  function setQuery(values: Record<string, string | undefined>): void {
    if (!inUrl) {
      Object.assign(memory, values);
      return;
    }
    void router.replace({
      query: {
        ...route.query,
        ...values,
      },
    });
  }

  const sort = computed<TableSort | null>({
    get: () => parseSort(readQuery('sort'), options.sortable),
    set: (value) =>
      setQuery({
        sort: formatSort(value),
        page: undefined,
      }),
  });

  // The sort as the sorting model of DataTable, which holds at most one sorted column here
  const sorting = computed<TableSorting>({
    get: () => (sort.value ? [sort.value] : []),
    set: (value) => {
      sort.value = value[0] ?? null;
    },
  });

  const pageSize = computed<number>({
    get: () => parsePageSize(readQuery('perPage')),
    set: (value) =>
      setQuery({
        perPage: value === DEFAULT_PAGE_SIZE ? undefined : String(value),
        page: undefined,
      }),
  });

  const sorted = computed(() => sortRows(toValue(rows), sort.value, options.value));
  const total = computed(() => sorted.value.length);
  const pages = computed(() => pageCount(total.value, pageSize.value));

  const page = computed<number>({
    get: () => Math.min(parsePage(readQuery('page')), pages.value),
    set: (value) =>
      setQuery({
        page: value <= 1 ? undefined : String(value),
      }),
  });

  const visible = computed(() => pageRows(sorted.value, page.value, pageSize.value));

  if (options.resetOn) {
    watch(options.resetOn, () => (page.value = 1), {
      deep: true,
    });
  }

  /**
   * goTo switches to the page that holds the first row matching the predicate. It reports whether one does.
   */
  function goTo(predicate: (row: T) => boolean): boolean {
    const target = pageOf(sorted.value, pageSize.value, predicate);
    if (target === null) return false;
    page.value = target;
    return true;
  }

  return {
    sort,
    sorting,
    page,
    pageSize,
    total,
    visible,
    goTo,
  };
}
