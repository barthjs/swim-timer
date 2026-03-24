<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { useRaceStore } from '@/stores/raceStore.ts';

import type { Lane } from '@/types';

const route = useRoute();
const store = useRaceStore();

const laneData = computed<Lane | null>(() => {
  if (!store.state) return null;
  const laneNum = Number(route.params.number as string);
  return store.state.lanes[laneNum] ?? null;
});
</script>

<template>
  <div class="items-start justify-end pb-[5vh] pl-[5vw]">
    <Transition appear name="slide-in">
      <div v-if="laneData" class="flex h-12 text-3xl font-extrabold uppercase">
        <div class="bg-ov-accent flex w-10 items-center justify-center px-6">
          {{ laneData.number }}
        </div>

        <div class="bg-ov-secondary flex items-center space-x-2 px-6 text-base shadow-lg">
          <span class="block truncate">{{ laneData.first_name }} {{ laneData.last_name }}</span>
          <span class="block">({{ laneData.birthyear }})</span>
        </div>

        <div class="bg-ov-primary flex items-center justify-center px-6 whitespace-nowrap">
          {{ laneData.club }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
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
