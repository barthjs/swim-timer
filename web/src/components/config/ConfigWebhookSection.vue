<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import WebhookTable from '@/components/webhook/WebhookTable.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/useConfigForm';

// The networks of the local network webhooks may reach, and the webhooks the live data is delivered to.
// The networks belong to the configuration and are saved with it, the webhooks are saved on their own.
const { t } = useI18n();
const { state, fieldProps } = useConfigForm();

/**
 * keepEnterInTable stops Enter in a field of the webhook table, such as the search, since the browser would
 * otherwise submit the form of the configuration around it.
 */
function keepEnterInTable(event: KeyboardEvent): void {
  if (event.target instanceof HTMLInputElement) event.preventDefault();
}
</script>

<template>
  <div class="mt-4 flex flex-col gap-4 lg:min-h-0 lg:flex-1">
    <UCard variant="subtle" class="shrink-0">
      <UFormField :label="t('fields.webhook.allowedNetworks')" name="webhook.allowedNetworks">
        <template #hint>
          <ConfigFieldHint
            path="webhook.allowedNetworks"
            :help="t('pages.config.webhook.allowedNetworksHint')"
          />
        </template>
        <UTextarea
          v-model="state.webhook.allowedNetworks"
          v-bind="fieldProps('webhook.allowedNetworks')"
          :maxlength="CONFIG_LIMITS.webhookAllowedNetworks"
          :rows="2"
          class="w-full"
        />
      </UFormField>
    </UCard>

    <div class="flex flex-col lg:min-h-0 lg:flex-1" @keydown.enter="keepEnterInTable">
      <WebhookTable />
    </div>
  </div>
</template>
