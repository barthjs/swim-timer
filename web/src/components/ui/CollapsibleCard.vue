<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { useLocalStorage } from '@vueuse/core';

// A card whose body can be collapsed by its header. Whether it is open is remembered in the browser
// under storageKey, so a collapsed section stays collapsed after a reload. The #summary slot shows
// next to the title and stays visible while the card is collapsed.
const props = defineProps<{
  title: string;
  icon: string;
  storageKey: string;
  // Whether the card starts open before the user has collapsed or opened it.
  defaultOpen?: boolean;
}>();

const { t } = useI18n();
const open = useLocalStorage(props.storageKey, props.defaultOpen ?? true);
</script>

<template>
  <UCard variant="subtle" :ui="{ body: open ? undefined : 'hidden' }">
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon :name="icon" class="text-primary size-5 shrink-0" />
        <h2 class="text-highlighted font-semibold">{{ title }}</h2>
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <slot name="summary" />
        </div>
        <UButton
          :icon="open ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          color="neutral"
          variant="ghost"
          size="sm"
          :aria-label="open ? t('common.collapse') : t('common.expand')"
          :aria-expanded="open"
          @click="open = !open"
        />
      </div>
    </template>

    <slot v-if="open" />
  </UCard>
</template>
