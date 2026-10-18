<script setup lang="ts">
import HeatHeading from '@/components/overlay/HeatHeading.vue';
import HeatTimer from '@/components/timing/HeatTimer.vue';
import { useLiveStore } from '@/stores/live';

const liveStore = useLiveStore();
</script>

<template>
  <div class="bottom-left">
    <div v-if="liveStore.state && liveStore.state.status" class="stack">
      <HeatTimer class="timer" />

      <div class="bar">
        <HeatHeading :state="liveStore.state" size="small" />
      </div>
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

.stack {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.timer {
  display: flex;
  align-self: flex-start;
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

.bar {
  display: flex;
  align-items: center;
  gap: 1rem;
  height: 3.5rem;
  padding: 0 1.5rem;
  border-radius: 0.375rem;
  background: var(--overlay-bar);
  font-size: 1.5rem;
  font-weight: 700;
  text-transform: uppercase;
}
</style>
