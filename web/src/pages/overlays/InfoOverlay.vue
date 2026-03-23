<script setup lang="ts">
import { computed } from 'vue';

import { useRaceStore } from '@/stores/raceStore.ts';

const store = useRaceStore();

const initialTimeDisplay = computed(() => store.getDisplayTime());
</script>

<template>
  <div class="items-start justify-end pb-[5vh] pl-[5vw]">
    <div v-if="store.state" class="flex flex-col gap-2">
      <div
        id="timer"
        class="bg-ov-primary flex h-14 items-center self-start rounded-md px-6 font-mono text-4xl font-extrabold tracking-wide tabular-nums"
      >
        {{ initialTimeDisplay }}
      </div>

      <div
        class="bg-ov-bg-solid flex h-14 items-center gap-4 rounded-md px-6 text-2xl font-bold uppercase"
      >
        <span class="text-ov-accent">
          {{ $t('overlays.event', { number: store.state.event }) }}
        </span>
        <span class="bg-ov-divider h-5 w-0.5"></span>
        <span>{{ store.state.name }}</span>
        <span class="bg-ov-divider h-5 w-0.5"></span>
        <span>
          {{ $t('overlays.heat', { number: store.state.heat, max: store.state.max_heat }) }}
        </span>
      </div>
    </div>
  </div>
</template>
