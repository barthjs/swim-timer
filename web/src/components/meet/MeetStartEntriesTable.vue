<script setup lang="ts">
import { computed, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import type { CompetitorResponse } from '@/api/types';
import DataTable from '@/components/table/DataTable.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import { useMeetLinks } from '@/composables/useMeetLinks';
import { useTableView } from '@/composables/table/useTableView';
import type { TableColumnState } from '@/lib/table/columns';
import {
  competitorName,
  eventName,
  formatDay,
  type MeetFilters,
  type MeetStartEntryRow,
} from '@/lib/meet';

type StartRow = {
  row: MeetStartEntryRow;
  name: string;
  birthYear: string;
  club: string;
  event: number;
  heat: number;
  lane: number;
  // The position of the event in the program, so sorting by it follows the schedule, see eventPositions.
  schedule: number;
};

const SORTABLE = ['name', 'birthYear', 'club', 'event', 'heat', 'lane', 'schedule'] as const;

// Every start entry of the meet that matches the filters.
const props = defineProps<{
  rows: MeetStartEntryRow[];
  filters: MeetFilters;
}>();

// The column layout, kept by the page, which shows the column menu next to the search
const columnState = defineModel<TableColumnState>('columnState', {
  required: true,
});

const emit = defineEmits<{
  showCompetitor: [competitor: CompetitorResponse];
  selectClub: [club: string];
  showHeat: [event: number, heat: number];
}>();

const { t, locale } = useI18n();
const { competitorLink, clubLink, eventLink } = useMeetLinks({
  button: resolveComponent('UButton'),
  tooltip: resolveComponent('UTooltip'),
});

const data = computed<StartRow[]>(() =>
  props.rows.map((row) => ({
    row,
    name: competitorName(row.competitor),
    birthYear: row.competitor?.birthYear ?? '',
    club: row.competitor?.club ?? '',
    event: row.event.number,
    heat: row.entry.heat,
    lane: row.entry.lane,
    schedule: row.eventPosition,
  })),
);

const view = useTableView(data, {
  sortable: SORTABLE,
  value: (row, id) => row[id as (typeof SORTABLE)[number]],
  resetOn: () => props.filters,
});

const columns = computed<TableColumn<StartRow>[]>(() => [
  {
    accessorKey: 'name',
    header: t('pages.meet.starts.name'),
    enableSorting: true,
    cell: ({ row }) => {
      const competitor = row.original.row.competitor;
      if (!competitor) return t('pages.meet.starts.unknown');
      return competitorLink(competitor, () => emit('showCompetitor', competitor));
    },
  },
  {
    accessorKey: 'birthYear',
    header: t('pages.meet.starts.birthYear'),
    enableSorting: true,
    size: 120,
  },
  {
    accessorKey: 'club',
    header: t('pages.meet.starts.club'),
    enableSorting: true,
    cell: ({ row }) => clubLink(row.original.club, () => emit('selectClub', row.original.club)),
  },
  {
    accessorKey: 'event',
    header: t('pages.meet.starts.event'),
    enableSorting: true,
    cell: ({ row }) =>
      eventLink(`${row.original.event} · ${eventName(row.original.row.event)}`, () =>
        emit('showHeat', row.original.event, row.original.heat),
      ),
  },
  {
    accessorKey: 'heat',
    header: t('pages.meet.starts.heat'),
    enableSorting: true,
    size: 90,
  },
  {
    accessorKey: 'lane',
    header: t('pages.meet.starts.lane'),
    enableSorting: true,
    size: 90,
  },
  {
    accessorKey: 'schedule',
    header: t('pages.meet.starts.schedule'),
    enableSorting: true,
    size: 170,
    cell: ({ row }) =>
      [formatDay(row.original.row.event.date, locale.value), row.original.row.event.time]
        .filter(Boolean)
        .join(' '),
  },
]);

defineExpose({
  columns,
});
</script>

<template>
  <DataTable
    v-model:column-state="columnState"
    v-model:sorting="view.sorting.value"
    v-model:page="view.page.value"
    v-model:limit="view.pageSize.value"
    :data="view.visible.value"
    :total="view.total.value"
    :columns="columns"
    :get-row-id="(row: StartRow) => row.row.key"
  >
    <template #empty>
      <TableEmpty icon="i-lucide-list" :text="t('pages.meet.empty')" />
    </template>
  </DataTable>
</template>
