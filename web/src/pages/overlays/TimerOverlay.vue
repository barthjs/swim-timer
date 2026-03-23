<script setup lang="ts">
import { computed } from 'vue';

import { useRaceStore } from '@/stores/raceStore.ts';

const store = useRaceStore();

const initialTimeDisplay = computed(() => store.getDisplayTime());
</script>

<template>
  <div class="items-start justify-end pb-[5vh] pl-[5vw]">
    <div v-if="store.state">
      <div
        id="timer"
        class="bg-ov-primary flex h-14 items-center self-start rounded-md px-6 font-mono text-4xl font-extrabold tracking-wide tabular-nums"
      >
        {{ initialTimeDisplay }}
      </div>
    </div>
  </div>
</template>
