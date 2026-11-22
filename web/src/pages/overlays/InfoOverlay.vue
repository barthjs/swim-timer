<script setup lang="ts">
import { computed } from 'vue';

import HeatHeading from '@/components/overlay/HeatHeading.vue';
import RecordBannerContent from '@/components/overlay/RecordBannerContent.vue';
import HeatTimer from '@/components/timing/HeatTimer.vue';
import { useRecordAnnouncements } from '@/composables/record/useRecordAnnouncements';
import { recordBanners, recordHeat } from '@/lib/record/recordBanner';
import { useLiveStore } from '@/stores/live';

// The running time with the event and the heat on the left, and every record of the heat on the right,
// one after another as soon as its lane touches.
const liveStore = useLiveStore();

const { current } = useRecordAnnouncements(
  computed(() => recordBanners(liveStore.state)),
  computed(() => recordHeat(liveStore.state)),
);
</script>

<template>
  <div class="bottom">
    <div class="row">
      <div v-if="liveStore.state && liveStore.state.status" class="stack">
        <HeatTimer class="timer" />

        <div class="bar">
          <HeatHeading :state="liveStore.state" size="small" />
        </div>
      </div>

      <Transition appear name="slide-in" mode="out-in">
        <RecordBannerContent v-if="current" :key="current.key" :banner="current" class="record" />
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.bottom {
  justify-content: flex-end;
  padding-right: 5vw;
  padding-bottom: 5vh;
  padding-left: 5vw;
}

.row {
  display: flex;
  align-items: flex-end;
  gap: 2rem;
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

.record {
  margin-left: auto;
}

.slide-in-enter-active,
.slide-in-leave-active {
  transition:
    transform 0.6s cubic-bezier(0.16, 1, 0.3, 1),
    opacity 0.6s ease;
}

.slide-in-enter-from,
.slide-in-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
