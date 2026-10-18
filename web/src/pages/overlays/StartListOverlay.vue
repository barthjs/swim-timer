<script setup lang="ts">
import { computed } from 'vue';

import { HeatStatus } from '@/api/types';
import HeatHeading from '@/components/overlay/HeatHeading.vue';
import { compareResults, isResultRanked } from '@/lib/heat';
import { fullName } from '@/lib/names';
import { formatSwimtime } from '@/lib/time';
import { useLiveStore } from '@/stores/live';

const liveStore = useLiveStore();

const official = computed(() => liveStore.state?.status === HeatStatus.Official);

// The start list is hidden before the first heat and while the heat runs, and shows the results once it is official.
const visible = computed(
  () =>
    !!liveStore.state &&
    !!liveStore.state.status &&
    Object.keys(liveStore.state.lanes).length > 0 &&
    liveStore.state.status !== HeatStatus.Start,
);

// Official results are ordered by rank, the start list by lane, see compareResults.
const sortedLanes = computed(() => {
  if (!liveStore.state) return [];
  const lanes = Object.values(liveStore.state.lanes);

  if (!official.value) {
    return lanes.sort((a, b) => a.number - b.number);
  }
  return lanes.sort(compareResults);
});
</script>

<template>
  <div class="center">
    <div v-if="visible && liveStore.state" class="panel">
      <div class="heading">
        <HeatHeading :state="liveStore.state" size="large" show-final />
      </div>

      <div class="rows">
        <div v-for="lane in sortedLanes" :key="lane.number" class="row">
          <div class="number" :class="{ official }">
            <template v-if="official && isResultRanked(lane)"
              >{{ lane.rank }}. ({{ lane.number }})</template
            >
            <template v-else-if="official">({{ lane.number }})</template>
            <template v-else>{{ lane.number }}</template>
          </div>

          <div class="name">
            <span class="truncate">{{ fullName(lane.firstName, lane.lastName) }}</span>
            <span v-if="lane.birthYear" class="year">({{ lane.birthYear }})</span>
          </div>

          <div class="club truncate" :class="{ official }">
            {{ lane.club }}
          </div>

          <div v-if="official" class="time">
            <template v-if="lane.finished">{{ formatSwimtime(lane.time) }}</template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.center {
  align-items: center;
  justify-content: center;
}

.panel {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  width: 100%;
  max-width: 80rem;
  padding: 2.5rem;
  border-radius: 0.75rem;
  background: var(--overlay-panel);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  text-transform: uppercase;
}

.heading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.25rem;
  font-size: 1.875rem;
  font-weight: 800;
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.row {
  display: flex;
  align-items: center;
  height: 4rem;
  overflow: hidden;
  border-radius: 0.375rem;
  background: var(--overlay-row);
  font-size: 1.5rem;
  font-weight: 700;
}

.number {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 5rem;
  height: 100%;
  background: var(--overlay-primary);
}

.number.official {
  width: 6rem;
}

.name {
  display: flex;
  flex-grow: 1;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  padding: 0 2rem;
}

.year {
  flex-shrink: 0;
}

.club {
  flex-shrink: 0;
  width: 30rem;
  padding-right: 2rem;
  text-align: right;
}

.club.official {
  width: auto;
  min-width: 12rem;
  max-width: 30rem;
}

.time {
  flex-shrink: 0;
  width: 10rem;
  padding-right: 1rem;
  text-align: center;
  letter-spacing: 0.025em;
  font-variant-numeric: tabular-nums;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
