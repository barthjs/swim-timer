<script setup lang="ts" generic="T">
import {
  computed,
  h,
  onMounted,
  onScopeDispose,
  ref,
  resolveComponent,
  useTemplateRef,
  watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useElementSize } from '@vueuse/core';
import type { TableColumn, TableRow } from '@nuxt/ui';
import type {
  ExpandedState,
  HeaderContext,
  RowSelectionState,
  SortingState,
} from '@tanstack/table-core';

import TableColumnHeader from '@/components/table/TableColumnHeader.vue';
import TablePagination from '@/components/table/TablePagination.vue';
import TableSkeletonRows from '@/components/table/TableSkeletonRows.vue';
import {
  emptyTableColumnState,
  isConfigurableColumn,
  isNumericColumn,
  isResizableColumn,
  tableColumnId,
  tableColumnOrder,
  tableColumnVisibility,
  tableColumnWidth,
  tableMeasuredSizing,
  tableMinWidth,
  tableRenderedSizing,
  tableResizableIds,
  tableResize,
  type TableResizeColumn,
  tableResizePartners,
  tableSortableColumn,
  TABLE_ACTIONS_COLUMN,
  TABLE_COLUMN_MAX_REM,
  TABLE_COLUMN_MIN_REM,
  type TableColumnState,
  type TableSorting,
} from '@/lib/table/columns';
import { measureColumnWidth, measureTableColumns, rootFontSize } from '@/lib/table/measure';
import { tableRowClass } from '@/lib/table/rows';
import { DEFAULT_PAGE_SIZE } from '@/lib/table/view';

// A table that grows into the free height of a flex column. In a container of bounded height the rows scroll inside it,
// while the header and the pagination stay in view. In a page that scrolls as a whole, the pagination sticks
// to the bottom of the window. It applies the column layout of the viewer (useTableColumnState),
// renders the header of every column with a text header as TableColumnHeader, and shows placeholder rows while loading.
// A column can be sorted with `enableSorting: true`. Once rows arrive, every column gets the width its content needs,
// except a column with `meta.grow`, which takes the remaining width. A column with `meta.numeric` aligns at the end.
// The handle at the right edge of a column moves the border to the next resizable column,
// so the table keeps its width and no column leaves the view. selectable adds a first column of checkboxes
// for the rowSelection model, and the expanded model opens rows for the #expanded slot. The slots reach UTable.
const props = defineProps<{
  columns: TableColumn<T>[];
  data: readonly T[];
  loading?: boolean;
  // Total number of rows, without it the table shows no pagination
  total?: number;
  getRowId?: (row: T) => string;
  selectable?: boolean;
  // Columns that stay hidden until the viewer shows them, the same list TableColumnMenu gets
  hiddenByDefault?: string[];
  // Makes the rows clickable, as the select event of UTable
  onSelect?: (event: Event, row: TableRow<T>) => void;
}>();

const columnState = defineModel<TableColumnState>('columnState', {
  default: emptyTableColumnState,
});
const sorting = defineModel<TableSorting>('sorting', {
  default: () => [],
});
const rowSelection = defineModel<Record<string, boolean>>('rowSelection', {
  default: () => ({}),
});
const expanded = defineModel<Record<string, boolean>>('expanded', {
  default: () => ({}),
});
const page = defineModel<number>('page', {
  default: 1,
});
const limit = defineModel<number>('limit', {
  default: DEFAULT_PAGE_SIZE,
});

const { t, locale } = useI18n();

// The cells render through h(), where components of a template do not resolve on their own
const UCheckbox = resolveComponent('UCheckbox');

/** Builds the checkbox column that selects rows, whose clicks do not reach the row. */
function selectColumn(): TableColumn<T> {
  return {
    id: 'select',
    header: ({ table }) =>
      h(UCheckbox, {
        modelValue: table.getIsSomePageRowsSelected()
          ? 'indeterminate'
          : table.getIsAllPageRowsSelected(),
        'onUpdate:modelValue': (value: unknown) => table.toggleAllPageRowsSelected(value === true),
        'aria-label': t('table.selectAll'),
      }),
    cell: ({ row }) =>
      h(
        'div',
        {
          class: 'flex',
          onClick: (event: Event) => event.stopPropagation(),
        },
        [
          h(UCheckbox, {
            modelValue: row.getIsSelected(),
            'onUpdate:modelValue': (value: unknown) => row.toggleSelected(value === true),
            'aria-label': t('table.selectRow'),
          }),
        ],
      ),
  };
}

// The columns of the page, after the selection column if the rows can be selected
const pageColumns = computed(() =>
  props.selectable ? [selectColumn(), ...props.columns] : props.columns,
);

const order = computed(() => tableColumnOrder(pageColumns.value, columnState.value.order));
const visibility = computed(() =>
  tableColumnVisibility(pageColumns.value, columnState.value, props.hiddenByDefault),
);
const visibleIds = computed(() => order.value.filter((id) => visibility.value[id] !== false));
const partners = computed(() => tableResizePartners(pageColumns.value, visibleIds.value));
// The widths the content of the columns needs, measured again only when other columns show or the language changes,
// so the columns keep their widths while the viewer pages, sorts, or filters.
const measured = ref<Record<string, number>>({});
const measuredKey = ref('');
// The header waits for the first rows, so it does not jump once their widths are measured
const rowsShown = ref(false);
const headerHidden = computed(() => !rowsShown.value && (props.loading || props.data.length > 0));

const minWidthStyle = computed(() => {
  const width = tableMinWidth(
    pageColumns.value,
    columnState.value,
    measured.value,
    props.hiddenByDefault,
  );
  // While a border moves, the width the table grows to overrides the stored one
  return {
    '--table-min-width': `var(--table-drag-min-width, calc(${width.pixels}px + ${width.rem}rem))`,
  };
});

const root = useTemplateRef<HTMLElement>('root');

// The visible width of the table, which keeps the empty state in view while the table is wider than its container
const { width: viewportWidth } = useElementSize(root);

/**
 * Measures the visible columns once the rows are rendered, or only their headers for a table without rows,
 * so no header gets cut off. The first rows of an empty table measure it again.
 */
function measureColumns() {
  if (props.loading) return;
  const hasRows = props.data.length > 0;
  if (hasRows) rowsShown.value = true;
  const key = `${locale.value}|${visibleIds.value.join(',')}|${hasRows}`;
  if (key === measuredKey.value) return;

  const table = root.value?.querySelector('table');
  if (!table) return;
  const widths = measureTableColumns(table);
  // A row that does not match the columns leaves the widths to the next rows
  if (widths.length !== visibleIds.value.length) return;
  const rem = rootFontSize();
  measured.value = tableMeasuredSizing(pageColumns.value, visibleIds.value, widths, {
    min: TABLE_COLUMN_MIN_REM * rem,
    max: TABLE_COLUMN_MAX_REM * rem,
  });
  measuredKey.value = key;
}
onMounted(() => {
  measureColumns();
  // A measure taken before the web fonts loaded is a few pixels short, which cuts the headers off
  void document.fonts.ready.then(() => {
    measuredKey.value = '';
    measureColumns();
  });
});
watch([() => props.data, () => props.loading, visibleIds, locale], measureColumns, {
  flush: 'post',
});

// The border being dragged after a column, where the pointer started, and, once it moved,
// what it started from and the widths it reached. The id only marks the dragged handle.
type ResizeDrag = {
  id: string;
  th: HTMLTableCellElement;
  startX: number;
  start?: ResizeStart;
  widths?: Record<string, number>;
};

// The rendered widths a move of a border starts from, the resizable columns with the index of the one before the border,
// and the width of the table with the part of it beyond its container.
type ResizeStart = {
  sizing: Record<string, number>;
  columns: TableResizeColumn[];
  index: number;
  tableWidth: number;
  overflow: number;
};

const resizing = ref<{ id: string } | null>(null);
let drag: ResizeDrag | null = null;

// A drag starts to change widths only after the pointer moved this far, so a double click stays one
const DRAG_THRESHOLD = 3;

/** Returns a column of the page by its id. */
function columnById(id: string): TableColumn<T> | undefined {
  return pageColumns.value.find((column) => tableColumnId(column) === id);
}

/** Returns how narrow a column may get. */
function minSizeOf(id: string): number {
  return columnById(id)?.minSize ?? TABLE_COLUMN_MIN_REM * rootFontSize();
}

/** Returns the custom property that holds the width of a column while a border moves. */
function cssVariable(id: string): string {
  return `--table-column-${id.replace(/[^\w-]/g, '-')}`;
}

/**
 * Returns the sizing with the width every visible resizable column has on screen,
 * read from the header row of the given cell.
 */
function renderedSizing(th: HTMLTableCellElement): Record<string, number> {
  const cells = Array.from(th.parentElement?.children ?? []);
  return tableRenderedSizing(
    pageColumns.value,
    columnState.value.sizing,
    visibleIds.value,
    cells.map((cell) => cell.getBoundingClientRect().width),
  );
}

/** Stores the widths of the columns. */
function setSizing(sizing: Record<string, number>) {
  columnState.value = {
    ...columnState.value,
    sizing,
  };
}

/** Returns the horizontal position of a mouse or touch event. */
function pointerX(event: MouseEvent | TouchEvent): number {
  if ('touches' in event) return event.touches[0]?.clientX ?? 0;
  return event.clientX;
}

/** Starts dragging the border at the right edge of a column. */
function startResize(event: MouseEvent | TouchEvent, id: string) {
  const th = event.currentTarget instanceof Element ? event.currentTarget.closest('th') : null;
  if (!th || !partners.value[id]) return;
  event.preventDefault();

  drag = {
    id,
    th,
    startX: pointerX(event),
  };
  window.addEventListener('mousemove', onResizeMove);
  window.addEventListener('touchmove', onResizeMove);
  window.addEventListener('mouseup', stopResize);
  window.addEventListener('touchend', stopResize);
}
/**
 * Moves the border with the pointer, see tableResize.
 * The widths go to custom properties of the table instead of the column state,
 * so the rows are not rendered again on every move.
 */
function onResizeMove(event: MouseEvent | TouchEvent) {
  if (!drag) return;
  const delta = pointerX(event) - drag.startX;

  if (!drag.start) {
    if (Math.abs(delta) < DRAG_THRESHOLD) return;
    // The first move fixes every width as it is on screen, so only the columns the border pushes change
    drag.start = resizeStart(drag.th, drag.id);
    setSizing(drag.start.sizing);
    resizing.value = {
      id: drag.id,
    };
  }
  const widths = tableResize(delta, drag.start.columns, drag.start.index, drag.start.overflow);
  showWidths(drag.start, widths, drag.widths);
  drag.widths = widths;
}
/** Ends a drag and stores the widths it reached. */
function stopResize() {
  window.removeEventListener('mousemove', onResizeMove);
  window.removeEventListener('touchmove', onResizeMove);
  window.removeEventListener('mouseup', stopResize);
  window.removeEventListener('touchend', stopResize);

  const finished = drag;
  drag = null;
  resizing.value = null;
  if (!finished?.widths) return;
  setSizing({
    ...columnState.value.sizing,
    ...roundedWidths(finished.widths),
  });
  for (const id of Object.keys(finished.widths)) {
    root.value?.style.removeProperty(cssVariable(id));
  }
  root.value?.style.removeProperty('--table-drag-min-width');
}
onScopeDispose(stopResize);

/** Fits a column to its header and its visible cells, moving the border like a drag does. */
function autoFit(event: MouseEvent, id: string) {
  const th = event.currentTarget instanceof Element ? event.currentTarget.closest('th') : null;
  if (!th || !partners.value[id]) return;

  const start = resizeStart(th, id);
  const width = start.columns[start.index]?.width ?? 0;
  const widths = tableResize(
    measureColumnWidth(th) - width,
    start.columns,
    start.index,
    start.overflow,
  );
  setSizing({
    ...start.sizing,
    ...roundedWidths(widths),
  });
}
/** Reads what a move of the border after a column starts from. */
function resizeStart(th: HTMLTableCellElement, id: string): ResizeStart {
  const sizing = renderedSizing(th);
  const ids = tableResizableIds(pageColumns.value, visibleIds.value);
  const tableWidth = th.parentElement?.getBoundingClientRect().width ?? 0;
  const containerWidth = th.closest('table')?.parentElement?.clientWidth ?? tableWidth;
  return {
    sizing,
    columns: ids.map((columnId) => ({
      id: columnId,
      width: sizing[columnId] ?? 0,
      minSize: minSizeOf(columnId),
    })),
    index: ids.indexOf(id),
    tableWidth,
    overflow: Math.max(0, tableWidth - containerWidth),
  };
}

/**
 * Shows the widths a moving border reached and the width the table grows to.
 * A column the previous move changed but this one does not gets its width from the start again.
 */
function showWidths(
  start: ResizeStart,
  widths: Record<string, number>,
  previous: Record<string, number> | undefined,
) {
  for (const id of Object.keys(previous ?? {})) {
    if (!(id in widths)) root.value?.style.removeProperty(cssVariable(id));
  }
  let growth = 0;
  for (const column of start.columns) {
    const width = widths[column.id];
    if (width === undefined) continue;
    growth += width - column.width;
    root.value?.style.setProperty(cssVariable(column.id), `${width}px`);
  }
  root.value?.style.setProperty('--table-drag-min-width', `${start.tableWidth + growth}px`);
}

/** Returns the widths rounded to whole pixels for the column state. */
function roundedWidths(widths: Record<string, number>): Record<string, number> {
  return Object.fromEntries(Object.entries(widths).map(([id, width]) => [id, Math.round(width)]));
}

/**
 * Returns the inline width of a column, empty for a column that shares the remaining width.
 * While a border moves, the custom property of the column overrides the stored width.
 */
function widthStyle(column: TableColumn<T>): Record<string, string> {
  const width = tableColumnWidth(column, columnState.value.sizing, measured.value);
  if (width === undefined) return {};
  return {
    width: `var(${cssVariable(tableColumnId(column))}, ${width}px)`,
  };
}

/**
 * Prepares a column of the page for UTable: sorting only where the page enables it,
 * the header with sorting and resizing, and the width from the column state.
 */
function tableColumn(column: TableColumn<T>): TableColumn<T> {
  const configurable = isConfigurableColumn(column);
  const label = configurable ? String(column.header) : '';
  const id = tableColumnId(column);

  const prepared = {
    ...tableSortableColumn(column),
    enableSorting: column.enableSorting ?? false,
    enableResizing: isResizableColumn(column),
    header: configurable
      ? ({ column: tableColumnApi }: HeaderContext<T, unknown>) =>
          h(TableColumnHeader, {
            label,
            sorted: tableColumnApi.getIsSorted(),
            sortable: tableColumnApi.getCanSort(),
            end: isNumericColumn(column),
            resizable: partners.value[id] !== undefined,
            resizing: resizing.value?.id === id,
            onSort: () => tableColumnApi.toggleSorting(tableColumnApi.getIsSorted() === 'asc'),
            onResizeStart: (event: MouseEvent | TouchEvent) => startResize(event, id),
            onAutoFit: (event: MouseEvent) => autoFit(event, id),
          })
      : column.header,
    meta: {
      ...column.meta,
      ...(isNumericColumn(column) ? numericClass : {}),
      ...(id === TABLE_ACTIONS_COLUMN ? pinnedClass : {}),
      style: {
        th: () => widthStyle(column),
        td: () => widthStyle(column),
      },
    },
  };
  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the column keeps its kind and gains header and meta
  return prepared as TableColumn<T>;
}
const numericClass = {
  class: {
    th: 'text-end',
    td: 'text-end',
  },
};

// The actions stay at the right edge while a wide table scrolls. A pinned cell needs an opaque background,
// so the stripe and the hover of its row are laid over it again.
const pinnedClass = {
  class: {
    th: 'sticky right-0 z-20 border-l border-default bg-default',
    td: (cell: { row: { index: number } }) =>
      `sticky right-0 z-10 border-l border-default bg-default before:pointer-events-none before:absolute before:inset-0 group-hover/row:before:bg-table-hover${
        cell.row.index % 2 === 1 ? ' before:bg-table-stripe' : ''
      }`,
  },
};

const tableColumns = computed(() => pageColumns.value.map(tableColumn));

const isPaginated = computed(() => props.total !== undefined);

/** Takes the sorting of UTable, which reports a table without sorting as undefined. */
function updateSorting(value: SortingState | undefined): void {
  sorting.value = value ?? [];
}

/** Takes the selected rows of UTable, which reports a table without a selection as undefined. */
function updateRowSelection(value: RowSelectionState | undefined): void {
  rowSelection.value = value ?? {};
}

/** Takes the expanded rows of UTable, which only reports true when every row expands at once. */
function updateExpanded(value: ExpandedState | undefined): void {
  if (value === true) return;
  expanded.value = value ?? {};
}
</script>

<template>
  <div
    ref="root"
    class="border-default bg-default flex flex-1 flex-col overflow-clip rounded-lg border"
    :class="resizing ? 'cursor-col-resize select-none' : ''"
    :style="{ '--table-viewport-width': `${viewportWidth}px` }"
  >
    <UTable
      :row-selection="rowSelection"
      :sorting="sorting"
      :expanded="expanded"
      :data="data"
      :columns="tableColumns"
      :column-order="order"
      :column-visibility="visibility"
      :loading="loading"
      :get-row-id="getRowId"
      :on-select="onSelect"
      :sorting-options="{ manualSorting: true }"
      :meta="{ class: { tr: tableRowClass } }"
      sticky="header"
      :style="minWidthStyle"
      :ui="{
        base: 'w-full min-w-(--table-min-width) table-fixed',
        thead: `bg-default backdrop-blur-none ${headerHidden ? 'invisible' : ''}`,
        th: 'relative',
        td: 'overflow-hidden text-ellipsis',
        loading: 'p-0 text-start',
        empty: 'p-0',
      }"
      class="min-h-0 flex-1"
      @update:sorting="updateSorting"
      @update:row-selection="updateRowSelection"
      @update:expanded="updateExpanded"
    >
      <template #loading>
        <TableSkeletonRows :columns="Math.min(order.length, 6)" :rows="limit" />
      </template>

      <template v-for="(_, name) in $slots" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps ?? {}" />
      </template>
    </UTable>

    <div v-if="isPaginated" class="border-default bg-default shrink-0 border-t px-3 py-2">
      <TablePagination v-model:page="page" v-model:limit="limit" :total="total ?? 0" />
    </div>
  </div>
</template>
