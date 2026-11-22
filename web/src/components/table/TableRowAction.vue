<script setup lang="ts">
import type { ButtonProps } from '@nuxt/ui';

// A frequent action of a table row as a button with a tooltip, placed in the slot of TableRowActions
withDefaults(
  defineProps<{
    label: string;
    icon: string;
    color?: ButtonProps['color'];
  }>(),
  {
    color: 'neutral',
  },
);

const emit = defineEmits<{
  click: [event: MouseEvent];
}>();
</script>

<template>
  <UTooltip :text="label">
    <UButton
      :icon="icon"
      :color="color"
      variant="ghost"
      size="sm"
      :aria-label="label"
      @click="(event: MouseEvent) => emit('click', event)"
    />
  </UTooltip>
</template>
