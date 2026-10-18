<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { HeatStatus, type CompetitorResponse } from '@/api/types';
import CollapsibleCard from '@/components/ui/CollapsibleCard.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import HeatTimer from '@/components/timing/HeatTimer.vue';
import MeetHeatTable from '@/components/meet/MeetHeatTable.vue';
import { useHeatLabel } from '@/composables/useHeatLabel';
import { eventName, nextHeats, scheduleDelta, type MeetIndex } from '@/lib/meet';
import { useLiveStore } from '@/stores/live';

// How many heats are listed, the live heat and the ones following it.
const LISTED_HEAT_COUNT = 4;
// A delay below this is shown as on schedule.
const ON_SCHEDULE_MS = 60_000;

// Where the meet stands against its schedule, with the live heat and the heats that follow.
const props = defineProps<{
  index: MeetIndex;
  currentIndex: number;
}>();

const emit = defineEmits<{
  showHeat: [event: number, heat: number];
  showCompetitor: [competitor: CompetitorResponse];
  selectClub: [club: string];
}>();

const { t } = useI18n();
const liveStore = useLiveStore();
const { heatLabel } = useHeatLabel();

// The delay only changes slowly, so it is recomputed every few seconds instead of on every frame.
const now = ref(new Date());
let timer: ReturnType<typeof setInterval> | null = null;
onMounted(() => {
  timer = setInterval(() => (now.value = new Date()), 10_000);
});
onBeforeUnmount(() => {
  if (timer !== null) clearInterval(timer);
});

const current = computed(() => props.index.heats[props.currentIndex] ?? null);
const listed = computed(() =>
  current.value
    ? [current.value, ...nextHeats(props.index, props.currentIndex, LISTED_HEAT_COUNT - 1)]
    : [],
);
const progress = computed(() => props.currentIndex + 1);

const statusColor = computed(() => {
  switch (liveStore.state?.status) {
    case HeatStatus.Start:
      return 'success';
    case HeatStatus.Official:
      return 'info';
    default:
      return 'neutral';
  }
});

const delay = computed(() => {
  const delta = scheduleDelta(props.index, props.currentIndex, now.value);
  if (delta === null) return null;

  const minutes = Math.round(Math.abs(delta) / 60_000);
  if (Math.abs(delta) < ON_SCHEDULE_MS) {
    return {
      label: t('pages.meet.live.onSchedule'),
      color: 'success' as const,
    };
  }
  if (delta > 0) {
    return {
      label: t('pages.meet.live.behind', {
        minutes,
      }),
      color: 'warning' as const,
    };
  }
  return {
    label: t('pages.meet.live.ahead', {
      minutes,
    }),
    color: 'info' as const,
  };
});
</script>

<template>
  <CollapsibleCard
    :title="t('pages.meet.live.title')"
    icon="i-lucide-radio"
    storage-key="meet.live"
  >
    <template #summary>
      <template v-if="current">
        <UBadge
          :label="t(`pages.meet.live.status.${liveStore.state?.status || 'none'}`)"
          :color="statusColor"
          variant="solid"
          class="shrink-0"
        />
        <span class="text-muted hidden truncate text-sm sm:inline">
          {{ current.event.number }} · {{ eventName(current.event) }} ·
          {{ heatLabel(current.heat, current.event.heatCount) }}
        </span>
        <HeatTimer class="text-highlighted ml-auto shrink-0 font-mono font-semibold tabular-nums" />
      </template>
    </template>

    <p v-if="!current" class="text-muted text-sm">{{ t('pages.meet.live.none') }}</p>

    <div v-else class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <div class="text-muted flex flex-wrap items-center justify-between gap-2 text-sm">
          <span>
            {{ t('pages.meet.live.progress', { current: progress, total: index.heats.length }) }}
          </span>
          <span v-if="delay" class="flex items-center gap-1">
            <UBadge :label="delay.label" :color="delay.color" variant="solid" size="sm" />
            <HelpHint :text="t('pages.meet.live.delayHint')" />
          </span>
        </div>
        <UProgress :model-value="progress" :max="index.heats.length" size="sm" />
      </div>

      <MeetHeatTable
        :heats="listed"
        show-event
        :highlight="{ event: current.event.number, heat: current.heat }"
        @show-heat="(event: number, heat: number) => emit('showHeat', event, heat)"
        @show-competitor="(competitor: CompetitorResponse) => emit('showCompetitor', competitor)"
        @select-club="(club: string) => emit('selectClub', club)"
      />
    </div>
  </CollapsibleCard>
</template>
