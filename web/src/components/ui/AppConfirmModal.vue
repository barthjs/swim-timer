<script setup lang="ts">
import { computed, useTemplateRef, type ComponentPublicInstance } from 'vue';
import { useI18n } from 'vue-i18n';

import { useConfirm } from '@/composables/ui/useConfirm';

const { t } = useI18n();
const { state, settle } = useConfirm();
const confirmButton = useTemplateRef<ComponentPublicInstance>('confirmButton');
const cancelButton = useTemplateRef<ComponentPublicInstance>('cancelButton');

// Dismissing the modal with Escape, a click outside, or the close button counts as cancel.
const open = computed({
  get: () => state.value.open,
  set: (value: boolean) => {
    if (!value) settle(false);
  },
});

/**
 * focusDefault focuses the button Enter should press when the dialog opens: the confirm button,
 * or the cancel button for a destructive action shown in the error color, so Enter never deletes.
 * Escape cancels in both cases.
 */
function focusDefault(event: Event): void {
  const target = state.value.color === 'error' ? cancelButton.value : confirmButton.value;
  const button: unknown = target?.$el;
  if (!(button instanceof HTMLElement)) return;
  event.preventDefault();
  button.focus();
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="state.title"
    :content="{
      onOpenAutoFocus: focusDefault,
    }"
    :ui="{ overlay: 'z-[100]', content: 'z-[100]' }"
  >
    <template #body>
      <p v-if="state.description" class="text-muted text-sm">
        {{ state.description }}
      </p>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton ref="cancelButton" variant="ghost" color="neutral" @click="settle(false)">
          {{ state.cancelLabel || t('common.cancel') }}
        </UButton>
        <UButton
          ref="confirmButton"
          :color="state.color || 'primary'"
          :icon="state.icon"
          @click="settle(true)"
        >
          {{ state.confirmLabel || t('common.confirm') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
