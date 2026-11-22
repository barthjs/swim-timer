<script setup lang="ts" generic="TFilters extends TableFilterValues">
import { computed, useSlots, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

import TableQuickFilters from '@/components/table/TableQuickFilters.vue';
import TableSearchInput from '@/components/table/TableSearchInput.vue';
import { useSearchShortcut } from '@/composables/ui/useSearchShortcut';
import { useStoredState } from '@/composables/ui/useStoredState';
import type { TableFilterPill } from '@/lib/table/filters';
import {
  sameTableFilters,
  type TableFilterValues,
  type TableQuickFilter,
} from '@/lib/table/quickFilters';

// The toolbar above a list with an optional title (#leading), the quick filters, the search, the filters,
// and the actions of the list. The filter grid keeps its open state per list in the browser,
// and while it is collapsed the set filters show as pills, except those the active quick filter already shows.
const props = withDefaults(
  defineProps<{
    placeholder?: string;
    hideSearch?: boolean;
    // The set filters, for the pills and the count on the filter button
    activeFilters?: TableFilterPill[];
    quickFilters?: TableQuickFilter[];
    // The current filters of the list, which decide the active quick filter
    filters?: Partial<TFilters>;
    // Whether a search or a filter is set, which enables the reset button
    canReset?: boolean;
    // Keeps whether the filters are open in the browser, per list
    storageKey?: string;
    loading?: boolean;
    // Shows a reload button
    onReload?: () => void;
  }>(),
  {
    placeholder: '',
    hideSearch: false,
    activeFilters: () => [],
    quickFilters: () => [],
    filters: () => ({}),
    canReset: false,
    storageKey: undefined,
    loading: false,
    onReload: undefined,
  },
);

const search = defineModel<string>('search', {
  default: '',
});

const emit = defineEmits<{
  reset: [];
  quickFilter: [quickFilter: TableQuickFilter];
}>();

const { t } = useI18n();
const slots = useSlots();

const filtersOpen = useStoredState(
  props.storageKey ? `table.${props.storageKey}.filters` : undefined,
  () => false,
  (raw) => (typeof raw === 'boolean' ? raw : undefined),
);

const hasFilters = computed(() => !!slots.filters);
const hasQuickFilters = computed(() => props.quickFilters.length > 0);

const activeQuickFilter = computed(() =>
  props.quickFilters.find((item) => sameTableFilters(props.filters, item.filters)),
);

const shownFilters = computed(() =>
  props.activeFilters.filter((pill) => !(pill.key in (activeQuickFilter.value?.filters ?? {}))),
);
const filterCount = computed(() => shownFilters.value.length);

const hasSearch = computed(() => !props.hideSearch);

const filterRowClass = computed(() =>
  slots.leading ? 'order-3 flex basis-full flex-wrap items-center gap-2 sm:contents' : 'contents',
);

const quickFiltersClass = computed(() => {
  if (!hasSearch.value) return slots.leading ? 'flex-1 sm:order-2' : 'order-1 flex-1';
  if (!slots.leading) return 'order-1 flex-auto sm:flex-none';
  return 'basis-full sm:order-2 sm:basis-auto';
});

const controlsClass = computed(() => {
  if (!hasSearch.value) return slots.leading ? 'sm:order-3' : 'order-1 sm:order-3';
  const wide = 'sm:order-3 sm:min-w-64 sm:flex-1 sm:basis-auto';
  if (slots.leading) return `basis-full ${wide}`;
  return hasQuickFilters.value ? `order-3 basis-full ${wide}` : `order-1 min-w-48 flex-1 ${wide}`;
});

const actionsClass = computed(() => (hasSearch.value || !hasQuickFilters.value ? 'ms-auto' : ''));

const searchInput = useTemplateRef<{ focus: () => void }>('searchInput');

/** Moves the focus into the search. */
function focusSearch() {
  searchInput.value?.focus();
}

useSearchShortcut(focusSearch, () => !props.hideSearch);

/** Opens or closes the filter grid. */
function toggleFilters() {
  filtersOpen.value = !filtersOpen.value;
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex flex-wrap items-center gap-2">
      <div v-if="$slots.leading" class="order-1 min-w-0 flex-1 sm:me-4 sm:flex-none sm:shrink-0">
        <slot name="leading" />
      </div>

      <div :class="filterRowClass">
        <div v-if="hasQuickFilters" class="flex" :class="quickFiltersClass">
          <TableQuickFilters
            :items="quickFilters"
            :filters="filters"
            @select="emit('quickFilter', $event)"
          />
        </div>

        <div
          v-if="hasSearch || hasFilters"
          class="table-toolbar-controls flex min-w-0 items-center gap-2"
          :class="controlsClass"
        >
          <TableSearchInput
            v-if="hasSearch"
            ref="searchInput"
            v-model="search"
            :placeholder="placeholder"
            class="min-w-0 flex-1"
          />

          <UButton
            v-if="hasFilters"
            :label="t('table.filter')"
            icon="i-lucide-list-filter"
            color="neutral"
            variant="outline"
            :aria-expanded="filtersOpen"
            @click="toggleFilters"
          >
            <template v-if="filterCount > 0" #trailing>
              <UBadge :label="String(filterCount)" size="sm" class="rounded-full" />
            </template>
          </UButton>

          <UTooltip :text="t('table.resetAll')">
            <UButton
              icon="i-lucide-funnel-x"
              color="neutral"
              variant="ghost"
              :disabled="!canReset"
              :aria-label="t('table.resetAll')"
              @click="emit('reset')"
            />
          </UTooltip>
        </div>
      </div>

      <div
        class="order-2 flex shrink-0 items-center justify-end gap-2 sm:order-4"
        :class="actionsClass"
      >
        <UTooltip v-if="onReload" :text="t('table.reload')">
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="outline"
            :loading="loading"
            :aria-label="t('table.reload')"
            @click="onReload"
          />
        </UTooltip>
        <slot name="actions" />
      </div>
    </div>

    <!-- Hidden with a class instead of v-if, so the filter controls stay mounted and keep what they loaded -->
    <div
      v-if="hasFilters"
      class="grid-cols-table-filters items-end gap-x-3 gap-y-2"
      :class="filtersOpen ? 'grid' : 'hidden'"
    >
      <slot name="filters" />
    </div>

    <div v-if="!filtersOpen && shownFilters.length > 0" class="flex flex-wrap items-center gap-2">
      <span class="text-muted text-sm">{{ t('table.activeFilters') }}</span>
      <span
        v-for="pill in shownFilters"
        :key="pill.key"
        class="bg-primary/10 ring-primary/25 inline-flex max-w-80 items-center gap-1 rounded-md py-0.5 ps-2 pe-1 text-sm ring ring-inset"
        :title="`${pill.label}: ${pill.value}`"
      >
        <span class="truncate">
          <span class="text-muted">{{ pill.label }}:</span>
          {{ pill.value }}
        </span>
        <UButton
          icon="i-lucide-x"
          size="xs"
          color="neutral"
          variant="link"
          class="p-0.5"
          :aria-label="t('table.removeFilter', { label: pill.label })"
          @click="pill.remove"
        />
      </span>
    </div>
  </div>
</template>
