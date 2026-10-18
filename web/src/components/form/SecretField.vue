<script setup lang="ts">
import { computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import PasswordField from '@/components/form/PasswordField.vue';

// A password field for a secret the server never sends. A saved secret shows as a placeholder.
// Leaving the field empty keeps it and typing replaces it. The remove button marks it for removal,
// which the parent sends as clearSecrets.
const model = defineModel<string>({
  required: true,
});
const cleared = defineModel<boolean>('cleared', {
  default: false,
});

const props = withDefaults(
  defineProps<{
    // Whether the server has a value for this secret.
    saved: boolean;
    color?: 'warning';
    highlight?: boolean;
    maxlength?: number;
  }>(),
  {
    color: undefined,
    highlight: false,
    maxlength: undefined,
  },
);

const { t } = useI18n();

const placeholder = computed(() => {
  if (!props.saved) return undefined;
  return cleared.value ? t('common.secretRemoved') : t('common.secretSaved');
});

// Typing a new value replaces the secret, so it is no longer marked for removal
watch(model, (value) => {
  if (value) cleared.value = false;
});
</script>

<template>
  <div class="flex items-center gap-2">
    <PasswordField
      v-model="model"
      :placeholder="placeholder"
      :color="color"
      :highlight="highlight || cleared"
      :maxlength="maxlength"
    />
    <UTooltip v-if="saved && !model && !cleared" :text="t('common.removeSecret')">
      <UButton
        icon="i-lucide-trash-2"
        color="neutral"
        variant="ghost"
        :aria-label="t('common.removeSecret')"
        @click="cleared = true"
      />
    </UTooltip>
    <UTooltip v-else-if="cleared" :text="t('common.keepSecret')">
      <UButton
        icon="i-lucide-undo-2"
        color="neutral"
        variant="ghost"
        :aria-label="t('common.keepSecret')"
        @click="cleared = false"
      />
    </UTooltip>
  </div>
</template>
