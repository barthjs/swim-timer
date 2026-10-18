<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

// Pagination footer of a table with the page size, the visible range, and the page controls.
// DataTable places it below the rows, where it stays in view while the rows scroll.
const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    total: number;
    limits?: number[];
  }>(),
  {
    limits: () => [10, 25, 50, 100],
  },
);

const page = defineModel<number>('page', {
  required: true,
});
const limit = defineModel<number>('limit', {
  required: true,
});

const rangeLabel = computed(() => {
  if (props.total === 0) return t('table.rangeEmpty');
  const start = (page.value - 1) * limit.value + 1;
  const end = Math.min(page.value * limit.value, props.total);
  return t('table.range', {
    start,
    end,
    total: props.total,
  });
});
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
    <div class="flex items-center gap-3">
      <USelect
        v-model="limit"
        :items="limits"
        :aria-label="t('table.perPage')"
        size="sm"
        class="w-20"
      />
      <span class="text-muted text-sm tabular-nums">{{ rangeLabel }}</span>
    </div>

    <UPagination
      v-model:page="page"
      :total="total"
      :items-per-page="limit"
      :sibling-count="1"
      show-edges
      size="sm"
    />
  </div>
</template>
