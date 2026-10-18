<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';

import type { Lane } from '@/api/types';
import { formatName } from '@/lib/names';
import { type HeatRun, createSplitTracker, heatRun } from '@/lib/splits';
import { formatSwimtime } from '@/lib/time';
import { useLiveStore } from '@/stores/live';

// How long a split stays visible before it slides out.
const DISPLAY_DURATION_MS = 7_000;

type DisplayLane = Lane & {
  displayId: string;
  removeTimeout: ReturnType<typeof setTimeout>;
};

const liveStore = useLiveStore();

const tracker = createSplitTracker();
const displayedLanes = ref<DisplayLane[]>([]);

const timedLanes = computed<Lane[]>(() => {
  const state = liveStore.state;
  if (!state || !state.status) return [];
  return Object.values(state.lanes)
    .filter((lane) => lane.time > 0)
    .sort((a, b) => a.time - b.time);
});

const currentRun = computed<HeatRun | null>(() => {
  const state = liveStore.state;
  if (!state || !state.status) return null;
  return heatRun(state.event, state.heat, state.startTime);
});

const lanesSignature = computed(() =>
  timedLanes.value.map((lane) => `${lane.number}:${lane.time}:${lane.meter}`).join('|'),
);

/** show appends a lane at the bottom for a fixed duration, replacing it if it is still visible. */
function show(lane: Lane): void {
  const existingIndex = displayedLanes.value.findIndex((shown) => shown.number === lane.number);
  const existing = displayedLanes.value[existingIndex];
  if (existing) {
    clearTimeout(existing.removeTimeout);
    displayedLanes.value.splice(existingIndex, 1);
  }

  const displayId = `${lane.number}-${Date.now()}`;
  const removeTimeout = setTimeout(() => {
    displayedLanes.value = displayedLanes.value.filter((shown) => shown.displayId !== displayId);
  }, DISPLAY_DURATION_MS);

  displayedLanes.value.push({
    ...lane,
    displayId,
    removeTimeout,
  });
}

// Every new time of a lane is shown, see createSplitTracker.
// The times already there when the overlay receives its first heat are only recorded,
// so opening the overlay during a heat shows no old times.
watch(
  [currentRun, lanesSignature],
  () => {
    if (currentRun.value === null) return;
    for (const lane of tracker.update(currentRun.value, timedLanes.value)) {
      show(lane);
    }
  },
  {
    immediate: true,
  },
);

onBeforeUnmount(() => {
  for (const lane of displayedLanes.value) {
    clearTimeout(lane.removeTimeout);
  }
});
</script>

<template>
  <div class="top-left">
    <TransitionGroup name="list-compact" tag="div" class="list">
      <div v-for="lane in displayedLanes" :key="lane.displayId" class="row">
        <div class="number">
          {{ lane.number }}
        </div>

        <div class="name">
          <span class="truncate">
            {{ formatName(lane.firstName, lane.lastName) }}
          </span>
          <span class="meter">{{ lane.meter }}m</span>
        </div>

        <div class="time">
          {{ formatSwimtime(lane.time) }}
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.top-left {
  align-items: flex-start;
  justify-content: flex-start;
  padding-top: 5vh;
  padding-left: 5vw;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.row {
  display: flex;
  height: 2.5rem;
  font-size: 1.5rem;
  font-weight: 800;
  text-transform: uppercase;
}

.number {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 2.5rem;
  padding: 0 1.5rem;
  background: var(--overlay-accent);
  font-variant-numeric: tabular-nums;
}

.name {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.25rem;
  width: 25rem;
  padding: 0 1.5rem;
  background: var(--overlay-secondary);
  color: var(--overlay-ink);
  box-shadow: var(--overlay-shadow);
}

.meter {
  flex-shrink: 0;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.time {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 10rem;
  padding: 0 1.5rem;
  background: var(--overlay-primary);
  letter-spacing: 0.025em;
  font-variant-numeric: tabular-nums;
}

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
