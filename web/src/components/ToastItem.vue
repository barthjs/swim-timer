<script setup lang="ts">
import { computed } from 'vue';
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from 'lucide-vue-next';

import { type Toast, ToastType } from '@/types/toasts.ts';

const props = defineProps<{
  toast: Toast;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

const icon = computed(() => {
  switch (props.toast.type) {
    case ToastType.Success:
      return CheckCircle2;
    case ToastType.Error:
      return AlertCircle;
    case ToastType.Warning:
      return AlertTriangle;
    case ToastType.Info:
    default:
      return Info;
  }
});

const containerClasses = computed(() => {
  switch (props.toast.type) {
    case ToastType.Success:
      return 'bg-success text-fg border-success';
    case ToastType.Error:
      return 'bg-danger text-fg border-danger';
    case ToastType.Warning:
      return 'bg-warning text-fg border-warning';
    case ToastType.Info:
    default:
      return 'bg-info text-fg border-info';
  }
});
</script>

<template>
  <div
    class="flex items-center gap-3 rounded-lg border p-3 shadow-md transition-all duration-300"
    :class="containerClasses"
  >
    <Component :is="icon" class="h-5 w-5 shrink-0" />
    <span class="flex-1 leading-tight">{{ toast.message }}</span>
    <button
      class="hover:bg-fg/10 rounded-full p-1 transition-colors"
      @click="emit('close')"
      aria-label="Close"
    >
      <X class="h-4 w-4" />
    </button>
  </div>
</template>
