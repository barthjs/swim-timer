<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableFilterField from '@/components/table/TableFilterField.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useRecordTexts } from '@/composables/record/useRecordTexts';
import {
  emptyRecordFilters,
  filterRecords,
  formatRecordDate,
  formatRecordDistance,
  formatRecordTime,
  recordDistanceKey,
  recordDistances,
  recordFacet,
  type RecordFilters,
  type RecordRow,
} from '@/lib/record/record';
import { tableFilterPill, tableOptionLabel } from '@/lib/table/filters';

// The records of the chosen record tables, searched, filtered, sorted, and paged on the client,
// since the record tables are held in memory as a whole like the meet data.
// #leading holds the choice of the record table next to the search.
const props = defineProps<{
  rows: RecordRow[];
  loading: boolean;
}>();

const SORTABLE = [
  'table',
  'course',
  'gender',
  'distance',
  'stroke',
  'ages',
  'time',
  'name',
  'club',
  'place',
  'date',
  'birthYear',
  'approved',
] as const;
type SortKey = (typeof SORTABLE)[number];

const { t, locale } = useI18n();
const { columnLabel, ageLabel } = useRecordTexts();

const columnState = useTableColumnState('records');
const filters = reactive<RecordFilters>(emptyRecordFilters());

const filtered = computed(() => filterRecords(props.rows, filters));

/** Returns the value a record is sorted by in a column. */
function sortValue(row: RecordRow, id: SortKey): unknown {
  const { record } = row;
  switch (id) {
    case 'table':
      return row.table?.name ?? '';
    case 'ages':
      return record.ageFrom * 1000 + record.ageTo;
    case 'approved':
      return record.approved ? 1 : 0;
    default:
      return record[id];
  }
}

const view = useTableView(filtered, {
  sortable: SORTABLE,
  value: sortValue,
  resetOn: () => filters,
});

const courseItems = computed(() =>
  recordFacet(props.rows, (record) => record.course).map(
    (course): { label: string; value: string } => ({
      label: t(`pages.records.courses.${course}`),
      value: course,
    }),
  ),
);
const genderItems = computed(() => recordFacet(props.rows, (record) => record.gender));
const strokeItems = computed(() => recordFacet(props.rows, (record) => record.stroke));
const distanceItems = computed(() =>
  recordDistances(props.rows).map((distance) => ({
    label: formatRecordDistance(distance),
    value: recordDistanceKey(distance),
  })),
);

/** Returns the pill of a filter with several values, which removes all of them. */
function listPill<K extends 'courses' | 'genders' | 'strokes' | 'distances'>(
  key: K,
  label: string,
  format: (value: RecordFilters[K][number]) => string,
) {
  const values = filters[key] as RecordFilters[K][number][];
  return tableFilterPill(key, label, values.map(format).join(', '), () => {
    filters[key] = [] as RecordFilters[K];
  });
}

const filterPills = computed(() => [
  ...listPill(
    'courses',
    columnLabel('course'),
    (course) => tableOptionLabel(courseItems.value, course) ?? course,
  ),
  ...listPill('genders', columnLabel('gender'), (gender) => gender),
  ...listPill('strokes', columnLabel('stroke'), (stroke) => stroke),
  ...listPill(
    'distances',
    columnLabel('distance'),
    (distance) => tableOptionLabel(distanceItems.value, distance) ?? distance,
  ),
]);

const canReset = computed(() => filters.query !== '' || filterPills.value.length > 0);

/** Clears the search and every filter. */
function reset(): void {
  Object.assign(filters, emptyRecordFilters());
}

const columns = computed<TableColumn<RecordRow>[]>(() => [
  {
    id: 'table',
    header: t('pages.records.table'),
    enableSorting: true,
  },
  {
    id: 'course',
    header: columnLabel('course'),
    enableSorting: true,
  },
  {
    id: 'gender',
    header: columnLabel('gender'),
    enableSorting: true,
  },
  {
    id: 'distance',
    header: columnLabel('distance'),
    enableSorting: true,
  },
  {
    id: 'stroke',
    header: columnLabel('stroke'),
    enableSorting: true,
  },
  {
    id: 'ages',
    header: t('pages.records.ages'),
    enableSorting: true,
  },
  {
    id: 'time',
    meta: {
      numeric: true,
    },
    header: columnLabel('time'),
    enableSorting: true,
  },
  {
    id: 'name',
    meta: {
      grow: true,
    },
    header: columnLabel('name'),
    enableSorting: true,
  },
  {
    id: 'club',
    header: columnLabel('club'),
    enableSorting: true,
  },
  {
    id: 'place',
    header: columnLabel('place'),
    enableSorting: true,
  },
  {
    id: 'date',
    header: columnLabel('date'),
    enableSorting: true,
  },
  {
    id: 'birthYear',
    header: columnLabel('birthYear'),
    enableSorting: true,
  },
  {
    id: 'approved',
    header: columnLabel('approved'),
    enableSorting: true,
  },
]);
</script>

<template>
  <div class="flex flex-col gap-3 lg:min-h-0 lg:flex-1">
    <TableToolbar
      v-model:search="filters.query"
      :placeholder="t('pages.records.filters.search')"
      :active-filters="filterPills"
      :can-reset="canReset"
      storage-key="records"
      @reset="reset"
    >
      <template #leading>
        <slot name="leading" />
      </template>
      <template #actions>
        <TableColumnMenu v-model:column-state="columnState" :columns="columns" />
      </template>

      <template #filters>
        <TableFilterField :label="columnLabel('course')" :active="filters.courses.length > 0">
          <USelectMenu
            v-model="filters.courses"
            :items="courseItems"
            value-key="value"
            multiple
            :placeholder="t('table.filterAll')"
            :search-input="false"
            class="w-full"
          />
        </TableFilterField>
        <TableFilterField :label="columnLabel('gender')" :active="filters.genders.length > 0">
          <USelectMenu
            v-model="filters.genders"
            :items="genderItems"
            multiple
            :placeholder="t('table.filterAll')"
            :search-input="false"
            class="w-full"
          />
        </TableFilterField>
        <TableFilterField :label="columnLabel('distance')" :active="filters.distances.length > 0">
          <USelectMenu
            v-model="filters.distances"
            :items="distanceItems"
            value-key="value"
            multiple
            :placeholder="t('table.filterAll')"
            :search-input="false"
            class="w-full"
          />
        </TableFilterField>
        <TableFilterField :label="columnLabel('stroke')" :active="filters.strokes.length > 0">
          <USelectMenu
            v-model="filters.strokes"
            :items="strokeItems"
            multiple
            :placeholder="t('table.filterAll')"
            :search-input="false"
            class="w-full"
          />
        </TableFilterField>
      </template>
    </TableToolbar>

    <DataTable
      v-model:column-state="columnState"
      v-model:sorting="view.sorting.value"
      v-model:page="view.page.value"
      v-model:limit="view.pageSize.value"
      class="lg:min-h-80"
      :data="view.visible.value"
      :total="view.total.value"
      :columns="columns"
      :loading="loading"
      :get-row-id="(row: RecordRow) => row.key"
    >
      <template #table-cell="{ row }">
        <div class="flex min-w-0 items-center gap-2">
          <UBadge
            v-if="row.original.table"
            :label="row.original.table.abbreviation"
            color="primary"
            variant="subtle"
          />
          <span class="truncate">{{ row.original.table?.name }}</span>
        </div>
      </template>
      <template #course-cell="{ row }">
        {{ t(`pages.records.courses.${row.original.record.course}`) }}
      </template>
      <template #gender-cell="{ row }">{{ row.original.record.gender }}</template>
      <template #distance-cell="{ row }">
        {{ formatRecordDistance(row.original.record) }}
      </template>
      <template #stroke-cell="{ row }">{{ row.original.record.stroke }}</template>
      <template #ages-cell="{ row }">
        {{ ageLabel(row.original.record) || t('pages.records.ageOpen') }}
      </template>
      <template #time-cell="{ row }">
        <span class="font-mono tabular-nums">{{ formatRecordTime(row.original.record.time) }}</span>
      </template>
      <template #name-cell="{ row }">
        <span class="text-highlighted">{{ row.original.record.name }}</span>
      </template>
      <template #club-cell="{ row }">{{ row.original.record.club }}</template>
      <template #place-cell="{ row }">{{ row.original.record.place }}</template>
      <template #date-cell="{ row }">
        {{ formatRecordDate(row.original.record.date, locale) }}
      </template>
      <template #birthYear-cell="{ row }">
        {{ row.original.record.birthYear || '' }}
      </template>
      <template #approved-cell="{ row }">
        <UBadge
          :label="
            row.original.record.approved
              ? t('pages.records.approved')
              : t('pages.records.notApproved')
          "
          :color="row.original.record.approved ? 'success' : 'neutral'"
          variant="subtle"
        />
      </template>

      <template #empty>
        <TableEmpty
          icon="i-lucide-trophy"
          :text="t('pages.records.empty')"
          :filtered="canReset"
          @reset="reset"
        />
      </template>
    </DataTable>
  </div>
</template>
