<script setup lang="ts">
// Crosshair lines through the pointer with its coordinates, shown while something on the canvas is dragged.
// The lines are placed in pixels of the canvas area, the coordinates are whole units of the content,
// as the fields of the editor show them.
defineProps<{
  // The position of the pointer in the canvas area.
  x: number;
  y: number;
  // The position of the pointer in units of the content.
  contentX: number;
  contentY: number;
}>();
</script>

<template>
  <div
    class="bg-primary pointer-events-none absolute inset-x-0 h-px mix-blend-difference"
    :style="{ top: `${y}px` }"
  ></div>
  <div
    class="bg-primary pointer-events-none absolute inset-y-0 w-px mix-blend-difference"
    :style="{ left: `${x}px` }"
  ></div>
  <div
    class="bg-default text-highlighted pointer-events-none absolute rounded-md p-1.5 font-mono text-xs shadow-lg"
    :style="{ left: `${x + 12}px`, top: `${y + 12}px` }"
  >
    <div class="leading-none whitespace-nowrap">X: {{ Math.round(contentX) }}</div>
    <div class="leading-none whitespace-nowrap">Y: {{ Math.round(contentY) }}</div>
  </div>
</template>
