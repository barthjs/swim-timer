<script setup lang="ts">
import { computed } from 'vue';

import { RULER_THICKNESS, rulerTicks } from '@/lib/editor/ruler';

// A ruler along the top or left edge of the canvas area, in the units of the content,
// such as pixels or millimeters. It follows the zoom and the pan of the canvas. A press on it places a guide
// across it at the pressed value, a vertical guide on the horizontal ruler and a horizontal one on the vertical ruler.
const props = defineProps<{
  orientation: 'horizontal' | 'vertical';
  // The length of the ruler on the screen.
  length: number;
  // Where the value 0 lies along the ruler, in pixels of the screen.
  offset: number;
  // The screen pixels per unit of the content.
  scale: number;
  // The position of the pointer along the ruler on the screen, marked while dragging, or null.
  pointer: number | null;
}>();

const emit = defineEmits<{
  guideDown: [event: PointerEvent];
}>();

const ticks = computed(() => rulerTicks(props.length, props.offset, props.scale));

const horizontal = computed(() => props.orientation === 'horizontal');
</script>

<template>
  <div
    class="bg-default/80 border-default text-muted absolute z-[5] touch-none overflow-hidden"
    :class="
      horizontal
        ? 'inset-x-0 top-0 h-5 cursor-col-resize border-b'
        : 'inset-y-0 left-0 w-5 cursor-row-resize border-r'
    "
    @pointerdown.left.stop="emit('guideDown', $event)"
  >
    <svg class="size-full">
      <template v-for="tick in ticks" :key="tick.position">
        <line
          v-if="horizontal"
          :x1="tick.position + 0.5"
          :x2="tick.position + 0.5"
          :y1="RULER_THICKNESS"
          :y2="RULER_THICKNESS - tick.size"
          stroke="currentColor"
        />
        <line
          v-else
          :y1="tick.position + 0.5"
          :y2="tick.position + 0.5"
          :x1="RULER_THICKNESS"
          :x2="RULER_THICKNESS - tick.size"
          stroke="currentColor"
        />
        <text
          v-if="tick.label && horizontal"
          :x="tick.position + 3"
          y="9"
          class="fill-current font-mono text-[9px]"
        >
          {{ tick.label }}
        </text>
        <text
          v-else-if="tick.label"
          x="2"
          :y="tick.position + 3"
          class="fill-current font-mono text-[9px]"
          :transform="`rotate(90 2 ${tick.position + 3})`"
        >
          {{ tick.label }}
        </text>
      </template>
    </svg>

    <div
      v-if="pointer !== null"
      class="bg-primary absolute"
      :class="horizontal ? 'inset-y-0 w-px' : 'inset-x-0 h-px'"
      :style="horizontal ? { left: `${pointer}px` } : { top: `${pointer}px` }"
    ></div>
  </div>
</template>
