<script setup lang="ts">
import { computed, useSlots, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { defineShortcuts } from '@nuxt/ui/composables';

import TableSearchInput from '@/components/table/TableSearchInput.vue';
import type { ShortcutKey } from '@/composables/useShortcutLabels';
import { useStoredState } from '@/composables/useStoredState';
import type { TableFilterPill } from '@/lib/table/filters';

// The toolbar above a list. One row holds an optional title (#leading), the search, which takes the free width,
// the filter button, the reset button, and the actions of the list (#actions) on the right.
// The filters (#filters) sit below in an even grid of at most eight columns, each one a TableFilterField.
// The grid collapses with the filter button and starts collapsed, the state is kept per list in the browser.
// While collapsed, the set filters show as pills that remove their filter.
// The reset button is there whenever the list has a search or filters,
// and only disabled while nothing is set, so the row does not shift.
const props = withDefaults(
  defineProps<{
    placeholder?: string;
    hideSearch?: boolean;
    // The keys that focus the search from anywhere on the page, also from another input
    searchShortcut?: ShortcutKey[];
    // The set filters, for the pills and the count on the filter button
    activeFilters?: TableFilterPill[];
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
    searchShortcut: undefined,
    activeFilters: () => [],
    canReset: false,
    storageKey: undefined,
    loading: false,
    onReload: undefined,
  },
);

const emit = defineEmits<{ reset: [] }>();

const search = defineModel<string>('search', {
  default: '',
});

const { t } = useI18n();
const slots = useSlots();

const filtersOpen = useStoredState(
  props.storageKey ? `table.${props.storageKey}.filters` : undefined,
  () => false,
  (raw) => (typeof raw === 'boolean' ? raw : undefined),
);

const hasFilters = computed(() => !!slots.filters);
const filterCount = computed(() => props.activeFilters.length);

const searchInput = useTemplateRef<{ focus: () => void }>('searchInput');

/** focusSearch moves the focus into the search. */
function focusSearch() {
  searchInput.value?.focus();
}

// The keys of a shortcut joined by "_" are the notation of defineShortcuts, such as "meta_f"
defineShortcuts(
  computed(() => {
    if (!props.searchShortcut || props.hideSearch) return {};
    return {
      [props.searchShortcut.join('_')]: {
        usingInput: true,
        handler: focusSearch,
      },
    };
  }),
);

/** toggleFilters opens or closes the filter grid. */
function toggleFilters() {
  filtersOpen.value = !filtersOpen.value;
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex flex-wrap items-center gap-x-6 gap-y-2">
      <div v-if="$slots.leading" class="max-w-full min-w-0 shrink-0">
        <slot name="leading" />
      </div>

      <div class="flex min-w-64 flex-1 flex-wrap items-center gap-2">
        <TableSearchInput
          v-if="!hideSearch"
          ref="searchInput"
          v-model="search"
          :placeholder="placeholder"
          class="min-w-48 flex-1"
        />
        <div v-else class="flex-1" />

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

        <UTooltip v-if="hasFilters || !hideSearch" :text="t('table.resetAll')">
          <UButton
            icon="i-lucide-funnel-x"
            color="neutral"
            variant="ghost"
            :disabled="!canReset"
            :aria-label="t('table.resetAll')"
            @click="emit('reset')"
          />
        </UTooltip>

        <div class="ml-auto flex flex-wrap items-center justify-end gap-1.5">
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
    </div>

    <!-- Hidden with a class instead of v-if, so the filter controls stay mounted and keep what they loaded -->
    <div
      v-if="hasFilters"
      class="grid-cols-[repeat(auto-fill,minmax(max(11rem,calc((100%_-_5.25rem)/8)),1fr))] items-end gap-x-3 gap-y-2"
      :class="filtersOpen ? 'grid' : 'hidden'"
    >
      <slot name="filters" />
    </div>

    <div v-if="!filtersOpen && activeFilters.length > 0" class="flex flex-wrap items-center gap-2">
      <span class="text-muted text-sm">{{ t('table.activeFilters') }}</span>
      <span
        v-for="pill in activeFilters"
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
