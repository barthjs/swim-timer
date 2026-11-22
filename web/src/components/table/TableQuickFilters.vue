<script setup lang="ts" generic="TFilters extends TableFilterValues">
import { computed, ref, watch } from 'vue';

import {
  sameTableFilters,
  type TableFilterValues,
  type TableQuickFilter,
} from '@/lib/table/quickFilters';

// The quick filters of a list as a group of buttons. The one whose values equal the filters of the list is active,
// so changing another filter leaves none of them active.
const props = defineProps<{
  items: TableQuickFilter[];
  // The current filters of the list
  filters: Partial<TFilters>;
}>();

const emit = defineEmits<{
  select: [quickFilter: TableQuickFilter];
}>();

// The chosen quick filter shows as active at once, since the filters only change once the navigation finished,
// and the button would otherwise show its pressed state and then the active one.
const chosenId = ref<string>();

watch(
  () => props.filters,
  () => {
    chosenId.value = undefined;
  },
);

const activeId = computed(
  () =>
    chosenId.value ?? props.items.find((item) => sameTableFilters(props.filters, item.filters))?.id,
);

const toggleItems = computed(() =>
  props.items.map((item) => ({
    label: item.label,
    icon: item.icon,
    value: item.id,
  })),
);

/**
 * Marks the chosen quick filter as active and passes it on, while a press on the active one keeps it,
 * since a list always shows one of its filters.
 */
const choose = (value: unknown) => {
  const item = props.items.find((candidate) => candidate.id === value);
  if (!item) return;
  chosenId.value = item.id;
  emit('select', item);
};
</script>

<template>
  <UToggleGroup
    type="single"
    :model-value="activeId"
    :items="toggleItems"
    attached
    color="neutral"
    variant="outline"
    active-color="neutral"
    active-variant="solid"
    :ui="{
      root: 'w-full',
      item: 'flex-auto justify-center transition-none',
    }"
    @update:model-value="choose"
  />
</template>
