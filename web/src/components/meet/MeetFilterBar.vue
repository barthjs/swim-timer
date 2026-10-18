<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import TableFilterField from '@/components/table/TableFilterField.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import { tableFilterPill } from '@/lib/table/filters';
import { EventKind, facetValues, formatDay, type MeetFilters, type MeetIndex } from '@/lib/meet';

// The search and the filters of the meet overview in the toolbar of the tables. Every menu offers only values
// that occur in the meet. #actions sits next to the search, for example the column menu of the shown table.
const filters = defineModel<MeetFilters>({
  required: true,
});

const props = defineProps<{
  index: MeetIndex;
}>();

const emit = defineEmits<{
  reset: [];
}>();

const { t, locale } = useI18n();

const dayItems = computed(() =>
  facetValues(props.index.events, (event) => event.date).map((value) => ({
    label: formatDay(value, locale.value),
    value,
  })),
);
const styleItems = computed(() => facetValues(props.index.events, (event) => event.style?.name));
const distanceItems = computed(() =>
  facetValues(props.index.events, (event) => event.distance?.name),
);
const categoryItems = computed(() =>
  facetValues(props.index.events, (event) => event.category?.name),
);
const clubItems = computed(() => props.index.clubs.map((club) => club.name));

const kindItems = computed(() => [
  {
    label: t('pages.meet.filters.kindAll'),
    value: EventKind.All,
  },
  {
    label: t('pages.meet.filters.kindIndividual'),
    value: EventKind.Individual,
  },
  {
    label: t('pages.meet.filters.kindRelay'),
    value: EventKind.Relay,
  },
]);

/** listPill returns the pill of a filter with several values, which removes all of them. */
function listPill(
  key: 'days' | 'styles' | 'distances' | 'categories' | 'clubs',
  label: string,
  format: (value: string) => string = (value) => value,
) {
  return tableFilterPill(key, label, filters.value[key].map(format).join(', '), () => {
    filters.value[key] = [];
  });
}

const filterPills = computed(() => [
  ...listPill('days', t('pages.meet.filters.days'), (value) => formatDay(value, locale.value)),
  ...listPill('styles', t('pages.meet.filters.styles')),
  ...listPill('distances', t('pages.meet.filters.distances')),
  ...listPill('categories', t('pages.meet.filters.categories')),
  ...listPill('clubs', t('pages.meet.filters.clubs')),
  ...tableFilterPill(
    'kind',
    t('pages.meet.filters.kind'),
    filters.value.kind === EventKind.All
      ? undefined
      : kindItems.value.find((item) => item.value === filters.value.kind)?.label,
    () => {
      filters.value.kind = EventKind.All;
    },
  ),
  ...tableFilterPill(
    'upcoming',
    t('pages.meet.filters.events'),
    filters.value.upcomingOnly ? t('pages.meet.filters.upcomingShort') : undefined,
    () => {
      filters.value.upcomingOnly = false;
    },
  ),
]);

const canReset = computed(() => filters.value.query !== '' || filterPills.value.length > 0);
</script>

<template>
  <!-- The shortcut of the browser search jumps to the search of the meet -->
  <TableToolbar
    v-model:search="filters.query"
    :placeholder="t('pages.meet.filters.search')"
    :search-shortcut="['meta', 'f']"
    :active-filters="filterPills"
    :can-reset="canReset"
    storage-key="meet"
    @reset="emit('reset')"
  >
    <template #actions>
      <slot name="actions" />
    </template>

    <template #filters>
      <TableFilterField :label="t('pages.meet.filters.days')" :active="filters.days.length > 0">
        <USelectMenu
          v-model="filters.days"
          :items="dayItems"
          value-key="value"
          multiple
          :placeholder="t('table.filterAll')"
          :search-input="false"
          class="w-full"
        />
      </TableFilterField>
      <TableFilterField :label="t('pages.meet.filters.styles')" :active="filters.styles.length > 0">
        <USelectMenu
          v-model="filters.styles"
          :items="styleItems"
          multiple
          :placeholder="t('table.filterAll')"
          :search-input="false"
          class="w-full"
        />
      </TableFilterField>
      <TableFilterField
        :label="t('pages.meet.filters.distances')"
        :active="filters.distances.length > 0"
      >
        <USelectMenu
          v-model="filters.distances"
          :items="distanceItems"
          multiple
          :placeholder="t('table.filterAll')"
          :search-input="false"
          class="w-full"
        />
      </TableFilterField>
      <TableFilterField
        :label="t('pages.meet.filters.categories')"
        :active="filters.categories.length > 0"
      >
        <USelectMenu
          v-model="filters.categories"
          :items="categoryItems"
          multiple
          :placeholder="t('table.filterAll')"
          :search-input="false"
          class="w-full"
        />
      </TableFilterField>
      <TableFilterField :label="t('pages.meet.filters.clubs')" :active="filters.clubs.length > 0">
        <USelectMenu
          v-model="filters.clubs"
          :items="clubItems"
          multiple
          :placeholder="t('table.filterAll')"
          :search-input="{ placeholder: t('pages.meet.filters.searchClub') }"
          class="w-full"
        />
      </TableFilterField>
      <TableFilterField
        :label="t('pages.meet.filters.kind')"
        :active="filters.kind !== EventKind.All"
      >
        <USelect v-model="filters.kind" :items="kindItems" class="w-full" />
      </TableFilterField>
      <TableFilterField
        :label="t('pages.meet.filters.events')"
        :active="filters.upcomingOnly"
        :hint="t('pages.meet.filters.upcomingOnlyHint')"
      >
        <div class="flex h-8 items-center">
          <USwitch v-model="filters.upcomingOnly" :label="t('pages.meet.filters.upcomingShort')" />
        </div>
      </TableFilterField>
    </template>
  </TableToolbar>
</template>
