<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

// The actions of a table row: the buttons of the frequent actions in the slot, followed by a menu with the rest.
// Clicks stay in the cell, so they do not also open the row.
defineProps<{
  items: DropdownMenuItem[] | DropdownMenuItem[][];
}>();

const { t } = useI18n();
</script>

<template>
  <div class="flex items-center justify-end gap-1" @click.stop>
    <slot />
    <UDropdownMenu :items="items" :content="{ align: 'end' }">
      <UButton
        icon="i-lucide-ellipsis-vertical"
        color="neutral"
        variant="ghost"
        size="sm"
        :aria-label="t('table.rowActions')"
      />
    </UDropdownMenu>
  </div>
</template>
