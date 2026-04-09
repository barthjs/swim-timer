<script setup lang="ts">
import type { LanePreset, Preset } from '@/types';
import { CornerKey } from '@/types';

defineProps<{
  currentPreset: Preset | null;
  selectedLanes: Set<number>;
  activeLaneNum: number | null;
  activeLane: LanePreset | null;
}>();

const emit = defineEmits<{
  (e: 'lane-down', event: MouseEvent, laneNum: number): void;
  (e: 'corner-down', event: MouseEvent, laneNum: number, corner: CornerKey): void;
}>();
</script>

<template>
  <div class="pool-canvas-wrapper">
    <template v-if="currentPreset">
      <svg class="lane-svg">
        <polygon
          v-for="(lane, laneNum) in currentPreset.lanes"
          :key="laneNum"
          :points="`${lane[CornerKey.TopLeft].x},${lane[CornerKey.TopLeft].y} ${lane[CornerKey.TopRight].x},${lane[CornerKey.TopRight].y} ${lane[CornerKey.BottomRight].x},${lane[CornerKey.BottomRight].y} ${lane[CornerKey.BottomLeft].x},${lane[CornerKey.BottomLeft].y}`"
          :class="{
            selected: selectedLanes.has(Number(laneNum)),
            locked: lane.locked,
          }"
          @mousedown="emit('lane-down', $event, Number(laneNum))"
        />
      </svg>

      <template v-if="activeLaneNum && activeLane && !activeLane.locked">
        <div
          v-for="corner in Object.values(CornerKey)"
          :key="corner"
          class="handle"
          :style="{
            left: activeLane[corner].x + 'px',
            top: activeLane[corner].y + 'px',
          }"
          @mousedown.left="emit('corner-down', $event, activeLaneNum, corner)"
        ></div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.pool-canvas-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  pointer-events: none;
}

.lane-svg {
  width: 100%;
  height: 100%;
  pointer-events: none;

  polygon {
    fill: rgba(234, 179, 8, 0.5);
    stroke: #eab308;
    stroke-width: 1.5;
    pointer-events: all;
    cursor: grab;
    transition:
      fill 0.2s,
      stroke 0.2s;
  }

  polygon.locked {
    fill: rgba(100, 116, 139, 0.15);
    stroke: #64748b;
    cursor: not-allowed;
    stroke-dasharray: 4;
  }

  polygon:active:not(.locked) {
    cursor: grabbing;
  }

  polygon.selected {
    fill: rgba(236, 72, 153, 0.5);
    stroke: #ec4899;
    stroke-width: 2;
  }
}

.handle {
  position: absolute;
  width: 12px;
  height: 12px;
  background: white;
  border: 2px solid #ec4899;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  cursor: crosshair;
  z-index: 10;
  pointer-events: all;
  box-shadow: 0 0 4px rgba(0, 0, 0, 0.5);
}
</style>
