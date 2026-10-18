<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useLocalStorage } from '@vueuse/core';

import { usePointerDrag } from '@/composables/usePointerDrag';

// The width range of the sidebar in pixels of the screen, and the width it starts with.
const MIN_WIDTH = 280;
const MAX_WIDTH = 640;
const DEFAULT_WIDTH = 352;

// The panels beside the canvas of an editor. From lg up, the handle on its left edge changes its width, which the
// browser keeps under storageKey, and a double click on the handle restores the default width. Below lg,
// the sidebar stacks under the canvas at full width.
const props = defineProps<{
  storageKey: string;
}>();

const { t } = useI18n();
const width = useLocalStorage(`${props.storageKey}.sidebarWidth`, DEFAULT_WIDTH);

let startWidth = DEFAULT_WIDTH;
const drag = usePointerDrag({
  onStart: () => {
    startWidth = width.value;
  },
  // The handle sits on the left edge, so moving the pointer to the left widens the sidebar
  onMove: (deltaX) => {
    width.value = clampWidth(startWidth - deltaX);
  },
});

const style = computed(() => ({
  '--sidebar-width': `${clampWidth(width.value)}px`,
}));

/** onHandleDown starts changing the width with the primary pointer button. */
function onHandleDown(event: PointerEvent): void {
  if (event.button !== 0) return;
  event.preventDefault();
  drag.start(event);
}

/** resetWidth restores the default width. */
function resetWidth(): void {
  width.value = DEFAULT_WIDTH;
}

/** clampWidth keeps a width within the range of the sidebar. */
function clampWidth(value: number): number {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, value));
}
</script>

<template>
  <div
    class="relative flex flex-col gap-3 lg:min-h-0 lg:w-(--sidebar-width) lg:shrink-0"
    :style="style"
  >
    <div
      role="separator"
      aria-orientation="vertical"
      :aria-label="t('editor.resizeSidebar')"
      :aria-valuenow="width"
      :aria-valuemin="MIN_WIDTH"
      :aria-valuemax="MAX_WIDTH"
      class="hover:bg-primary/20 active:bg-primary/30 absolute inset-y-0 -left-3 z-10 hidden w-3 cursor-col-resize touch-none rounded-full transition-colors lg:block"
      @pointerdown="onHandleDown"
      @dblclick="resetWidth"
    ></div>
    <!-- The panels are a container, so their rows wrap by the width of the sidebar and not of the window. -->
    <div class="@container flex flex-col gap-3 lg:min-h-0 lg:overflow-y-auto">
      <slot />
    </div>
  </div>
</template>
