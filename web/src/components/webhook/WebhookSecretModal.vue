<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { WebhookResponse } from '@/api/types';
import WebhookSecretField from '@/components/webhook/WebhookSecretField.vue';

// Shows the secret of a webhook right after it was created, so it can be entered at the receiver.
defineProps<{
  webhook: WebhookResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();
</script>

<template>
  <UModal
    v-model:open="open"
    :title="t('pages.config.webhook.createdTitle', { name: webhook?.name ?? '' })"
    :description="t('pages.config.webhook.createdDescription')"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <UFormField v-if="webhook" :label="t('fields.secret')">
        <WebhookSecretField :model-value="webhook.secret" readonly />
      </UFormField>
    </template>
    <template #footer>
      <div class="flex w-full justify-end">
        <UButton :label="t('common.ok')" @click="open = false" />
      </div>
    </template>
  </UModal>
</template>
