<script setup lang="ts">
import { computed } from 'vue';

import type { ViewTransform } from '@/composables/editor/useCanvasViewport';
import type { Size } from '@/lib/editor/geometry';
import type { SnapFeedback } from '@/lib/editor/snap';

// The diameter of the ring around the point a dragged item snapped onto.
const RING_SIZE = 16;

// Shows what a dragged item snapped to: the lines it aligned with across the whole content, a ring around the point
// it snapped onto, and its distances to the neighbors with their length. It lies in the area above the content,
// so lines, ring, and labels keep their size at every zoom.
const props = defineProps<{
  feedback: SnapFeedback;
  content: Size;
  view: ViewTransform;
  // The unit the distances are labeled in, such as "px" or "mm".
  unit: string;
}>();

const lines = computed(() =>
  props.feedback.lines.map((line) => {
    const { x, y, scale } = props.view;
    if (line.axis === 'x') {
      return {
        left: `${x + line.value * scale}px`,
        top: `${y}px`,
        width: '1px',
        height: `${props.content.height * scale}px`,
      };
    }
    return {
      left: `${x}px`,
      top: `${y + line.value * scale}px`,
      width: `${props.content.width * scale}px`,
      height: '1px',
    };
  }),
);

const gaps = computed(() =>
  props.feedback.gaps.map((gap) => {
    const { x, y, scale } = props.view;
    const label = `${Math.round(gap.to - gap.from)} ${props.unit}`;
    if (gap.axis === 'x') {
      return {
        label,
        line: {
          left: `${x + gap.from * scale}px`,
          top: `${y + gap.at * scale}px`,
          width: `${(gap.to - gap.from) * scale}px`,
          height: '1px',
        },
      };
    }
    return {
      label,
      line: {
        left: `${x + gap.at * scale}px`,
        top: `${y + gap.from * scale}px`,
        width: '1px',
        height: `${(gap.to - gap.from) * scale}px`,
      },
    };
  }),
);

const ring = computed(() => {
  const point = props.feedback.point;
  if (!point) return null;
  return {
    left: `${props.view.x + point.x * props.view.scale - RING_SIZE / 2}px`,
    top: `${props.view.y + point.y * props.view.scale - RING_SIZE / 2}px`,
    width: `${RING_SIZE}px`,
    height: `${RING_SIZE}px`,
  };
});
</script>

<template>
  <div
    v-for="(line, index) in lines"
    :key="index"
    class="bg-primary pointer-events-none absolute z-20"
    :style="line"
  ></div>
  <div
    v-for="(gap, index) in gaps"
    :key="`gap-${index}`"
    class="bg-warning pointer-events-none absolute z-20"
    :style="gap.line"
  >
    <span
      class="bg-warning text-inverted absolute top-1/2 left-1/2 -translate-1/2 rounded px-1 text-[10px] whitespace-nowrap tabular-nums"
    >
      {{ gap.label }}
    </span>
  </div>
  <div
    v-if="ring"
    class="border-primary pointer-events-none absolute z-20 rounded-full border-2"
    :style="ring"
  ></div>
</template>
