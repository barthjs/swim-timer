<script setup lang="ts">
import { useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

import { recordTableRepository } from '@/api/repositories/recordTable';
import type { RecordTableResponse, RecordTableUpdateRequest } from '@/api/types/recordTable';
import FormModal from '@/components/form/FormModal.vue';
import RecordTableFields from '@/components/record/RecordTableFields.vue';
import { useEditDialog } from '@/composables/form/useEditDialog';
import { useFormValidation, type FormErrorTarget } from '@/composables/form/useFormValidation';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { RECORD_TABLE_RULES } from '@/lib/record/recordTable';

// Edits the name, the abbreviation, and the state of a record table in a dialog.
// Its records are replaced by an import instead.
const props = defineProps<{
  table: RecordTableResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  saved: [table: RecordTableResponse];
}>();

const { t } = useI18n();
const form = useTemplateRef<FormErrorTarget>('form');

const {
  open: openProxy,
  state,
  attemptClose,
  save,
  saving,
} = useEditDialog<RecordTableResponse, RecordTableUpdateRequest, RecordTableResponse>({
  open,
  row: () => props.table,
  initial: (table) => ({
    name: table?.name ?? '',
    abbreviation: table?.abbreviation ?? '',
    enabled: table?.enabled ?? true,
  }),
  // The dialog only edits, a new table starts with an import
  update: async (table, current) => recordTableRepository.update(table.id, current),
  success: () => t('pages.records.tables.saved'),
  onSaved: (table) => emit('saved', table),
  form,
});

const { validate } = useFormValidation<RecordTableUpdateRequest>(() => RECORD_TABLE_RULES);
</script>

<template>
  <FormModal
    v-model:open="openProxy"
    :title="t('pages.records.tables.edit')"
    form-id="record-table-form"
    :submit-label="t('common.save')"
    :loading="saving"
    :ui="{ content: 'sm:max-w-2xl' }"
    @cancel="attemptClose"
  >
    <UForm
      id="record-table-form"
      ref="form"
      :validate-on="FORM_VALIDATE_ON"
      :state="state"
      :validate="validate"
      class="flex flex-col gap-4"
      @submit="void save()"
    >
      <RecordTableFields v-model:state="state" />
    </UForm>
  </FormModal>
</template>
