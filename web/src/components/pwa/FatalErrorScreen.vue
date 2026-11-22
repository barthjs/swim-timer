<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import { fatalError } from '@/lib/pwa/fatalError';

const { t } = useI18n();

/** Loads the page anew. */
function reload(): void {
  window.location.reload();
}
</script>

<template>
  <UModal
    :open="fatalError !== null"
    fullscreen
    :dismissible="false"
    :close="false"
    :title="t('pwa.error.title')"
    :description="t('pwa.error.message')"
  >
    <template #content>
      <div class="flex h-full flex-col items-center justify-center gap-4 p-6 text-center">
        <UIcon name="i-lucide-triangle-alert" class="text-muted size-10" />
        <h2 class="heading-page">{{ t('pwa.error.title') }}</h2>
        <p class="text-muted max-w-md">{{ t('pwa.error.message') }}</p>
        <UButton :label="t('pwa.reload')" icon="i-lucide-refresh-cw" @click="reload()" />
      </div>
    </template>
  </UModal>
</template>
