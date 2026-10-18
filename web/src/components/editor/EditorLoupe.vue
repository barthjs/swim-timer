<script setup lang="ts">
import { computed } from 'vue';

import type { Point, Size } from '@/lib/editor/geometry';

// The diameter of the loupe on the screen.
const SIZE = 180;
// The magnification relative to the content as it is shown on the screen.
const ZOOM = 4;
// The distance between the pointer and the loupe, which flips to the other side near the area edge.
const OFFSET = 24;

// A magnified view of the content around the pointer while a corner or an edge is dragged,
// for placing it exactly. The default slot draws the content in content units, with the stroke width
// that keeps outlines at one pixel of the loupe.
const props = defineProps<{
  // The position of the pointer in the area and in content units.
  pointer: Point;
  contentPointer: Point;
  // The size of the area on the screen and of the content in its units.
  area: Size;
  content: Size;
  // The screen pixels per content unit.
  scale: number;
}>();

defineSlots<{
  default: (props: { strokeWidth: number }) => unknown;
}>();

const magnification = computed(() => props.scale * ZOOM);

const position = computed(() => {
  const { x, y } = props.pointer;
  const left = x + OFFSET + SIZE > props.area.width ? x - OFFSET - SIZE : x + OFFSET;
  const top = y + OFFSET + SIZE > props.area.height ? y - OFFSET - SIZE : y + OFFSET;
  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${SIZE}px`,
    height: `${SIZE}px`,
  };
});

const contentStyle = computed(() => {
  const dx = SIZE / 2 - magnification.value * props.contentPointer.x;
  const dy = SIZE / 2 - magnification.value * props.contentPointer.y;
  return {
    width: `${props.content.width}px`,
    height: `${props.content.height}px`,
    transform: `translate(${dx}px, ${dy}px) scale(${magnification.value})`,
  };
});

const strokeWidth = computed(() => 1 / magnification.value);
</script>

<template>
  <div
    class="border-default bg-default pointer-events-none absolute z-30 overflow-hidden rounded-full border-2 shadow-xl"
    :style="position"
  >
    <div class="absolute top-0 left-0 origin-top-left" :style="contentStyle">
      <slot :stroke-width="strokeWidth" />
    </div>
    <div class="bg-primary absolute inset-x-0 top-1/2 h-px"></div>
    <div class="bg-primary absolute inset-y-0 left-1/2 w-px"></div>
  </div>
</template>
