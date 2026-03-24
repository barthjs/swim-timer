<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import { useRaceStore } from '@/stores/raceStore.ts';

import { formatName } from '@/utils/names.ts';

import { formatSwimtime } from '@/utils/time.ts';

import type { Lane } from '@/types';

interface DisplayLane extends Lane {
  displayId: string;
  removeTimeout: ReturnType<typeof setTimeout>;
}

const store = useRaceStore();

const trackedTimes = ref<Map<number, number>>(new Map());
const displayedLanes = ref<DisplayLane[]>([]);

const allLanes = computed<Lane[]>(() => {
  if (!store.state) return [];
  return Object.values(store.state.lanes)
    .filter((l) => l.time > 0)
    .sort((a, b) => a.time - b.time);
});

const lanesSignature = computed(() => {
  return allLanes.value.map((l) => `${l.number}:${l.time}:${l.meter}`).join('|');
});

watch(
  lanesSignature,
  () => {
    allLanes.value.forEach((lane) => {
      const prevTime = trackedTimes.value.get(lane.number);

      if (prevTime !== lane.time) {
        trackedTimes.value.set(lane.number, lane.time);

        const existingIndex = displayedLanes.value.findIndex((l) => l.number === lane.number);
        if (existingIndex >= 0) {
          const existingLane = displayedLanes.value[existingIndex];
          if (existingLane) {
            clearTimeout(existingLane.removeTimeout);
            displayedLanes.value.splice(existingIndex, 1);
          }
        }

        const displayId = `${lane.number}-${Date.now()}`;
        const removeTimeout = setTimeout(() => {
          displayedLanes.value = displayedLanes.value.filter((l) => l.displayId !== displayId);
        }, 7000);

        const displayLane: DisplayLane = {
          ...lane,
          displayId,
          removeTimeout,
        };

        displayedLanes.value.push(displayLane);
      }
    });
  },
  { immediate: true },
);
</script>

<template>
  <div class="items-start justify-start pt-[5vh] pl-[5vw]">
    <TransitionGroup name="list-compact" tag="div" class="flex flex-col gap-2">
      <div
        v-for="lane in displayedLanes"
        :key="lane.displayId"
        class="flex h-10 text-2xl font-extrabold uppercase"
      >
        <div class="bg-ov-accent flex w-10 items-center justify-center px-6 tabular-nums">
          {{ lane.number }}
        </div>

        <div
          class="bg-ov-secondary flex w-100 items-center justify-between space-x-1 px-6 text-base shadow-lg"
        >
          <span class="block truncate">
            {{ formatName(lane.first_name, lane.last_name) }}
          </span>
          <span class="block">{{ lane.meter }}m</span>
        </div>

        <div
          class="bg-ov-primary flex w-40 items-center justify-center px-6 tracking-wide tabular-nums"
        >
          {{ formatSwimtime(lane.time) }}
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.list-compact-enter-active {
  transition: none;
}

.list-compact-leave-active {
  transition: all 0.4s ease;
}

.list-compact-leave-to {
  opacity: 0;
  transform: translateY(-80px);
}

.list-compact-move {
  transition: all 0.3s ease;
}
</style>
