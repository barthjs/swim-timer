<script setup lang="ts">
import { useI18n } from 'vue-i18n';

// A question mark that shows a short explanation on hover, keyboard focus,
// or a tap on touch screens, where a second tap or a tap outside closes it.
// The short close delay lets the pointer move onto the text without closing it.
defineProps<{
  text: string;
}>();

const { t } = useI18n();
</script>

<template>
  <UPopover
    mode="hover"
    enable-touch
    :open-delay="100"
    :close-delay="150"
    :content="{ side: 'right', align: 'start' }"
  >
    <UButton
      icon="i-lucide-circle-help"
      color="neutral"
      variant="link"
      size="xs"
      class="p-0"
      :aria-label="t('common.help')"
    />

    <template #content>
      <p class="text-default max-w-72 p-3 text-sm whitespace-pre-line">{{ text }}</p>
    </template>
  </UPopover>
</template>
