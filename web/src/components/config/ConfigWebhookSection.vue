<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { WebhookEvent } from '@/api/types/webhook';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import WebhookTable from '@/components/webhook/WebhookTable.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/config/useConfigForm';
import type { WebhookEventOption } from '@/lib/webhook/webhook';

// The networks of the local network webhooks may reach, and the webhooks the live data is delivered to.
// The networks belong to the configuration and are saved with it, the webhooks are saved on their own.
const { t } = useI18n();
const { state, fieldProps } = useConfigForm();

const webhookEvents = computed<WebhookEventOption[]>(() => [
  {
    value: WebhookEvent.HeatUpdated,
    label: t('pages.config.webhook.events.heatUpdated'),
    description: t('pages.config.webhook.events.heatUpdatedHint'),
  },
  {
    value: WebhookEvent.MeetReloaded,
    label: t('pages.config.webhook.events.meetReloaded'),
    description: t('pages.config.webhook.events.meetReloadedHint'),
  },
]);

/** Stops Enter in a field of the webhook table, which would submit the form of the configuration around it. */
function keepEnterInTable(event: KeyboardEvent): void {
  if (event.target instanceof HTMLInputElement) event.preventDefault();
}
</script>

<template>
  <div class="flex flex-col gap-4">
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

    <div @keydown.enter="keepEnterInTable">
      <WebhookTable :events="webhookEvents" />
    </div>
  </div>
</template>
