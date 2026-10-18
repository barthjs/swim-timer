<script setup lang="ts">
import { computed } from 'vue';

import type { CornerKey } from '@/api/types';
import {
  CORNERS,
  heatLaneNumber,
  isLaneValid,
  laneCenter,
  polygonPoints,
  presetLaneCount,
  type PixelLane,
} from '@/lib/lanes';

// Draws the lanes of a preset on the canvas in pixels of the reference resolution,
// with the corner handles of the single selected, unlocked lane. A label also names the lane of the heat
// the AR overlay shows there if it differs.
const props = defineProps<{
  lanes: Record<string, PixelLane>;
  selectedLanes: ReadonlySet<string>;
  activeLaneKey: string | null;
  reverseLanes: boolean;
  firstLane: number;
  // Draws only the outlines, so the preview underneath stays visible.
  outlineOnly: boolean;
}>();

const emit = defineEmits<{
  laneDown: [event: PointerEvent, laneKey: string];
  cornerDown: [event: PointerEvent, laneKey: string, corner: CornerKey];
}>();

const laneCount = computed(() => presetLaneCount(props.lanes));

const labels = computed(() =>
  Object.entries(props.lanes).map(([key, lane]) => {
    const heatLane = heatLaneNumber(
      Number(key),
      laneCount.value,
      props.firstLane,
      props.reverseLanes,
    );
    return {
      key,
      center: laneCenter(lane),
      text: heatLane === Number(key) ? key : `${key} (${heatLane})`,
    };
  }),
);

// The lane whose corner handles are shown, if it can be edited.
const handleLane = computed(() => {
  if (!props.activeLaneKey) return null;
  const lane = props.lanes[props.activeLaneKey];
  return lane && !lane.locked ? lane : null;
});
</script>

<template>
  <!-- touch-none keeps a finger on a lane or a handle from scrolling the page instead of dragging. -->
  <div class="pointer-events-none absolute inset-0 touch-none">
    <svg class="size-full overflow-visible">
      <polygon
        v-for="(lane, laneKey) in lanes"
        :key="laneKey"
        :points="polygonPoints(lane)"
        class="lane"
        :class="{
          selected: selectedLanes.has(String(laneKey)),
          locked: lane.locked,
          invalid: !isLaneValid(lane),
          'outline-only': outlineOnly,
        }"
        @pointerdown.left="emit('laneDown', $event, String(laneKey))"
      />
      <text
        v-for="label in labels"
        :key="`label-${label.key}`"
        :x="label.center.x"
        :y="label.center.y"
        class="label"
      >
        {{ label.text }}
      </text>
    </svg>

    <template v-if="activeLaneKey && handleLane">
      <div
        v-for="corner in CORNERS"
        :key="corner"
        class="handle"
        :style="{
          left: `${handleLane[corner].x}px`,
          top: `${handleLane[corner].y}px`,
        }"
        @pointerdown.left.stop="emit('cornerDown', $event, activeLaneKey, corner)"
      ></div>
    </template>
  </div>
</template>

<style scoped>
.lane {
  fill: color-mix(in srgb, var(--color-editor-lane) 50%, transparent);
  stroke: var(--color-editor-lane);
  stroke-width: calc(1.5 * var(--editor-px));
  pointer-events: all;
  cursor: grab;
  transition:
    fill 0.2s,
    stroke 0.2s;
}

.lane.locked {
  fill: color-mix(in srgb, var(--color-editor-lane-locked) 15%, transparent);
  stroke: var(--color-editor-lane-locked);
  stroke-dasharray: calc(4 * var(--editor-px));
  cursor: not-allowed;
}

.lane:active:not(.locked) {
  cursor: grabbing;
}

.lane.selected {
  fill: color-mix(in srgb, var(--color-editor-lane-selected) 50%, transparent);
  stroke: var(--color-editor-lane-selected);
  stroke-width: calc(2 * var(--editor-px));
}

.lane.invalid {
  fill: color-mix(in srgb, var(--color-editor-lane-invalid) 50%, transparent);
  stroke: var(--color-editor-lane-invalid);
  stroke-width: calc(2 * var(--editor-px));
}

/* Keeps the polygon clickable while the preview underneath stays visible. */
.lane.outline-only {
  fill: transparent;
}

.label {
  fill: var(--color-editor-handle);
  stroke: var(--color-editor-label-outline);
  stroke-width: calc(3 * var(--editor-px));
  paint-order: stroke;
  font-size: calc(14 * var(--editor-px));
  font-weight: 700;
  text-anchor: middle;
  dominant-baseline: central;
  pointer-events: none;
}

.handle {
  position: absolute;
  width: calc(12 * var(--editor-px));
  height: calc(12 * var(--editor-px));
  background: var(--color-editor-handle);
  border: calc(2 * var(--editor-px)) solid var(--color-editor-lane-selected);
  border-radius: 50%;
  transform: translate(-50%, -50%);
  cursor: crosshair;
  pointer-events: all;
}
</style>
