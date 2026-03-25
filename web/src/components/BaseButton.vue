<script setup lang="ts">
import { type Component, computed } from 'vue';

interface Props {
  variant?: 'primary' | 'success' | 'danger' | 'warning' | 'info' | 'ghost';
  size?: 'sm' | 'md' | 'lg' | 'icon';
  icon?: Component;
  iconSize?: number;
  active?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'toggle';
  title?: string;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  iconSize: 18,
  active: false,
  disabled: false,
  type: 'button',
});

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void;
}>();

const baseClasses =
  'flex items-center justify-center gap-2 rounded-md font-medium transition-colors duration-200 cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 select-none text-fg';

const variantClasses = computed(() => {
  if (props.type === 'toggle') {
    return props.active ? 'bg-primary hover:bg-primary-hover' : 'bg-surface hover:bg-subtle';
  }

  switch (props.variant) {
    case 'primary':
      return 'bg-primary hover:bg-primary-hover';
    case 'success':
      return 'bg-success hover:bg-success-hover';
    case 'danger':
      return 'bg-danger hover:bg-danger-hover';
    case 'warning':
      return 'bg-warning hover:bg-warning-hover';
    case 'info':
      return 'bg-info hover:bg-info-hover';
    case 'ghost':
      return 'bg-transparent hover:bg-surface';
    default:
      return 'bg-primary hover:bg-primary-hover';
  }
});

const sizeClasses = computed(() => {
  if (props.size === 'icon') return 'p-2';

  switch (props.size) {
    case 'sm':
      return 'px-3 py-1.5 text-sm';
    case 'md':
      return 'px-4 py-2 text-base';
    case 'lg':
      return 'px-6 py-3 text-lg';
    default:
      return 'px-4 py-2 text-base';
  }
});

const handleClick = (event: MouseEvent) => {
  if (!props.disabled) {
    emit('click', event);
  }
};
</script>

<template>
  <button
    :type="type === 'toggle' ? 'button' : type"
    :class="[baseClasses, variantClasses, sizeClasses]"
    :disabled="disabled"
    :title="title"
    @click="handleClick"
  >
    <component :is="icon" v-if="icon" :size="iconSize" />
    <slot />
  </button>
</template>
