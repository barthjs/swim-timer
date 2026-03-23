<script setup lang="ts">
import { onMounted } from 'vue';

import { useRaceStore } from '@/stores/raceStore';

const store = useRaceStore();

onMounted(() => {
  store.connect();
});
</script>

<template>
  <RouterView
    class="text-ov-secondary pointer-events-none box-border flex h-screen w-screen flex-col overflow-hidden bg-transparent"
  />
</template>
