<script setup lang="ts">
import HelpHint from '@/components/form/HelpHint.vue';

// A filter in the filter grid of TableToolbar. It fills one cell of the grid, with the name of the filter
// above the control, so the control keeps its full width for the value. The name turns into the primary color
// while the filter is set, which shows what the list is filtered by at a glance. The slot holds exactly one control.
withDefaults(
  defineProps<{
    label: string;
    active?: boolean;
    // A short explanation behind a question mark next to the name
    hint?: string;
  }>(),
  {
    active: false,
    hint: undefined,
  },
);
</script>

<template>
  <div class="flex w-full min-w-0 flex-col gap-1">
    <div class="flex min-w-0 items-center gap-1">
      <span class="truncate text-xs font-medium" :class="active ? 'text-primary' : 'text-muted'">
        {{ label }}
      </span>
      <HelpHint v-if="hint" :text="hint" />
    </div>
    <slot />
  </div>
</template>
