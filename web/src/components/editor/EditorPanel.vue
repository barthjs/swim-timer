<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { useLocalStorage } from '@vueuse/core';

// A card of the sidebar beside the canvas whose body collapses by its header, like the panels of image editors.
// Whether it is open is a matter of taste and kept in the browser under storageKey. The summary slot shows
// next to the title and stays visible while the panel is collapsed.
const props = defineProps<{
  title: string;
  storageKey: string;
}>();

defineSlots<{
  default: () => unknown;
  summary: () => unknown;
}>();

const { t } = useI18n();
const open = useLocalStorage(props.storageKey, true);

/** toggle opens or collapses the panel. */
function toggle(): void {
  open.value = !open.value;
}
</script>

<template>
  <UCard :ui="{ root: 'bg-elevated', header: 'p-0 sm:p-0', body: open ? 'p-4 sm:p-4' : 'hidden' }">
    <template #header>
      <div class="flex items-center gap-2 py-1 pr-1 pl-4">
        <h2 class="text-highlighted font-semibold">{{ title }}</h2>
        <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
          <slot name="summary" />
        </div>
        <UButton
          :icon="open ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          color="neutral"
          variant="ghost"
          size="sm"
          :aria-label="open ? t('editor.collapse') : t('editor.expand')"
          :aria-expanded="open"
          @click="toggle"
        />
      </div>
    </template>

    <div v-if="open" class="flex flex-col gap-3">
      <slot />
    </div>
  </UCard>
</template>
