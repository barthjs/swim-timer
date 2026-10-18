<script setup lang="ts">
import { computed, h, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import type { CompetitorResponse } from '@/api/types';
import MeetHeatTable from '@/components/meet/MeetHeatTable.vue';
import DataTable from '@/components/table/DataTable.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import type { TableColumnState, TableSorting } from '@/lib/table/columns';
import { EventState, formatDay, type MeetIndex } from '@/lib/meet';

export type EventRow = {
  number: number;
  // The position of the event in the program, so sorting by it follows the schedule, see eventPositions.
  schedule: number;
  date: string;
  // Empty for an event without a scheduled time.
  time: string;
  // The name together with the phase, see eventName.
  name: string;
  heatCount: number;
  startCount: number;
  state: EventState | null;
};

// The events of the meet, each expanding to its heats. Sorting and pages are owned by the page,
// so it can jump to an event, see useTableView.
const props = defineProps<{
  index: MeetIndex;
  rows: EventRow[];
  total: number;
  liveEvent: number | null;
  liveHeat: number | null;
}>();

// The column layout, kept by the page, which shows the column menu next to the search
const columnState = defineModel<TableColumnState>('columnState', {
  required: true,
});

const sorting = defineModel<TableSorting>('sorting', {
  required: true,
});
const page = defineModel<number>('page', {
  required: true,
});
const pageSize = defineModel<number>('pageSize', {
  required: true,
});
const expanded = defineModel<Record<string, boolean>>('expanded', {
  required: true,
});

const emit = defineEmits<{
  showCompetitor: [competitor: CompetitorResponse];
  selectClub: [club: string];
}>();

const { t, locale } = useI18n();

// The cells render through h(), where components of a template do not resolve on their own
const UBadge = resolveComponent('UBadge');
const UButton = resolveComponent('UButton');

const stateColors = {
  [EventState.Done]: 'neutral',
  [EventState.Current]: 'success',
  [EventState.Upcoming]: 'info',
} as const;

const columns = computed<TableColumn<EventRow>[]>(() => [
  {
    id: 'expand',
    size: 64,
    cell: ({ row }) =>
      h(UButton, {
        icon: row.getIsExpanded() ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right',
        color: 'neutral',
        variant: 'ghost',
        size: 'sm',
        'aria-label': t('pages.meet.events.showHeats'),
        onClick: () => row.toggleExpanded(),
      }),
  },
  {
    accessorKey: 'number',
    header: t('pages.meet.events.number'),
    enableSorting: true,
    size: 90,
    cell: ({ row }) =>
      h(
        'span',
        {
          id: `meet-event-${row.original.number}`,
          class: 'text-highlighted scroll-mt-24 font-mono font-semibold',
        },
        String(row.original.number),
      ),
  },
  {
    accessorKey: 'schedule',
    header: t('pages.meet.events.schedule'),
    enableSorting: true,
    size: 170,
    cell: ({ row }) =>
      [formatDay(row.original.date, locale.value), row.original.time].filter(Boolean).join(' '),
  },
  {
    accessorKey: 'name',
    header: t('pages.meet.events.name'),
    enableSorting: true,
  },
  {
    accessorKey: 'heatCount',
    header: t('pages.meet.events.heats'),
    enableSorting: true,
    size: 100,
  },
  {
    accessorKey: 'startCount',
    header: t('pages.meet.events.starts'),
    enableSorting: true,
    size: 100,
  },
  {
    accessorKey: 'state',
    header: t('pages.meet.events.state'),
    size: 140,
    cell: ({ row }) => {
      const state = row.original.state;
      if (!state) return '';
      return h(UBadge, {
        label: t(`pages.meet.events.states.${state}`),
        color: stateColors[state],
        variant: 'solid',
      });
    },
  },
]);

defineExpose({
  columns,
});
</script>

<template>
  <DataTable
    v-model:column-state="columnState"
    v-model:sorting="sorting"
    v-model:page="page"
    v-model:limit="pageSize"
    v-model:expanded="expanded"
    :data="rows"
    :total="total"
    :columns="columns"
    :get-row-id="(row: EventRow) => String(row.number)"
  >
    <template #expanded="{ row }">
      <MeetHeatTable
        :heats="props.index.heats.filter((heat) => heat.event.number === row.original.number)"
        :highlight="
          liveEvent === row.original.number && liveHeat !== null
            ? { event: liveEvent, heat: liveHeat }
            : null
        "
        @show-competitor="(competitor: CompetitorResponse) => emit('showCompetitor', competitor)"
        @select-club="(club: string) => emit('selectClub', club)"
      />
    </template>

    <template #empty>
      <TableEmpty icon="i-lucide-list-ordered" :text="t('pages.meet.empty')" />
    </template>
  </DataTable>
</template>
