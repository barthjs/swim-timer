<script setup lang="ts">
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from 'vue';

import { HeatStatus } from '@/api/types';
import { formatSwimtime } from '@/lib/time';
import { useLiveStore } from '@/stores/live';

// Shows the heat time. While the heat runs, the element's text is written directly on every animation frame
// instead of through a reactive binding, so the time-critical display never waits for Vue's render cycle.
// The element has no Vue-managed children, so the two never conflict. The timer stops at the stop time
// once every lane in use has finished, before the heat is official. It carries no styling of its own,
// since both the overlays and the admin pages use it.
const liveStore = useLiveStore();
const element = useTemplateRef<HTMLElement>('element');

// The heat start on the performance.now() clock, or null while the heat is not running.
let heatStartTime: number | null = null;
let frame: number | null = null;

/** write replaces the displayed text. */
function write(text: string): void {
  if (element.value) {
    element.value.textContent = text;
  }
}

/** tick shows the elapsed heat time and schedules the next frame while the heat runs. */
function tick(): void {
  if (heatStartTime === null) {
    frame = null;
    return;
  }
  write(formatSwimtime(Math.max(0, performance.now() - heatStartTime)));
  frame = requestAnimationFrame(tick);
}

/**
 * sync re-derives the start and the stop on every state change. The server sends the start on its own clock,
 * which the live store converts to the monotonic performance clock of this client.
 */
function sync(): void {
  const state = liveStore.state;
  const startTime = liveStore.startTimeLocal();
  const stopTime = state?.stopTime ?? 0;

  if (state?.status === HeatStatus.Start && startTime !== null && stopTime === 0) {
    heatStartTime = startTime;
    if (frame === null) {
      frame = requestAnimationFrame(tick);
    }
    return;
  }

  heatStartTime = null;
  if (frame !== null) {
    cancelAnimationFrame(frame);
    frame = null;
  }
  write(stopTime > 0 ? formatSwimtime(stopTime) : liveStore.displayTime);
}

onMounted(() => {
  watch(
    () => [
      liveStore.state?.status,
      liveStore.state?.startTime,
      liveStore.state?.stopTime,
      liveStore.displayTime,
    ],
    sync,
    {
      immediate: true,
    },
  );
});

onBeforeUnmount(() => {
  heatStartTime = null;
  if (frame !== null) {
    cancelAnimationFrame(frame);
  }
});
</script>

<template>
  <span ref="element"></span>
</template>
