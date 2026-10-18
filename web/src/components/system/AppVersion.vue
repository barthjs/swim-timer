<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import UpdatePanel from '@/components/system/UpdatePanel.vue';
import { useAuthStore } from '@/stores/auth';

const version = import.meta.env.VITE_APP_VERSION || 'dev';

const { t } = useI18n();
const auth = useAuthStore();
const updateOpen = ref(false);
// A running update keeps the dialog open, as closing it would stop the reload into the new version
const updateBusy = ref(false);
</script>

<template>
  <div class="flex items-center justify-center gap-1">
    <a
      href="https://github.com/barthjs/swim-timer/releases"
      target="_blank"
      rel="noopener noreferrer"
      class="flex items-center gap-1.5"
    >
      <UIcon name="i-lucide-tag" :size="12" />
      <span class="font-mono text-xs">{{ version }}</span>
    </a>

    <UModal
      v-if="auth.authenticated"
      v-model:open="updateOpen"
      :title="t('update.title')"
      :dismissible="!updateBusy"
      :close="!updateBusy"
    >
      <UTooltip :text="t('update.check')">
        <UButton
          icon="i-lucide-refresh-cw"
          size="xs"
          color="neutral"
          variant="ghost"
          :aria-label="t('update.check')"
        />
      </UTooltip>

      <template #body>
        <UpdatePanel @busy="updateBusy = $event" />
      </template>
    </UModal>
  </div>
</template>
