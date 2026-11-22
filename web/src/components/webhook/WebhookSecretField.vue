<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import { useCopy } from '@/composables/ui/useCopy';
import { generateWebhookSecret } from '@/lib/webhook/webhook';

// The secret of a webhook in monospace with a button that copies it. An editable field also offers a button
// that fills in a new random secret. The legacy copy also works in the local network without https.
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

const model = defineModel<string>({
  required: true,
});

const { t } = useI18n();
const { copy } = useCopy();

/** Copies the secret to the clipboard. */
async function copySecret(): Promise<void> {
  await copy(model.value, t('webhook.secretCopied'));
}

/** Fills in a new random secret. */
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
    <UTooltip :text="t('webhook.copySecret')">
      <UButton
        icon="i-lucide-copy"
        color="neutral"
        variant="outline"
        :aria-label="t('webhook.copySecret')"
        :disabled="!model"
        @click="copySecret"
      />
    </UTooltip>
    <UTooltip v-if="!readonly" :text="t('webhook.generateSecret')">
      <UButton
        icon="i-lucide-refresh-cw"
        color="neutral"
        variant="outline"
        :aria-label="t('webhook.generateSecret')"
        @click="generateSecret"
      />
    </UTooltip>
  </div>
</template>
