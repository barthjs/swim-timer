<script setup lang="ts">
import HelpHint from '@/components/form/HelpHint.vue';

// The hint slot of a form field: an optional character count and an optional explanation
// behind a question mark, so explanations do not take up space below every field.
defineProps<{
  help?: string;
  count?: string;
}>();
</script>

<template>
  <span class="flex items-center gap-1.5">
    <span v-if="count">{{ count }}</span>
    <HelpHint v-if="help" :text="help" />
  </span>
</template>
