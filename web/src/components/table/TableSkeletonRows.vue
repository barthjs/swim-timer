<script setup lang="ts">
// Placeholder rows for the loading slot of a table, so a table that loads shows the shape of rows
// instead of its empty state. Each row is as high as a row of the table, so a page of them fills the table.
// The widths only hint at content, they do not follow the columns.
withDefaults(
  defineProps<{
    columns?: number;
    rows?: number;
  }>(),
  {
    columns: 5,
    rows: 8,
  },
);

const WIDTHS = ['60%', '85%', '45%', '70%'];

/** widthFor returns the width of a placeholder bar, varied by its position. */
function widthFor(row: number, column: number): string {
  return WIDTHS[(row + column) % WIDTHS.length] ?? '60%';
}
</script>

<template>
  <div class="divide-default flex flex-col divide-y" aria-hidden="true">
    <div v-for="row in rows" :key="row" class="flex items-center gap-6 px-4 py-4">
      <div v-for="column in columns" :key="column" class="min-w-0 flex-1">
        <USkeleton class="h-4" :style="{ width: widthFor(row, column) }" />
      </div>
    </div>
  </div>
</template>
