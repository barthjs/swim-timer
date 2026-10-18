<script setup lang="ts" generic="T">
import { computed, nextTick, onBeforeUnmount, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import Sortable from 'sortablejs';
import type { TableColumn } from '@nuxt/ui';

import {
  emptyTableColumnState,
  moveItem,
  tableColumnOptions,
  type TableColumnState,
} from '@/lib/table/columns';

// The column menu of a table: shows and hides the configurable columns,
// moves them by dragging their handle, and resets the order, visibility, and widths.
// Only the handle starts a drag, so a click on a checkbox always toggles it.
const props = defineProps<{
  columns: TableColumn<T>[];
}>();

const columnState = defineModel<TableColumnState>('columnState', {
  required: true,
});

const { t } = useI18n();

const options = computed(() => tableColumnOptions(props.columns, columnState.value.order));

const visibleCount = computed(
  () => options.value.filter((option) => !columnState.value.hidden.includes(option.id)).length,
);

/** isVisible reports whether a column is shown. */
function isVisible(id: string): boolean {
  return !columnState.value.hidden.includes(id);
}

/** setVisible shows or hides a column. */
function setVisible(id: string, visible: boolean) {
  const hidden = columnState.value.hidden.filter((hiddenId) => hiddenId !== id);
  if (!visible) hidden.push(id);
  columnState.value = {
    ...columnState.value,
    hidden,
  };
}

/** reset restores the order, visibility, and widths the page defines. */
function reset() {
  columnState.value = emptyTableColumnState();
}

const list = useTemplateRef<HTMLElement>('list');
let sortable: Sortable | null = null;

/** initSortable makes the list sortable once the popover rendered it. */
function initSortable() {
  if (!list.value || sortable) return;
  sortable = new Sortable(list.value, {
    handle: '[data-drag-handle]',
    animation: 150,
    forceFallback: true,
    delay: 150,
    delayOnTouchOnly: true,
    ghostClass: 'opacity-30',
    onEnd: (event) => {
      const { oldIndex, newIndex, item, from } = event;
      if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) return;
      // Sortable moved the element itself. It goes back to its place, so only Vue changes the DOM
      // and renders the new order from the state.
      from.removeChild(item);
      from.insertBefore(item, from.children[oldIndex] ?? null);
      columnState.value = {
        ...columnState.value,
        order: moveItem(
          options.value.map((option) => option.id),
          oldIndex,
          newIndex,
        ),
      };
    },
  });
}

/** destroySortable removes the drag handling when the popover closes. */
function destroySortable() {
  sortable?.destroy();
  sortable = null;
}

/** onOpenChange sets up the drag handling while the popover is open. */
function onOpenChange(open: boolean) {
  if (open) {
    void nextTick(initSortable);
    return;
  }
  destroySortable();
}

onBeforeUnmount(destroySortable);
</script>

<template>
  <UPopover :content="{ align: 'end' }" @update:open="onOpenChange">
    <UTooltip :text="t('table.columnMenu.button')">
      <UButton
        icon="i-lucide-columns-3"
        color="neutral"
        variant="outline"
        :aria-label="t('table.columnMenu.button')"
      />
    </UTooltip>

    <template #content>
      <div class="w-64 p-2">
        <p class="text-muted px-2 pb-1 text-xs font-medium">{{ t('table.columnMenu.title') }}</p>

        <div ref="list" class="flex flex-col">
          <div
            v-for="option in options"
            :key="option.id"
            class="hover:bg-elevated flex items-center gap-2 rounded-md px-2 py-1.5"
          >
            <UIcon
              data-drag-handle
              name="i-lucide-grip-vertical"
              class="text-muted size-4 shrink-0 cursor-grab"
            />
            <UCheckbox
              :model-value="isVisible(option.id)"
              :label="option.label"
              :disabled="isVisible(option.id) && visibleCount === 1"
              class="min-w-0 flex-1"
              @update:model-value="
                (value: boolean | 'indeterminate') => setVisible(option.id, value === true)
              "
            />
          </div>
        </div>

        <USeparator class="my-1.5" />

        <UButton
          :label="t('table.columnMenu.reset')"
          icon="i-lucide-rotate-ccw"
          color="neutral"
          variant="ghost"
          size="xs"
          block
          @click="reset"
        />
      </div>
    </template>
  </UPopover>
</template>
