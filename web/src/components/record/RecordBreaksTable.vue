<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';

import { recordBreakRepository } from '@/api/repositories/recordBreak';
import type { RecordBreakResponse } from '@/api/types/recordBreak';
import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableRowActions from '@/components/table/TableRowActions.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useRecordTexts } from '@/composables/record/useRecordTexts';
import { useDates } from '@/composables/ui/useDates';
import { formatRecordTime, matchesRecordSearch } from '@/lib/record/record';
import { TABLE_ACTIONS_COLUMN } from '@/lib/table/columns';

// The records set or equalled at the meet, the most recent first. A meet sets a handful of them, so the table searches,
// sorts, and pages on the client. A record break is removed after a disqualification,
// and all of them before the next meet, so the next heats are checked against the tables again.
const props = defineProps<{
  breaks: RecordBreakResponse[];
  loading: boolean;
}>();

const emit = defineEmits<{
  changed: [];
}>();

const SORTABLE = ['brokenAt', 'table', 'heat', 'discipline', 'name', 'time', 'previous'] as const;

const { t } = useI18n();
const { formatDateTimeNumeric } = useDates();
const { ageLabel, disciplineLabel } = useRecordTexts();
const { confirm } = useConfirm();
const { confirmDelete } = useDeleteConfirm();

const columnState = useTableColumnState('record-breaks');
const search = ref('');

const filtered = computed(() =>
  props.breaks.filter((item) => matchesRecordSearch(item.record, search.value)),
);

/** Returns the value a record break is sorted by in a column. */
function sortValue(item: RecordBreakResponse, id: (typeof SORTABLE)[number]): unknown {
  switch (id) {
    case 'brokenAt':
      return item.brokenAt;
    case 'table':
      return item.tableName;
    case 'heat':
      return item.event * 10_000 + item.heat * 100 + item.lane;
    case 'discipline':
      return disciplineLabel(item.record);
    case 'name':
      return item.record.name;
    case 'time':
      return item.record.time;
    case 'previous':
      return item.previous.time;
  }
}

const view = useTableView(filtered, {
  sortable: SORTABLE,
  value: sortValue,
  resetOn: search,
});

const { mutate: deleteBreak } = useMutation(
  async (item: RecordBreakResponse) => recordBreakRepository.delete(item.id),
  {
    key: (item) => item.id,
    confirm: async (item) =>
      confirmDelete({
        title: t('pages.records.breaks.deleteTitle'),
        description: t('pages.records.breaks.deleteDescription', {
          name: item.record.name,
        }),
      }),
    success: () => t('pages.records.breaks.deleted'),
    onSuccess: () => emit('changed'),
  },
);

const { mutate: clear, pending: clearing } = useMutation(
  async () => recordBreakRepository.clear(),
  {
    confirm: async () =>
      confirm({
        title: t('pages.records.breaks.clearTitle'),
        description: t('pages.records.breaks.clearDescription'),
        confirmLabel: t('pages.records.breaks.clear'),
        color: 'error',
        icon: 'i-lucide-trash-2',
      }),
    success: () => t('pages.records.breaks.cleared'),
    onSuccess: () => emit('changed'),
  },
);

/** Returns the actions of a row. */
const rowMenuItems = (item: RecordBreakResponse): DropdownMenuItem[][] => [
  [
    {
      label: t('common.delete'),
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => void deleteBreak(item),
    },
  ],
];

const columns = computed<TableColumn<RecordBreakResponse>[]>(() => [
  {
    id: 'brokenAt',
    header: t('pages.records.breaks.brokenAt'),
    enableSorting: true,
  },
  {
    id: 'table',
    header: t('pages.records.table'),
    enableSorting: true,
  },
  {
    id: 'heat',
    header: t('pages.records.breaks.heat'),
    enableSorting: true,
  },
  {
    id: 'discipline',
    header: t('pages.records.breaks.discipline'),
    enableSorting: true,
  },
  {
    id: 'name',
    meta: {
      grow: true,
    },
    header: t('fields.name'),
    enableSorting: true,
  },
  {
    id: 'time',
    meta: {
      numeric: true,
    },
    header: t('fields.time'),
    enableSorting: true,
  },
  {
    id: 'previous',
    meta: {
      numeric: true,
    },
    header: t('pages.records.breaks.previous'),
    enableSorting: true,
  },
  {
    id: TABLE_ACTIONS_COLUMN,
  },
]);
</script>

<template>
  <div class="flex flex-col gap-3 lg:min-h-0 lg:flex-1">
    <TableToolbar
      v-model:search="search"
      :placeholder="t('pages.records.filters.search')"
      :can-reset="!!search"
      @reset="search = ''"
    >
      <template #actions>
        <TableColumnMenu v-model:column-state="columnState" :columns="columns" />
        <UButton
          :label="t('pages.records.breaks.clear')"
          icon="i-lucide-trash-2"
          color="error"
          variant="outline"
          :loading="clearing"
          :disabled="breaks.length === 0"
          @click="clear()"
        />
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
      :get-row-id="(item: RecordBreakResponse) => item.id"
    >
      <template #brokenAt-cell="{ row }">
        {{ formatDateTimeNumeric(row.original.brokenAt) }}
      </template>
      <template #table-cell="{ row }">
        <div class="flex min-w-0 items-center gap-2">
          <UBadge :label="row.original.abbreviation" color="primary" variant="subtle" />
          <span class="truncate">{{ row.original.tableName }}</span>
        </div>
      </template>
      <template #heat-cell="{ row }">
        {{
          t('pages.records.breaks.heatLabel', {
            event: row.original.event,
            heat: row.original.heat,
            lane: row.original.lane,
          })
        }}
      </template>
      <template #discipline-cell="{ row }">
        {{ disciplineLabel(row.original.record) }}
        <span v-if="ageLabel(row.original.record)" class="text-muted">
          · {{ ageLabel(row.original.record) }}
        </span>
      </template>
      <template #name-cell="{ row }">
        <span class="text-highlighted">{{ row.original.record.name }}</span>
      </template>
      <template #time-cell="{ row }">
        <div class="flex items-center gap-2">
          <span class="font-mono tabular-nums">{{
            formatRecordTime(row.original.record.time)
          }}</span>
          <UBadge
            v-if="row.original.equalled"
            :label="t('pages.records.breaks.equalled')"
            color="info"
            variant="subtle"
          />
        </div>
      </template>
      <template #previous-cell="{ row }">
        <div class="flex min-w-0 items-center gap-2">
          <span class="font-mono tabular-nums">{{
            formatRecordTime(row.original.previous.time)
          }}</span>
          <span class="text-muted truncate">{{ row.original.previous.name }}</span>
        </div>
      </template>
      <template #actions-cell="{ row }">
        <TableRowActions :items="rowMenuItems(row.original)" />
      </template>

      <template #empty>
        <TableEmpty
          icon="i-lucide-medal"
          :text="t('pages.records.breaks.empty')"
          :filtered="!!search"
          @reset="search = ''"
        />
      </template>
    </DataTable>
  </div>
</template>
