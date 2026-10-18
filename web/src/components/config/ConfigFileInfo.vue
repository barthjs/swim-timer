<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useConfigForm } from '@/composables/useConfigForm';

// The config file the configuration was loaded from and a save writes to. Without a file found
// at startup, a save creates one in the first writable configuration directory.
const { t } = useI18n();
const { file } = useConfigForm();

const status = computed(() => {
  if (!file.value.path) return t('pages.config.file.none');
  return file.value.exists ? t('pages.config.file.loaded') : t('pages.config.file.missing');
});
</script>

<template>
  <div class="text-muted flex min-w-0 items-start gap-2 text-sm">
    <UIcon name="i-lucide-file-cog" class="mt-0.5 size-4 shrink-0" />
    <p class="min-w-0">
      <span>{{ status }}</span>
      <span v-if="file.path" class="text-default ml-1 font-mono break-all">{{ file.path }}</span>
    </p>
  </div>
</template>
