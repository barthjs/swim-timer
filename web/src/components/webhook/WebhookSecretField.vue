<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';
import { useClipboard } from '@vueuse/core';

import { generateWebhookSecret } from '@/lib/webhook';

// The secret of a webhook in monospace with a button that copies it. An editable field also offers a button
// that fills in a new random secret. The legacy copy also works in the local network without https.
const model = defineModel<string>({
  required: true,
});

withDefaults(
  defineProps<{
    readonly?: boolean;
    maxlength?: number;
    placeholder?: string;
  }>(),
  {
    readonly: false,
    maxlength: undefined,
    placeholder: undefined,
  },
);

const { t } = useI18n();
const toast = useToast();
const { copy, isSupported } = useClipboard({
  legacy: true,
});

/** copySecret copies the secret to the clipboard. */
async function copySecret(): Promise<void> {
  await copy(model.value);
  toast.add({
    title: t('pages.config.webhook.secretCopied'),
    icon: 'i-lucide-clipboard-check',
    color: 'success',
  });
}

/** generateSecret fills in a new random secret. */
function generateSecret(): void {
  model.value = generateWebhookSecret();
}
</script>

<template>
  <div class="flex items-center gap-2">
    <UInput
      v-model="model"
      :readonly="readonly"
      :maxlength="maxlength"
      :placeholder="placeholder"
      autocomplete="off"
      class="w-full font-mono"
    />
    <UTooltip v-if="isSupported" :text="t('pages.config.webhook.copySecret')">
      <UButton
        icon="i-lucide-copy"
        color="neutral"
        variant="outline"
        :aria-label="t('pages.config.webhook.copySecret')"
        :disabled="!model"
        @click="copySecret"
      />
    </UTooltip>
    <UTooltip v-if="!readonly" :text="t('pages.config.webhook.generateSecret')">
      <UButton
        icon="i-lucide-refresh-cw"
        color="neutral"
        variant="outline"
        :aria-label="t('pages.config.webhook.generateSecret')"
        @click="generateSecret"
      />
    </UTooltip>
  </div>
</template>
