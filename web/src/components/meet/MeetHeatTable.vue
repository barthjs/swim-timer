<script setup lang="ts">
import { computed, h, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import type { CompetitorResponse } from '@/api/types';
import { useHeatLabel } from '@/composables/useHeatLabel';
import { useMeetLinks } from '@/composables/useMeetLinks';
import { eventName, type MeetHeat, type MeetStartEntryRow } from '@/lib/meet';
import { tableRowClass } from '@/lib/table/rows';

// Heats stacked below each other, each as a table with one row per lane.
// The tables use fixed column widths, so the columns line up from heat to heat.
// With showEvent, the heading of a heat also names the event and jumps to it.
const props = defineProps<{
  heats: MeetHeat[];
  showEvent?: boolean;
  // The heat to highlight, usually the live heat.
  highlight?: { event: number; heat: number } | null;
}>();

const emit = defineEmits<{
  showCompetitor: [competitor: CompetitorResponse];
  selectClub: [club: string];
  showHeat: [event: number, heat: number];
}>();

const { t } = useI18n();
const { competitorLink, clubLink, eventLink } = useMeetLinks({
  button: resolveComponent('UButton'),
  tooltip: resolveComponent('UTooltip'),
});
const { heatLabel: formatHeat } = useHeatLabel();

const columns = computed<TableColumn<MeetStartEntryRow>[]>(() => [
  {
    id: 'lane',
    header: t('pages.meet.starts.lane'),
    meta: {
      class: {
        th: 'w-20',
        td: 'font-mono text-highlighted',
      },
    },
    cell: ({ row }) => String(row.original.entry.lane),
  },
  {
    id: 'name',
    header: t('pages.meet.starts.name'),
    meta: {
      class: {
        th: 'w-2/5',
        td: 'truncate',
      },
    },
    cell: ({ row }) => {
      const competitor = row.original.competitor;
      if (!competitor)
        return h(
          'span',
          {
            class: 'text-muted',
          },
          t('pages.meet.starts.unknown'),
        );
      return competitorLink(competitor, () => emit('showCompetitor', competitor));
    },
  },
  {
    id: 'birthYear',
    header: t('pages.meet.starts.birthYear'),
    meta: {
      class: {
        th: 'w-28',
      },
    },
    cell: ({ row }) => row.original.competitor?.birthYear ?? '',
  },
  {
    id: 'club',
    header: t('pages.meet.starts.club'),
    meta: {
      class: {
        td: 'truncate',
      },
    },
    cell: ({ row }) => {
      const club = row.original.competitor?.club ?? '';
      return clubLink(club, () => emit('selectClub', club));
    },
  },
]);

/** isHighlighted reports whether a heat is the highlighted one. */
function isHighlighted(heat: MeetHeat): boolean {
  return props.highlight?.event === heat.event.number && props.highlight.heat === heat.heat;
}

/** heatLabel names a heat with its number and the heat count of its event. */
function heatLabel(heat: MeetHeat): string {
  return formatHeat(heat.heat, heat.event.heatCount);
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <section
      v-for="heat in heats"
      :key="`${heat.event.number}-${heat.heat}`"
      class="border-default bg-default scroll-mt-24 overflow-hidden rounded-md border"
      :class="{ 'ring-primary ring-2': isHighlighted(heat) }"
      :id="`meet-heat-${heat.event.number}-${heat.heat}`"
    >
      <header class="border-default bg-elevated/50 flex items-center gap-2 border-b px-4 py-2">
        <component
          :is="
            showEvent
              ? eventLink(
                  `${heat.event.number} · ${eventName(heat.event)} · ${heatLabel(heat)}`,
                  () => emit('showHeat', heat.event.number, heat.heat),
                )
              : h('span', { class: 'text-highlighted font-semibold' }, heatLabel(heat))
          "
        />
        <UBadge
          v-if="isHighlighted(heat)"
          :label="t('pages.meet.events.states.current')"
          color="primary"
          variant="solid"
          size="sm"
        />
      </header>

      <UTable
        :data="heat.entries"
        :columns="columns"
        :get-row-id="(row: MeetStartEntryRow) => row.key"
        :empty="t('pages.meet.noEntries')"
        :meta="{ class: { tr: tableRowClass } }"
        :ui="{ base: 'table-fixed' }"
      />
    </section>
  </div>
</template>
