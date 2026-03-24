<script setup lang="ts">
import { computed } from 'vue';

import { useRaceStore } from '@/stores/raceStore.ts';

import { formatSwimtime } from '@/utils/time.ts';

import { RaceStatus } from '@/types';

const store = useRaceStore();

const sortedLanes = computed(() => {
  if (!store.state) return [];
  const lanes = Object.values(store.state.lanes);

  if (store.state.status === RaceStatus.Official) {
    return [...lanes].sort((a, b) => a.rank - b.rank);
  }

  return lanes.sort((a, b) => a.number - b.number);
});
</script>

<template>
  <div class="items-center justify-center">
    <div
      v-if="
        store.state &&
        Object.keys(store.state.lanes).length > 0 &&
        store.state.status !== RaceStatus.Start
      "
      class="bg-ov-bg-dark flex w-full max-w-7xl flex-col gap-8 rounded-xl p-10 uppercase shadow-2xl"
    >
      <div class="flex items-center justify-center gap-5 text-3xl font-extrabold">
        <span class="text-ov-accent">
          {{ $t('overlays.event', { number: store.state.event }) }}
        </span>
        <span class="bg-ov-divider h-7 w-1"></span>
        <span>{{ store.state.name }}</span>
        <span class="bg-ov-divider h-7 w-1"></span>
        <span>
          {{ $t('overlays.heat', { number: store.state.heat, max: store.state.max_heat }) }}
        </span>
      </div>

      <div class="flex flex-col gap-3">
        <div
          v-for="lane in sortedLanes"
          :key="lane.number"
          :class="[
            'bg-ov-row-bg flex h-16 items-center overflow-hidden rounded-md text-2xl font-bold',
          ]"
        >
          <div
            :class="[
              'bg-ov-primary flex h-full shrink-0 items-center justify-center',
              store.state.status === RaceStatus.Official ? 'w-24' : 'w-20',
            ]"
          >
            <template v-if="store.state.status === RaceStatus.Official">
              {{ lane.rank }}. ({{ lane.number }})
            </template>
            <template v-else>
              {{ lane.number }}
            </template>
          </div>

          <div class="flex grow items-center space-x-2 px-8">
            <span class="block truncate">{{ lane.first_name }} {{ lane.last_name }}</span>
            <span class="block">({{ lane.birthyear }})</span>
          </div>

          <div
            :class="[
              'shrink-0 truncate pr-8 text-right',
              store.state.status === RaceStatus.Official ? 'min-w-48' : 'w-120 truncate',
            ]"
          >
            {{ lane.club }}
          </div>

          <div
            v-if="store.state.status === RaceStatus.Official"
            class="w-40 shrink-0 pr-4 text-center tracking-wide tabular-nums"
          >
            {{ formatSwimtime(lane.time) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
