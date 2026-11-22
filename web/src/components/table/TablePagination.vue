<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useBreakpoint } from '@/composables/ui/useBreakpoint';
import { PAGE_SIZES } from '@/lib/table/view';

const props = withDefaults(
  defineProps<{
    total: number;
    limits?: number[];
  }>(),
  {
    limits: () => [...PAGE_SIZES],
  },
);

const page = defineModel<number>('page', {
  required: true,
});

const limit = defineModel<number>('limit', {
  required: true,
});

// Pagination footer of a table with the page size, the visible range, and the page controls.
// DataTable places it below the rows, where it stays in view while the rows scroll.
// It keeps one row on narrow screens, which drop the range and show only the current page between the arrows.
const { t } = useI18n();

const wide = useBreakpoint('sm');

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
  <div class="flex min-w-0 items-center justify-between gap-3 sm:gap-4">
    <div class="flex min-w-0 items-center gap-3">
      <USelect
        v-model="limit"
        :items="limits"
        :aria-label="t('table.perPage')"
        size="sm"
        class="w-20 shrink-0"
      />
      <span class="text-muted hidden truncate text-sm tabular-nums sm:inline">{{
        rangeLabel
      }}</span>
    </div>

    <UPagination
      v-model:page="page"
      :total="total"
      :items-per-page="limit"
      :sibling-count="wide ? 1 : 0"
      :show-edges="wide"
      active-color="neutral"
      active-variant="solid"
      size="sm"
      class="shrink-0"
    />
  </div>
</template>
