<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { presetRepository } from '@/api/repositories/preset';
import { HeatStatus, type Lane } from '@/api/types/live';
import type { PresetResponse } from '@/api/types/preset';
import ArLaneContent from '@/components/overlay/ArLaneContent.vue';
import { useRealtimeChange } from '@/composables/realtime/useRealtimeChange';
import { useRetry } from '@/composables/ui/useRetry';
import {
  heatLaneNumber,
  laneStyle,
  liveFirstLane,
  presetLaneCount,
  toPixelLane,
} from '@/lib/preset/lanes';
import { useLiveStore } from '@/stores/live';

const route = useRoute();
const liveStore = useLiveStore();

const preset = ref<PresetResponse | null>(null);
const presetName = computed(() =>
  typeof route.params.preset === 'string' ? route.params.preset : '',
);

// The overlay runs unattended in the streaming software, so it stays empty instead of showing an error
// and loads the preset again until it arrives, and again whenever it changes on the server.
const presetLoad = useRetry(
  async () => presetRepository.get(presetName.value),
  (result) => {
    preset.value = result;
  },
);

watch(
  presetName,
  (name) => {
    preset.value = null;
    if (name) {
      presetLoad.start();
      return;
    }
    presetLoad.stop();
  },
  {
    immediate: true,
  },
);

useRealtimeChange('preset', (name) => {
  if (presetName.value && (name === undefined || name === presetName.value)) presetLoad.start();
});

// The preset stores the lanes as fractions of the image,
// which the overlay draws at the size of its window, the browser source.
// A plain resize listener keeps the overlay bundle free of helpers that might need a newer browser.
const windowSize = reactive({
  width: window.innerWidth,
  height: window.innerHeight,
});

/** Stores the current size of the window. */
function onResize(): void {
  windowSize.width = window.innerWidth;
  windowSize.height = window.innerHeight;
}

onMounted(() => {
  window.addEventListener('resize', onResize);
  onResize();
});

onUnmounted(() => {
  window.removeEventListener('resize', onResize);
});

// Maps each lane drawn in the editor to its heat lane, see heatLaneNumber
const activeLanes = computed(() => {
  const state = liveStore.state;
  if (!preset.value || !state?.status) return [];

  const current = preset.value;
  const laneCount = presetLaneCount(current.lanes);
  const firstLane = liveFirstLane(state);
  const result: { lane: Lane; style: Record<string, string> }[] = [];

  for (const [presetLaneKey, presetLane] of Object.entries(current.lanes)) {
    const heatLane = heatLaneNumber(
      Number(presetLaneKey),
      laneCount,
      firstLane,
      current.reverseLanes,
    );

    const lane = state.lanes[String(heatLane)];
    if (lane) {
      result.push({
        lane,
        style: laneStyle(toPixelLane(presetLane, windowSize)),
      });
    }
  }

  return result;
});
</script>

<template>
  <div class="canvas">
    <div v-if="preset && liveStore.state && liveStore.state.status">
      <div v-for="item in activeLanes" :key="item.lane.number" class="lane" :style="item.style">
        <!-- During the heat, a lane only appears once its competitor has finished. -->
        <Transition name="fold">
          <ArLaneContent
            v-show="liveStore.state.status !== HeatStatus.Start || item.lane.finished"
            :lane="item.lane"
            :text-anchor="preset.textAnchor"
            :text-size="preset.textSize"
          />
        </Transition>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* The lanes are placed in pixels of the window, the content of a lane is scaled by its transform. */
.canvas {
  position: relative;
  width: 100vw;
  height: 100vh;
}

.lane {
  position: absolute;
}

.fold-enter-active,
.fold-leave-active {
  transition:
    transform 0.5s cubic-bezier(0.16, 1, 0.3, 1),
    opacity 0.5s ease;
}

.fold-enter-from,
.fold-leave-to {
  transform: scaleX(0);
  opacity: 0;
}
</style>
