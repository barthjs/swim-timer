<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import { focusFieldOnOpen } from '@/lib/form/focus';

// A dialog with a form in its body and the cancel and submit buttons in its footer.
// The submit button submits the form with the id formId, so Enter submits it as well.
// Cancel leaves closing to the parent, which asks first while there are unsaved changes. Opening it focuses the first field.
defineProps<{
  title: string;
  formId: string;
  submitLabel: string;
  description?: string;
  loading?: boolean;
  disabled?: boolean;
  ui?: Record<string, string>;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  cancel: [];
}>();

const { t } = useI18n();
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    :description="description"
    :content="{ onOpenAutoFocus: focusFieldOnOpen }"
    :ui="ui"
  >
    <template #body>
      <slot />
    </template>
    <template #footer>
      <div class="flex w-full items-center justify-end gap-2">
        <slot name="footer-start" />
        <UButton variant="ghost" color="neutral" :disabled="loading" @click="emit('cancel')">
          {{ t('common.cancel') }}
        </UButton>
        <UButton type="submit" :form="formId" :loading="loading" :disabled="disabled">
          {{ submitLabel }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
