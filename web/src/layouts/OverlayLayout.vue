<script setup lang="ts">
import { onMounted } from 'vue';

import { useLiveStore } from '@/stores/live';

// Frames the broadcast overlays as a transparent page over the whole viewport,
// fed live by the websocket of the live heat.
const liveStore = useLiveStore();

onMounted(() => {
  liveStore.connect();
});
</script>

<template>
  <RouterView class="overlay-page" />
</template>

<style scoped>
.overlay-page {
  display: flex;
  flex-direction: column;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  pointer-events: none;
  color: var(--overlay-secondary);
}
</style>
