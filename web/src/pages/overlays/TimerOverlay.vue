<script setup lang="ts">
import HeatTimer from '@/components/timing/HeatTimer.vue';
import { useLiveStore } from '@/stores/live';

const liveStore = useLiveStore();
</script>

<template>
  <div class="bottom-left">
    <div v-if="liveStore.state && liveStore.state.status">
      <HeatTimer class="timer" />
    </div>
  </div>
</template>

<style scoped>
.bottom-left {
  align-items: flex-start;
  justify-content: flex-end;
  padding-bottom: 5vh;
  padding-left: 5vw;
}

.timer {
  display: flex;
  align-items: center;
  height: 3.5rem;
  padding: 0 1.5rem;
  border-radius: 0.375rem;
  background: var(--overlay-primary);
  font-family: var(--overlay-font-mono), monospace;
  font-size: 2.25rem;
  font-weight: 800;
  letter-spacing: 0.025em;
  font-variant-numeric: tabular-nums;
}
</style>
