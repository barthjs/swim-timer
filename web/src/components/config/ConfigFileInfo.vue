<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useConfigForm } from '@/composables/config/useConfigForm';

// The config file the configuration was loaded from and a save writes to, with its state in the tooltip.
// Without a file found at startup, a save creates one in the first writable configuration directory.
const { t } = useI18n();
const { file } = useConfigForm();

const status = computed(() => {
  if (!file.value.path) return t('pages.config.file.none');
  return file.value.exists ? t('pages.config.file.loaded') : t('pages.config.file.missing');
});
</script>

<template>
  <UTooltip :text="status">
    <span class="text-muted flex min-w-0 items-center gap-1.5 text-sm">
      <UIcon name="i-lucide-file-cog" class="size-4 shrink-0" />
      <span class="truncate font-mono">{{ file.path || status }}</span>
    </span>
  </UTooltip>
</template>
