<script setup lang="ts">
import { computed, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import DataTable from '@/components/table/DataTable.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import { useMeetLinks } from '@/composables/useMeetLinks';
import { useTableView } from '@/composables/table/useTableView';
import type { TableColumnState } from '@/lib/table/columns';
import type { MeetClub, MeetFilters } from '@/lib/meet';

const SORTABLE = ['name', 'competitorCount', 'startCount'] as const;

// The clubs of the meet with their number of competitors and starts. Selecting a club filters the overview by it.
const props = defineProps<{
  clubs: MeetClub[];
  filters: MeetFilters;
}>();

// The column layout, kept by the page, which shows the column menu next to the search
const columnState = defineModel<TableColumnState>('columnState', {
  required: true,
});

const emit = defineEmits<{
  selectClub: [name: string];
}>();

const { t } = useI18n();
const { clubLink } = useMeetLinks({
  button: resolveComponent('UButton'),
  tooltip: resolveComponent('UTooltip'),
});

const view = useTableView(() => props.clubs, {
  sortable: SORTABLE,
  value: (club, id) => club[id as (typeof SORTABLE)[number]],
  resetOn: () => props.filters,
});

const columns = computed<TableColumn<MeetClub>[]>(() => [
  {
    accessorKey: 'name',
    header: t('pages.meet.clubs.name'),
    enableSorting: true,
    cell: ({ row }) => clubLink(row.original.name, () => emit('selectClub', row.original.name)),
  },
  {
    accessorKey: 'competitorCount',
    header: t('pages.meet.clubs.competitors'),
    enableSorting: true,
    size: 160,
  },
  {
    accessorKey: 'startCount',
    header: t('pages.meet.clubs.starts'),
    enableSorting: true,
    size: 160,
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
    :get-row-id="(club: MeetClub) => club.name"
  >
    <template #empty>
      <TableEmpty icon="i-lucide-users" :text="t('pages.meet.empty')" />
    </template>
  </DataTable>
</template>
