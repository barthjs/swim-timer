<script setup lang="ts">
import { computed, onMounted } from 'vue';

import { useRaceStore } from '@/stores/raceStore.ts';

import { formatSwimtime } from '@/utils/time.ts';

const store = useRaceStore();

const initialTimeDisplay = computed(() => store.getDisplayTime());

onMounted(() => {
  store.connect();
});
</script>

<template>
  <div v-if="store.state" class="flex h-screen flex-col text-white">
    <div
      class="bg-base border-ov-divider flex items-center justify-between gap-8 border-b-2 px-6 py-4 text-4xl font-bold uppercase"
    >
      <div>{{ $t('overlays.event', { number: store.state.event }) }} - {{ store.state.name }}</div>
      <div>
        {{ $t('overlays.heat', { number: store.state.heat, max: store.state.max_heat }) }}
      </div>
      <div>{{ $t('raceOverview.lapCount') }}: {{ store.state.lap_count }}</div>
      <div id="timer" class="font-mono tracking-wide tabular-nums">
        {{ initialTimeDisplay }}
      </div>
    </div>

    <header
      class="bg-base border-ov-divider flex shrink-0 gap-4 border-b-2 px-6 py-2 text-2xl font-bold uppercase"
    >
      <div class="w-16 text-center">{{ $t('raceOverview.laneAbbr') }}</div>
      <div class="grow-3 basis-0">{{ $t('raceOverview.name') }}</div>
      <div class="grow-2 basis-0">{{ $t('raceOverview.club') }}</div>
      <div class="w-24 text-center">{{ $t('raceOverview.meter') }}</div>
      <div class="w-40 text-center">{{ $t('raceOverview.time') }}</div>
      <div class="w-20 text-center">{{ $t('raceOverview.rank') }}</div>
    </header>

    <div class="bg-base flex flex-1 flex-col">
      <div
        v-for="lane in store.state.lanes"
        :key="lane.number"
        class="relative flex min-h-0 flex-1 items-center gap-4 px-6 py-3 text-4xl font-extrabold"
        :class="[
          lane.number % 2 === 0 ? 'bg-base' : 'bg-surface',
          lane.finished ? 'text-ov-accent' : '',
        ]"
      >
        <div
          v-if="!lane.finished && store.state.lap_count > 0"
          class="bg-ov-accent absolute bottom-0 left-0 h-1 transition-all duration-500"
          :style="{ width: `${(lane.lap / store.state.lap_count) * 100}%` }"
        ></div>
        <div class="w-16 text-center font-bold">{{ lane.number }}</div>
        <div class="grow-3 basis-0 truncate">
          {{ lane.first_name }} {{ lane.last_name }}
          <span v-if="lane.birthyear">({{ lane.birthyear }})</span>
        </div>
        <div class="grow-2 basis-0 truncate">{{ lane.club }}</div>
        <div class="w-24 text-center font-mono font-bold">{{ lane.meter }}</div>
        <div class="w-40 text-center font-mono font-bold">{{ formatSwimtime(lane.time) }}</div>
        <div class="w-20 text-center font-mono font-bold">{{ lane.rank }}</div>
      </div>
    </div>
  </div>
</template>
