<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import type { Lane } from '@/api/types';
import { fullName } from '@/lib/names';
import { useLiveStore } from '@/stores/live';

const route = useRoute();
const liveStore = useLiveStore();

const lane = computed<Lane | null>(() => {
  const state = liveStore.state;
  if (!state || !state.status) return null;
  return state.lanes[String(route.params.number)] ?? null;
});

// A lane without a competitor only shows its number.
const name = computed(() =>
  lane.value ? fullName(lane.value.firstName, lane.value.lastName) : '',
);
</script>

<template>
  <div class="bottom-left">
    <Transition appear name="slide-in">
      <div v-if="lane" class="row">
        <div class="number">
          {{ lane.number }}
        </div>

        <div v-if="name" class="name">
          <span class="truncate">{{ name }}</span>
          <span v-if="lane.birthYear" class="year">({{ lane.birthYear }})</span>
        </div>

        <div v-if="lane.club" class="club">
          <span class="truncate">{{ lane.club }}</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.bottom-left {
  align-items: flex-start;
  justify-content: flex-end;
  padding-bottom: 5vh;
  padding-left: 5vw;
}

.row {
  display: flex;
  max-width: 90vw;
  height: 3rem;
  font-size: 1.875rem;
  font-weight: 800;
  text-transform: uppercase;
}

.number {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  min-width: 2.5rem;
  padding: 0 1.5rem;
  background: var(--overlay-accent);
}

.name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
  padding: 0 1.5rem;
  background: var(--overlay-secondary);
  color: var(--overlay-ink);
  box-shadow: var(--overlay-shadow);
}

.year {
  flex-shrink: 0;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.club {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  padding: 0 1.5rem;
  background: var(--overlay-primary);
  white-space: nowrap;
}

.slide-in-enter-active,
.slide-in-leave-active {
  transition:
    transform 0.6s cubic-bezier(0.16, 1, 0.3, 1),
    opacity 0.6s ease;
}

.slide-in-enter-from,
.slide-in-leave-to {
  transform: translateX(-100%);
  opacity: 0;
}
</style>
