<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { presetsApi } from '@/api/presetsApi.ts';

import { useRaceStore } from '@/stores/raceStore.ts';

import { formatName } from '@/utils/names.ts';
import { formatSwimtime } from '@/utils/time.ts';

import type { Lane, LanePreset, Preset } from '@/types';
import { CornerKey, RaceStatus, TextAnchor } from '@/types';

const preset = ref<Preset | null>(null);
const isLoading = ref(true);

const route = useRoute();

const store = useRaceStore();

const loadPreset = async (presetName: string) => {
  isLoading.value = true;

  try {
    preset.value = await presetsApi.getPreset(presetName);
  } catch (e) {
    console.error(e);
    preset.value = null;
  } finally {
    isLoading.value = false;
  }
};

onMounted(() => {
  // Automatically load the preset defined in the URL parameter to support direct routing
  if (route.params.preset) {
    loadPreset(route.params.preset as string);
  }
});

watch(
  () => route.params.preset,
  (newPreset) => {
    if (newPreset) loadPreset(newPreset as string);
  },
);

const activeLanes = computed(() => {
  if (!preset.value || !store.state) return [];

  const lanesData: Array<{ lane: Lane; style: Record<string, string> }> = [];

  // Total lanes physically available in the pool
  const laneCount = Object.keys(preset.value.lanes).length;

  for (const [laneNumStr, presetLane] of Object.entries(preset.value.lanes)) {
    const visualLaneNum = Number(laneNumStr);

    // Map visual lane number from editor to physical race lane based on preset config
    const raceLaneNum = preset.value.reverseLanes ? laneCount - visualLaneNum + 1 : visualLaneNum;

    const raceLane = store.state.lanes[raceLaneNum];

    if (raceLane) {
      lanesData.push({
        lane: raceLane,
        style: calculateMatrix3d(presetLane),
      });
    }
  }

  return lanesData;
});

// Applies a 2D Homography to map a perfect rectangular virtual space onto the distorted camera perspective
const calculateMatrix3d = (presetLane: LanePreset): Record<string, string> => {
  const x0 = presetLane[CornerKey.TopLeft].x,
    y0 = presetLane[CornerKey.TopLeft].y;
  const x1 = presetLane[CornerKey.TopRight].x,
    y1 = presetLane[CornerKey.TopRight].y;
  const x2 = presetLane[CornerKey.BottomRight].x,
    y2 = presetLane[CornerKey.BottomRight].y;
  const x3 = presetLane[CornerKey.BottomLeft].x,
    y3 = presetLane[CornerKey.BottomLeft].y;

  const dx1 = x1 - x2;
  const dx2 = x3 - x2;
  const dy1 = y1 - y2;
  const dy2 = y3 - y2;
  const sumX = x0 - x1 + x2 - x3;
  const sumY = y0 - y1 + y2 - y3;

  const det = dx1 * dy2 - dx2 * dy1;

  // Prevent division by zero if coordinates form an invalid polygon (e.g., points collapsing into a line)
  if (Math.abs(det) < 1e-6) {
    return { display: 'none' };
  }

  const vWidth = preset.value!.virtualWidth;
  const vHeight = preset.value!.virtualHeight;

  // Compute projection coefficients
  const g = (sumX * dy2 - sumY * dx2) / det;
  const h = (dx1 * sumY - dy1 * sumX) / det;

  const a = x1 - x0 + g * x1;
  const b = x3 - x0 + h * x3;
  const c = x0;
  const d = y1 - y0 + g * y1;
  const e = y3 - y0 + h * y3;
  const f = y0;

  // Normalize coefficients against the virtual resolution to instruct the CSS matrix properly
  const aPrime = a / vWidth;
  const bPrime = b / vHeight;
  const dPrime = d / vWidth;
  const ePrime = e / vHeight;
  const gPrime = g / vWidth;
  const hPrime = h / vHeight;

  const transform = `matrix3d(${aPrime}, ${dPrime}, 0, ${gPrime}, ${bPrime}, ${ePrime}, 0, ${hPrime}, 0, 0, 1, 0, ${c}, ${f}, 0, 1)`;

  return {
    position: 'absolute',
    top: '0',
    left: '0',
    width: `${vWidth}px`,
    height: `${vHeight}px`,
    transformOrigin: '0 0',
    transform: transform,
  };
};
</script>

<template>
  <div class="relative h-screen w-screen">
    <div v-if="!isLoading && preset && store.state">
      <div v-for="item in activeLanes" :key="item.lane.number" class="absolute" :style="item.style">
        <Transition name="fold">
          <div
            v-show="store.state.status !== RaceStatus.Start || item.lane.finished"
            class="bg-ov-row-bg box-border flex h-full w-full items-center gap-10 font-bold"
            :class="[
              preset.textAnchor === TextAnchor.Right
                ? 'origin-right flex-row-reverse pr-0 pl-10'
                : 'origin-left pr-10 pl-0',
            ]"
          >
            <div
              v-if="item.lane.finished"
              class="bg-ov-accent flex h-full w-40 shrink-0 items-center justify-center text-9xl text-base"
            >
              {{ item.lane.rank }}
            </div>

            <div v-else class="flex h-full w-40 shrink-0 items-center justify-center text-9xl">
              {{ item.lane.number }}
            </div>

            <div class="text-9xl whitespace-nowrap uppercase">
              {{ formatName(item.lane.first_name, item.lane.last_name) }}
            </div>

            <div
              v-if="item.lane.finished"
              class="shrink-0 text-9xl tracking-wide tabular-nums"
              :class="[preset.textAnchor === TextAnchor.Right ? 'mr-auto pl-10' : 'ml-auto pr-10']"
            >
              {{ formatSwimtime(item.lane.time) }}
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
