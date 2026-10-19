<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import type { CompetitorResponse, EventResponse } from '@/api/types';
import { useApiError } from '@/composables/useApiError';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import HelpHint from '@/components/form/HelpHint.vue';
import MeetClubsTable from '@/components/meet/MeetClubsTable.vue';
import MeetEventsTable, { type EventRow } from '@/components/meet/MeetEventsTable.vue';
import MeetFilterBar from '@/components/meet/MeetFilterBar.vue';
import MeetLiveCard from '@/components/meet/MeetLiveCard.vue';
import MeetStartEntriesTable from '@/components/meet/MeetStartEntriesTable.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import { useMeetFilters } from '@/composables/useMeetFilters';
import {
  currentHeatIndex,
  eventName,
  eventState,
  filterClubs,
  filterEvents,
  filterRows,
} from '@/lib/meet';
import { useLiveStore } from '@/stores/live';
import { useMeetStore } from '@/stores/meet';

// How many animation frames a jump waits at most for its target to be rendered.
const MAX_SCROLL_ATTEMPTS = 30;

// Below sm, a header button only shows its leading icon, so the title and the buttons fit in one row.
const COMPACT_BUTTON_UI = {
  base: 'max-sm:p-1.5',
  label: 'max-sm:hidden',
  trailingIcon: 'max-sm:hidden',
};

const EVENT_SORTABLE = ['number', 'schedule', 'name', 'heatCount', 'startCount'] as const;

// The overview of the meet, searched and filtered in the browser on the loaded meet.
const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const { messageFor } = useApiError();

const meetStore = useMeetStore();
const liveStore = useLiveStore();
const { filters, reset } = useMeetFilters();

onMounted(() => meetStore.start());

// The column layouts of the tables live here, so the column menu of the shown table can sit next to the search
const eventColumnState = useTableColumnState('meet-events');
const startColumnState = useTableColumnState('meet-starts');
const clubColumnState = useTableColumnState('meet-clubs');
const eventsTable = useTemplateRef<InstanceType<typeof MeetEventsTable>>('eventsTable');
const startsTable = useTemplateRef<InstanceType<typeof MeetStartEntriesTable>>('startsTable');
const clubsTable = useTemplateRef<InstanceType<typeof MeetClubsTable>>('clubsTable');

const TABS = ['events', 'starts', 'clubs'] as const;
type Tab = (typeof TABS)[number];

// Each tab has its own columns, so switching tabs starts without sorting on the first page.
const tab = computed<Tab>({
  get: () => TABS.find((value) => value === route.query.tab) ?? 'events',
  set: (value) => {
    void router.replace({
      query: {
        ...route.query,
        tab: value === 'events' ? undefined : value,
        sort: undefined,
        page: undefined,
      },
    });
  },
});

const tabs = computed(() => [
  {
    label: t('pages.meet.tabs.events'),
    icon: 'i-lucide-list-ordered',
    value: 'events',
  },
  {
    label: t('pages.meet.tabs.starts'),
    icon: 'i-lucide-users',
    value: 'starts',
  },
  {
    label: t('pages.meet.tabs.clubs'),
    icon: 'i-lucide-shield',
    value: 'clubs',
  },
]);

const index = computed(() => meetStore.index);
const currentIndex = computed(() =>
  index.value ? currentHeatIndex(index.value, liveStore.state) : -1,
);
const liveEvent = computed(() => index.value?.heats[currentIndex.value]?.event.number ?? null);
const liveHeat = computed(() => index.value?.heats[currentIndex.value]?.heat ?? null);

/** stateOf tells whether an event lies before, at, or after the live heat. */
function stateOf(event: EventResponse) {
  return index.value ? eventState(index.value, event, currentIndex.value) : null;
}

const events = computed(() => (index.value ? filterEvents(index.value, filters, stateOf) : []));
const rows = computed(() => (index.value ? filterRows(index.value, filters, stateOf) : []));
const clubs = computed(() => (index.value ? filterClubs(index.value, filters) : []));

const eventRows = computed<EventRow[]>(() =>
  events.value.map((event) => ({
    number: event.number,
    schedule: index.value?.positionsByEvent.get(event.number) ?? 0,
    date: event.date,
    time: event.time,
    name: eventName(event),
    heatCount: event.heatCount,
    startCount: index.value?.rowsByEvent.get(event.number)?.length ?? 0,
    state: stateOf(event),
  })),
);

const eventView = useTableView(eventRows, {
  sortable: EVENT_SORTABLE,
  value: (row, id) => row[id as (typeof EVENT_SORTABLE)[number]],
  resetOn: () => filters,
});

const expanded = ref<Record<string, boolean>>({});

// The event of the live heat opens once it is known, so the current lanes are visible without a click
watch(
  liveEvent,
  (event) => {
    if (event !== null)
      expanded.value = {
        ...expanded.value,
        [String(event)]: true,
      };
  },
  {
    immediate: true,
  },
);

const loadedAt = computed(() => {
  const value = meetStore.meet?.loadedAt;
  if (!value) return null;
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(new Date(value));
});

/**
 * scrollTo scrolls to the first of the elements that exists,
 * waiting a few frames for the new page, tab, or expanded row to render.
 */
function scrollTo(ids: string[], attempt = 0): void {
  for (const id of ids) {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      });
      return;
    }
  }
  if (attempt < MAX_SCROLL_ATTEMPTS) {
    requestAnimationFrame(() => scrollTo(ids, attempt + 1));
  }
}

/**
 * showHeat opens an event in the event list on the page that holds it and scrolls to the heat.
 * Filters are only cleared if they hide the event.
 */
function showHeat(event: number, heat: number): void {
  if (!events.value.some((item) => item.number === event)) {
    reset();
  }
  tab.value = 'events';
  expanded.value = {
    ...expanded.value,
    [String(event)]: true,
  };

  // The tab switch clears the sorting first, so the page is looked up once the URL is updated
  requestAnimationFrame(() => {
    eventView.goTo((row) => row.number === event);
    scrollTo([`meet-heat-${event}-${heat}`, `meet-event-${event}`]);
  });
}

/** showCompetitor filters the overview by a competitor: its starts and the events it swims in. */
function showCompetitor(competitor: CompetitorResponse): void {
  reset();
  filters.query = [competitor.lastName, competitor.firstName].filter(Boolean).join(' ');
  tab.value = 'starts';
}

/** selectClub filters the overview by a club: its starts and the events its competitors swim in. */
function selectClub(name: string): void {
  reset();
  filters.clubs = [name];
  tab.value = 'starts';
}
</script>

<template>
  <!-- From lg up the page fills the viewport, so the table scrolls on its own and keeps its header in view. -->
  <div class="flex flex-col gap-4 lg:min-h-0 lg:flex-1">
    <!--
      On small screens the buttons only show their icon next to the title, with the label in the tooltip,
      and the time of the meet data takes its own row below. From sm up, the time stands next to the buttons.
    -->
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
      <div class="flex items-center gap-1">
        <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.meet.title') }}</h1>
        <DocsLink page="meet" />
      </div>
      <span
        v-if="loadedAt"
        class="text-muted order-last flex w-full items-center gap-1 text-sm sm:order-none sm:ml-auto sm:w-auto"
      >
        {{ t('pages.meet.loadedAt', { time: loadedAt }) }}
        <HelpHint :text="t('pages.meet.loadedAtHint')" />
      </span>
      <div class="ml-auto flex items-center gap-2" :class="{ 'sm:ml-0': loadedAt }">
        <UTooltip :text="t('pages.meet.currentHeatHint')">
          <UButton
            :label="t('pages.meet.currentHeat')"
            :aria-label="t('pages.meet.currentHeat')"
            icon="i-lucide-locate-fixed"
            :ui="COMPACT_BUTTON_UI"
            :disabled="liveEvent === null || liveHeat === null"
            @click="liveEvent !== null && liveHeat !== null && showHeat(liveEvent, liveHeat)"
          />
        </UTooltip>
        <UTooltip :text="t('pages.meet.scoreboardHint')">
          <UButton
            :label="t('pages.meet.scoreboard')"
            :aria-label="t('pages.meet.scoreboard')"
            icon="i-lucide-monitor"
            trailing-icon="i-lucide-external-link"
            :ui="COMPACT_BUTTON_UI"
            color="neutral"
            variant="outline"
            :to="{ name: 'scoreboard' }"
            target="_blank"
          />
        </UTooltip>
      </div>
    </div>

    <div v-if="meetStore.loading && !index" class="flex flex-col gap-4">
      <USkeleton class="h-32 w-full" />
      <USkeleton class="h-64 w-full" />
    </div>

    <UAlert
      v-else-if="meetStore.error && !index"
      :title="t('pages.meet.loadError')"
      :description="messageFor(meetStore.error)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <UAlert
      v-else-if="index && !index.events.length"
      :title="t('pages.meet.noData')"
      icon="i-lucide-info"
      color="info"
    />

    <template v-else-if="index">
      <!-- The card keeps its height in the filled viewport, the table below takes the remaining space -->
      <MeetLiveCard
        class="shrink-0"
        :index="index"
        :current-index="currentIndex"
        @show-heat="showHeat"
        @show-competitor="showCompetitor"
        @select-club="selectClub"
      />

      <MeetFilterBar v-model="filters" :index="index" @reset="reset">
        <template #actions>
          <TableColumnMenu
            v-if="tab === 'events' && eventsTable"
            v-model:column-state="eventColumnState"
            :columns="eventsTable.columns"
          />
          <TableColumnMenu
            v-else-if="tab === 'starts' && startsTable"
            v-model:column-state="startColumnState"
            :columns="startsTable.columns"
          />
          <TableColumnMenu
            v-else-if="tab === 'clubs' && clubsTable"
            v-model:column-state="clubColumnState"
            :columns="clubsTable.columns"
          />
        </template>
      </MeetFilterBar>

      <UTabs v-model="tab" :items="tabs" class="w-full" :content="false" />

      <MeetEventsTable
        v-if="tab === 'events'"
        ref="eventsTable"
        class="lg:min-h-80"
        v-model:column-state="eventColumnState"
        v-model:sorting="eventView.sorting.value"
        v-model:page="eventView.page.value"
        v-model:page-size="eventView.pageSize.value"
        v-model:expanded="expanded"
        :index="index"
        :rows="eventView.visible.value"
        :total="eventView.total.value"
        :live-event="liveEvent"
        :live-heat="liveHeat"
        @show-competitor="showCompetitor"
        @select-club="selectClub"
      />
      <MeetStartEntriesTable
        v-else-if="tab === 'starts'"
        ref="startsTable"
        class="lg:min-h-80"
        v-model:column-state="startColumnState"
        :rows="rows"
        :filters="filters"
        @show-competitor="showCompetitor"
        @select-club="selectClub"
        @show-heat="showHeat"
      />
      <MeetClubsTable
        v-else
        ref="clubsTable"
        class="lg:min-h-80"
        v-model:column-state="clubColumnState"
        :clubs="clubs"
        :filters="filters"
        @select-club="selectClub"
      />
    </template>
  </div>
</template>
