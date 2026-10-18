<script setup lang="ts">
import { computed } from 'vue';

import type { ViewTransform } from '@/composables/editor/useCanvasViewport';
import type { Bounds } from '@/lib/editor/geometry';

// The rectangle drawn while selecting items on the empty canvas. It lies in the area above the content,
// so its border keeps one screen pixel at every zoom.
const props = defineProps<{
  // The selected area in content units.
  bounds: Bounds;
  view: ViewTransform;
}>();

const style = computed(() => ({
  left: `${props.view.x + props.bounds.minX * props.view.scale}px`,
  top: `${props.view.y + props.bounds.minY * props.view.scale}px`,
  width: `${(props.bounds.maxX - props.bounds.minX) * props.view.scale}px`,
  height: `${(props.bounds.maxY - props.bounds.minY) * props.view.scale}px`,
}));
</script>

<template>
  <div
    class="border-primary bg-primary/10 pointer-events-none absolute z-30 border border-dashed"
    :style="style"
  ></div>
</template>
