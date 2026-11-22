<script setup lang="ts">
import { computed, reactive, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { recordTableRepository } from '@/api/repositories/recordTable';
import type { ImportMapping } from '@/api/types/import';
import type { RecordTableResponse, RecordTableUpdateRequest } from '@/api/types/recordTable';
import ImportDialog from '@/components/import/ImportDialog.vue';
import RecordTableFields from '@/components/record/RecordTableFields.vue';
import { useFormValidation } from '@/composables/form/useFormValidation';
import { useRecordTexts } from '@/composables/record/useRecordTexts';
import { RECORD_TABLE_RULES } from '@/lib/record/recordTable';

// Creates a record table from a file, or replaces the records of the given table with the records of a file
const props = defineProps<{
  // The table whose records are replaced, null for a new table
  table: RecordTableResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  imported: [];
}>();

const { t } = useI18n();
const { columnLabel, columnHint, valueLabel } = useRecordTexts();

const fields = reactive<RecordTableUpdateRequest>({
  name: '',
  abbreviation: '',
  enabled: true,
});

// A new table starts without a name
watch(open, (isOpen) => {
  if (isOpen) {
    Object.assign(fields, {
      name: '',
      abbreviation: '',
      enabled: true,
    });
  }
});

const { validate } = useFormValidation<RecordTableUpdateRequest>(() =>
  props.table ? {} : RECORD_TABLE_RULES,
);

const title = computed(() =>
  props.table
    ? t('pages.records.tables.importInto', {
        name: props.table.name,
      })
    : t('pages.records.tables.new'),
);

/** Stores a new table with the records of the file, or replaces the records of the table. */
async function submit(file: File, mapping: ImportMapping) {
  if (props.table) {
    return recordTableRepository.import(props.table.id, file, {
      mapping,
    });
  }
  return recordTableRepository.create(file, {
    ...fields,
    mapping,
  });
}
</script>

<template>
  <ImportDialog
    v-model:open="open"
    :title="title"
    :submit-label="t('pages.records.tables.importAction')"
    :fields="fields"
    :validate="validate"
    :column-label="columnLabel"
    :column-hint="columnHint"
    :value-label="valueLabel"
    :read="recordTableRepository.preview"
    :submit="submit"
    :success="t('pages.records.tables.imported')"
    @imported="emit('imported')"
  >
    <template v-if="!table" #fields>
      <RecordTableFields v-model:state="fields" />
    </template>
  </ImportDialog>
</template>
