<script setup lang="ts">
import { onMounted } from 'vue';
import { useI18n } from 'vue-i18n';

import type { LiveHeat } from '@/api/types';
import HeatTimer from '@/components/timing/HeatTimer.vue';
import { useHeatLabel } from '@/composables/useHeatLabel';
import { hasRank } from '@/lib/heat';
import { eventName } from '@/lib/meet';
import { fullName } from '@/lib/names';
import { formatSwimtime } from '@/lib/time';
import { useLiveStore } from '@/stores/live';

// Full-screen scoreboard for the operators, with every lane sharing the screen height.
const { t } = useI18n();
const liveStore = useLiveStore();
const { heatLabel } = useHeatLabel();

onMounted(() => {
  liveStore.connect();
});

/** eventLabel names the event of the live heat, without a name if the event is missing from the meet data. */
function eventLabel(state: LiveHeat): string {
  return [
    t('overlays.event', {
      number: state.event,
    }),
    eventName(state),
  ]
    .filter(Boolean)
    .join(' - ');
}
</script>

<template>
  <div
    v-if="liveStore.state && liveStore.state.status"
    class="bg-default text-highlighted flex h-screen flex-col"
  >
    <div
      class="border-default flex items-center justify-between gap-8 border-b-2 px-6 py-4 text-4xl font-bold uppercase"
    >
      <div>{{ eventLabel(liveStore.state) }}</div>
      <div>
        {{ heatLabel(liveStore.state.heat, liveStore.state.heatCount) }}
      </div>
      <div>{{ t('pages.scoreboard.lapCount') }}: {{ liveStore.state.lapCount }}</div>
      <HeatTimer class="font-mono tracking-wide tabular-nums" />
    </div>

    <header
      class="border-default flex shrink-0 gap-4 border-b-2 px-6 py-2 text-2xl font-bold uppercase"
    >
      <div class="w-16 text-center">{{ t('pages.scoreboard.laneAbbr') }}</div>
      <div class="grow-3 basis-0">{{ t('pages.scoreboard.name') }}</div>
      <div class="grow-2 basis-0">{{ t('pages.scoreboard.club') }}</div>
      <div class="w-24 text-center">{{ t('pages.scoreboard.meter') }}</div>
      <div class="w-40 text-center">{{ t('pages.scoreboard.time') }}</div>
      <div class="w-20 text-center">{{ t('pages.scoreboard.rank') }}</div>
    </header>

    <div class="flex flex-1 flex-col">
      <div
        v-for="lane in liveStore.state.lanes"
        :key="lane.number"
        class="relative flex min-h-0 flex-1 items-center gap-4 px-6 py-3 text-4xl font-extrabold"
        :class="[
          lane.number % 2 === 0 ? 'bg-default' : 'bg-elevated',
          lane.finished ? 'text-warning' : '',
        ]"
      >
        <!-- Lap progress of a lane that has not finished yet. -->
        <div
          v-if="!lane.finished && liveStore.state.lapCount > 0"
          class="bg-warning absolute bottom-0 left-0 h-1 transition-all duration-500"
          :style="{ width: `${(lane.lap / liveStore.state.lapCount) * 100}%` }"
        ></div>
        <div class="w-16 text-center font-bold">{{ lane.number }}</div>
        <div class="grow-3 basis-0 truncate">
          {{ fullName(lane.firstName, lane.lastName) }}
          <span v-if="lane.birthYear">({{ lane.birthYear }})</span>
        </div>
        <div class="grow-2 basis-0 truncate">{{ lane.club }}</div>
        <div class="w-24 text-center font-mono font-bold">{{ lane.meter }}</div>
        <div class="w-40 text-center font-mono font-bold">{{ formatSwimtime(lane.time) }}</div>
        <div class="w-20 text-center font-mono font-bold">
          <template v-if="hasRank(lane)">{{ lane.rank }}</template>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="bg-default text-muted flex h-screen items-center justify-center text-4xl">
    {{ t('pages.scoreboard.noHeat') }}
  </div>
</template>
