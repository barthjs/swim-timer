<script setup lang="ts">
import { useI18n } from 'vue-i18n';

// The header of a table column: the label, as a button that switches the order if the column can be sorted,
// and a handle at the right edge that resizes the column. A double click on the handle fits the column to its content.
// DataTable connects it to the table.
const props = defineProps<{
  label: string;
  sorted: false | 'asc' | 'desc';
  sortable: boolean;
  resizable: boolean;
  resizing: boolean;
}>();

const emit = defineEmits<{
  sort: [];
  resizeStart: [event: MouseEvent | TouchEvent];
  autoFit: [event: MouseEvent];
}>();

const { t } = useI18n();

/** sortIcon returns the icon that shows the current order. */
function sortIcon(): string {
  if (props.sorted === 'asc') return 'i-lucide-arrow-up-narrow-wide';
  if (props.sorted === 'desc') return 'i-lucide-arrow-down-wide-narrow';
  return 'i-lucide-arrow-up-down';
}
</script>

<template>
  <div class="flex min-w-0 items-center">
    <UButton
      v-if="sortable"
      :label="label"
      :trailing-icon="sortIcon()"
      color="neutral"
      variant="ghost"
      class="-mx-2.5 min-w-0"
      :ui="{ label: 'truncate' }"
      @click="emit('sort')"
    />
    <span v-else class="truncate">{{ label }}</span>

    <!-- The th must be positioned, DataTable gives it the class "relative" -->
    <div
      v-if="resizable"
      role="separator"
      aria-orientation="vertical"
      :aria-label="t('table.resizeColumn', { label })"
      class="group absolute inset-y-0 -right-1.5 z-1 flex w-3 cursor-col-resize touch-none justify-center select-none"
      @mousedown="emit('resizeStart', $event)"
      @touchstart="emit('resizeStart', $event)"
      @dblclick="emit('autoFit', $event)"
    >
      <div
        class="h-full w-px transition-colors group-hover:bg-(--ui-border-accented)"
        :class="resizing ? 'bg-primary' : 'bg-transparent'"
      />
    </div>
  </div>
</template>
