<script setup lang="ts" generic="T">
import { computed, h, onScopeDispose, ref, resolveComponent, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn, TableRow } from '@nuxt/ui';

import TableColumnHeader from '@/components/table/TableColumnHeader.vue';
import TablePagination from '@/components/table/TablePagination.vue';
import TableSkeletonRows from '@/components/table/TableSkeletonRows.vue';
import {
  emptyTableColumnState,
  isConfigurableColumn,
  isResizableColumn,
  tableColumnId,
  tableColumnOrder,
  tableColumnVisibility,
  tableColumnWidth,
  tableMinWidth,
  tableRenderedSizing,
  tableResizeDelta,
  tableResizePartners,
  TABLE_ACTIONS_COLUMN,
  TABLE_COLUMN_MIN_SIZE,
  type TableColumnState,
  type TableSorting,
} from '@/lib/table/columns';
import { measureColumnWidth } from '@/lib/table/measure';
import { tableRowClass } from '@/lib/table/rows';

// A table that grows into the free height of a flex column. In a container of bounded height the rows scroll
// inside it, while the header and the pagination stay in view. In a page that scrolls as a whole,
// the pagination sticks to the bottom of the window. It applies the column layout of the viewer (useTableColumnState),
// renders the header of every column with a text header as TableColumnHeader, and shows placeholder rows while loading.
// A column can be sorted with `enableSorting: true` and keeps the width of its `size`,
// without one it shares the remaining width. The handle at the right edge of a column moves the border
// to the next resizable column, so the table keeps its width and no column leaves the view.
// selectable adds a first column of checkboxes for the rowSelection model, and the expanded model opens rows
// for the #expanded slot. Slots, such as the cells, reach UTable.
const props = defineProps<{
  columns: TableColumn<T>[];
  data: T[];
  loading?: boolean;
  // Total number of rows, without it the table shows no pagination
  total?: number;
  getRowId?: (row: T) => string;
  selectable?: boolean;
  // Makes the rows clickable, as the select event of UTable
  onSelect?: (event: Event, row: TableRow<T>) => void;
}>();

const columnState = defineModel<TableColumnState>('columnState', {
  default: emptyTableColumnState,
});
const sorting = defineModel<TableSorting>('sorting', {
  default: () => [],
});
const rowSelection = defineModel<Record<string, boolean>>('rowSelection');
const expanded = defineModel<Record<string, boolean>>('expanded');
const page = defineModel<number>('page', {
  default: 1,
});
const limit = defineModel<number>('limit', {
  default: 25,
});

const { t } = useI18n();

// The cells render through h(), where components of a template do not resolve on their own
const UCheckbox = resolveComponent('UCheckbox');

/**
 * selectColumn builds the checkbox column that selects rows. The header selects every row of the page.
 * A click on a checkbox does not reach the row, so a clickable row does not open as well.
 */
function selectColumn(): TableColumn<T> {
  return {
    id: 'select',
    size: 48,
    header: ({ table }) =>
      h(UCheckbox, {
        modelValue: table.getIsSomePageRowsSelected()
          ? 'indeterminate'
          : table.getIsAllPageRowsSelected(),
        'onUpdate:modelValue': (value: unknown) => table.toggleAllPageRowsSelected(!!value),
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
            'onUpdate:modelValue': (value: unknown) => row.toggleSelected(!!value),
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
  tableColumnVisibility(pageColumns.value, columnState.value.hidden),
);
const visibleIds = computed(() => order.value.filter((id) => visibility.value[id] !== false));
const partners = computed(() => tableResizePartners(pageColumns.value, visibleIds.value));
const minWidthStyle = computed(() => ({
  '--table-min-width': `${tableMinWidth(pageColumns.value, columnState.value)}px`,
}));

const root = useTemplateRef<HTMLElement>('root');

// The border being dragged: the columns on both sides, where the pointer started,
// and, once it moved, the widths the drag started from. The id only marks the dragged handle.
type ResizeDrag = {
  id: string;
  partnerId: string;
  th: HTMLTableCellElement;
  startX: number;
  left?: { width: number; minSize: number };
  right?: { width: number; minSize: number };
  delta: number;
};

const resizing = ref<{ id: string } | null>(null);
let drag: ResizeDrag | null = null;

// A drag starts to change widths only after the pointer moved this far, so a double click stays one
const DRAG_THRESHOLD = 3;

/** columnById returns a column of the page by its id. */
function columnById(id: string): TableColumn<T> | undefined {
  return pageColumns.value.find((column) => tableColumnId(column) === id);
}

/** minSizeOf returns how narrow a column may get. */
function minSizeOf(id: string): number {
  return columnById(id)?.minSize ?? TABLE_COLUMN_MIN_SIZE;
}

/** cssVariable returns the custom property that holds the width of a column while a border moves. */
function cssVariable(id: string): string {
  return `--table-column-${id.replace(/[^\w-]/g, '-')}`;
}

/**
 * renderedSizing returns the sizing with the width every visible resizable column has on screen,
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

/** setSizing stores the widths of the columns. */
function setSizing(sizing: Record<string, number>) {
  columnState.value = {
    ...columnState.value,
    sizing,
  };
}

/** pointerX returns the horizontal position of a mouse or touch event. */
function pointerX(event: MouseEvent | TouchEvent): number {
  if ('touches' in event) return event.touches[0]?.clientX ?? 0;
  return event.clientX;
}

/** startResize starts dragging the border at the right edge of a column. */
function startResize(event: MouseEvent | TouchEvent, id: string) {
  const th = (event.currentTarget as HTMLElement | null)?.closest('th');
  const partnerId = partners.value[id];
  if (!th || !partnerId) return;
  event.preventDefault();

  drag = {
    id,
    partnerId,
    th,
    startX: pointerX(event),
    delta: 0,
  };
  window.addEventListener('mousemove', onResizeMove);
  window.addEventListener('touchmove', onResizeMove);
  window.addEventListener('mouseup', stopResize);
  window.addEventListener('touchend', stopResize);
}

/**
 * onResizeMove moves the border with the pointer. The widths go to custom properties of the table
 * instead of the column state, so the rows are not rendered again on every move.
 */
function onResizeMove(event: MouseEvent | TouchEvent) {
  if (!drag) return;
  const delta = pointerX(event) - drag.startX;

  if (!drag.left || !drag.right) {
    if (Math.abs(delta) < DRAG_THRESHOLD) return;
    // The first move fixes every width as it is on screen, so only the two columns at the border change
    const sizing = renderedSizing(drag.th);
    setSizing(sizing);
    drag.left = {
      width: sizing[drag.id] ?? 0,
      minSize: minSizeOf(drag.id),
    };
    drag.right = {
      width: sizing[drag.partnerId] ?? 0,
      minSize: minSizeOf(drag.partnerId),
    };
    resizing.value = {
      id: drag.id,
    };
  }

  drag.delta = tableResizeDelta(delta, drag.left, drag.right);
  root.value?.style.setProperty(cssVariable(drag.id), `${drag.left.width + drag.delta}px`);
  root.value?.style.setProperty(cssVariable(drag.partnerId), `${drag.right.width - drag.delta}px`);
}

/** stopResize ends a drag and stores the widths it reached. */
function stopResize() {
  window.removeEventListener('mousemove', onResizeMove);
  window.removeEventListener('touchmove', onResizeMove);
  window.removeEventListener('mouseup', stopResize);
  window.removeEventListener('touchend', stopResize);

  const finished = drag;
  drag = null;
  resizing.value = null;
  if (!finished?.left || !finished.right) return;

  setSizing({
    ...columnState.value.sizing,
    [finished.id]: Math.round(finished.left.width + finished.delta),
    [finished.partnerId]: Math.round(finished.right.width - finished.delta),
  });
  root.value?.style.removeProperty(cssVariable(finished.id));
  root.value?.style.removeProperty(cssVariable(finished.partnerId));
}

onScopeDispose(stopResize);

/**
 * autoFit fits a column to its header and its visible cells. The next resizable column gives or takes the difference,
 * as far as its minimum allows.
 */
function autoFit(event: MouseEvent, id: string) {
  const th = (event.currentTarget as HTMLElement | null)?.closest('th');
  const partnerId = partners.value[id];
  if (!th || !partnerId) return;

  const sizing = renderedSizing(th);
  const left = {
    width: sizing[id] ?? 0,
    minSize: minSizeOf(id),
  };
  const right = {
    width: sizing[partnerId] ?? 0,
    minSize: minSizeOf(partnerId),
  };
  const delta = tableResizeDelta(measureColumnWidth(th) - left.width, left, right);
  setSizing({
    ...sizing,
    [id]: Math.round(left.width + delta),
    [partnerId]: Math.round(right.width - delta),
  });
}

/**
 * widthStyle returns the inline width of a column, empty for a column that shares the remaining width.
 * While a border moves, the custom property of the column overrides the stored width.
 */
function widthStyle(column: TableColumn<T>): Record<string, string> {
  const width = tableColumnWidth(column, columnState.value.sizing);
  if (width === undefined) return {};
  return {
    width: `var(${cssVariable(tableColumnId(column))}, ${width}px)`,
  };
}

/**
 * tableColumn prepares a column of the page for UTable: sorting only where the page enables it,
 * the header with sorting and resizing, and the width from the column state.
 */
function tableColumn(column: TableColumn<T>): TableColumn<T> {
  const configurable = isConfigurableColumn(column);
  const label = configurable ? String(column.header) : '';
  const id = tableColumnId(column);

  return {
    ...column,
    enableSorting: column.enableSorting ?? false,
    enableResizing: isResizableColumn(column),
    header: configurable
      ? ({ column: tableColumnApi }) =>
          h(TableColumnHeader, {
            label,
            sorted: tableColumnApi.getIsSorted(),
            sortable: tableColumnApi.getCanSort(),
            resizable: partners.value[id] !== undefined,
            resizing: resizing.value?.id === id,
            onSort: () => tableColumnApi.toggleSorting(tableColumnApi.getIsSorted() === 'asc'),
            onResizeStart: (event: MouseEvent | TouchEvent) => startResize(event, id),
            onAutoFit: (event: MouseEvent) => autoFit(event, id),
          })
      : column.header,
    meta: {
      ...column.meta,
      ...(id === TABLE_ACTIONS_COLUMN ? pinnedClass : {}),
      style: {
        th: () => widthStyle(column),
        td: () => widthStyle(column),
      },
    },
  } as TableColumn<T>;
}

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
</script>

<template>
  <div
    ref="root"
    class="border-default bg-default flex flex-1 flex-col overflow-clip rounded-lg border"
    :class="resizing ? 'cursor-col-resize select-none' : ''"
  >
    <UTable
      v-model:sorting="sorting"
      v-model:row-selection="rowSelection"
      v-model:expanded="expanded"
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
        thead: 'bg-default backdrop-blur-none',
        th: 'relative',
        td: 'overflow-hidden text-ellipsis',
        loading: 'p-0 text-start',
      }"
      class="min-h-0 flex-1"
    >
      <template #loading>
        <TableSkeletonRows :columns="Math.min(order.length, 6)" :rows="limit" />
      </template>

      <template v-for="(_, name) in $slots" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps ?? {}" />
      </template>
    </UTable>

    <!--
      overflow-clip on the root keeps the corners without a scroll container,
      so this footer can stick to the window.
    -->
    <div
      v-if="isPaginated"
      class="border-default bg-default sticky bottom-0 shrink-0 border-t px-3 py-2"
    >
      <TablePagination v-model:page="page" v-model:limit="limit" :total="total ?? 0" />
    </div>
  </div>
</template>
