<script setup lang="ts">
import { ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { recordTableRepository } from '@/api/repositories/recordTable';
import type { RecordTableResponse } from '@/api/types/recordTable';
import { useMutation } from '@/composables/form/useMutation';
import { useSortableList } from '@/composables/ui/useSortableList';
import { moveItem } from '@/lib/table/columns';

// Orders the record tables by dragging them in a dialog and saves the order with a button.
// The first enabled table in which a lane breaks or equals a record decides which record it counts for.
const props = defineProps<{
  tables: RecordTableResponse[];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  saved: [];
}>();

const { t } = useI18n();

const order = ref<RecordTableResponse[]>([]);
const list = useTemplateRef<HTMLElement>('list');
const { start, stop } = useSortableList(list, (oldIndex, newIndex) => {
  order.value = moveItem(order.value, oldIndex, newIndex);
});

// Each opening starts from the saved order
watch(open, (isOpen) => {
  if (!isOpen) {
    stop();
    return;
  }
  order.value = [...props.tables];
  start();
});

const { mutate: save, pending: saving } = useMutation(
  async () => recordTableRepository.reorder(order.value.map((table) => table.id)),
  {
    success: () => t('pages.records.tables.orderSaved'),
    onSuccess: () => {
      open.value = false;
      emit('saved');
    },
  },
);
</script>

<template>
  <UModal
    v-model:open="open"
    :title="t('pages.records.tables.order')"
    :description="t('pages.records.tables.orderHint')"
  >
    <template #body>
      <div
        ref="list"
        class="border-default divide-default flex flex-col divide-y rounded-md border"
      >
        <div
          v-for="(table, index) in order"
          :key="table.id"
          class="bg-default flex items-center gap-3 px-3 py-2"
        >
          <UIcon
            data-drag-handle
            name="i-lucide-grip-vertical"
            class="text-muted size-4 shrink-0 cursor-grab"
          />
          <span class="text-muted w-5 shrink-0 text-sm tabular-nums">{{ index + 1 }}.</span>
          <UBadge :label="table.abbreviation" color="primary" variant="subtle" />
          <span class="text-highlighted min-w-0 flex-1 truncate text-sm">{{ table.name }}</span>
          <UBadge
            v-if="!table.enabled"
            :label="t('pages.records.tables.disabled')"
            color="neutral"
            variant="subtle"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full items-center justify-end gap-2">
        <UButton variant="ghost" color="neutral" :disabled="saving" @click="open = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton :loading="saving" @click="save()">{{ t('common.save') }}</UButton>
      </div>
    </template>
  </UModal>
</template>
