<script setup lang="ts">
import { computed } from 'vue';

import ToastItem from '@/components/ToastItem.vue';

import { useToastStore } from '@/stores/toastStore.ts';

import { type Toast, ToastPosition } from '@/types/toasts.ts';

const toastStore = useToastStore();

const positions = Object.values(ToastPosition);

const toastsByPosition = computed(() => {
  const result = {} as Record<ToastPosition, Toast[]>;
  positions.forEach((pos) => {
    result[pos] = toastStore.toasts.filter((t) => (t.position || ToastPosition.TopCenter) === pos);
  });
  return result;
});

const getPositionClasses = (position: ToastPosition) => {
  switch (position) {
    case ToastPosition.TopLeft:
      return 'top-6 left-6';
    case ToastPosition.TopCenter:
      return 'top-6 left-1/2 -translate-x-1/2';
    case ToastPosition.TopRight:
      return 'top-6 right-6';
    case ToastPosition.BottomLeft:
      return 'bottom-6 left-6';
    case ToastPosition.BottomCenter:
      return 'bottom-6 left-1/2 -translate-x-1/2';
    case ToastPosition.BottomRight:
      return 'bottom-6 right-6';
    default:
      return '';
  }
};
</script>

<template>
  <div
    v-for="position in positions"
    :key="position"
    class="pointer-events-none fixed z-9999 flex max-w-xl flex-col gap-2"
    :class="getPositionClasses(position)"
  >
    <TransitionGroup name="toast-list" tag="div" class="pointer-events-auto flex flex-col gap-2">
      <ToastItem
        v-for="toast in toastsByPosition[position]"
        :key="toast.id"
        :toast="toast"
        @close="toastStore.remove(toast.id)"
      />
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-list-enter-active,
.toast-list-leave-active {
  transition: all 0.3s ease;
}
.toast-list-enter-from {
  opacity: 0;
  transform: translateY(-20px) scale(0.9);
}
.toast-list-leave-to {
  opacity: 0;
  transform: scale(0.9);
}
.toast-list-move {
  transition: transform 0.3s ease;
}
</style>
