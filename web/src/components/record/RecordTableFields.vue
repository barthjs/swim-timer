<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { RecordTableUpdateRequest } from '@/api/types/recordTable';
import FieldHint from '@/components/form/FieldHint.vue';
import { countLabel } from '@/lib/form/validation';
import { RECORD_TABLE_ABBREVIATION_MAX, RECORD_TABLE_NAME_MAX } from '@/lib/record/recordTable';

// The name, the abbreviation, and the state of a record table, shared by its import and its edit dialog
const state = defineModel<RecordTableUpdateRequest>('state', {
  required: true,
});

const { t } = useI18n();
</script>

<template>
  <UFormField :label="t('fields.name')" name="name" required>
    <template #hint>
      <FieldHint :count="countLabel(state.name, RECORD_TABLE_NAME_MAX)" />
    </template>
    <UInput v-model="state.name" :maxlength="RECORD_TABLE_NAME_MAX" class="w-full" />
  </UFormField>
  <div class="grid grid-cols-1 items-start gap-4 sm:grid-cols-2">
    <UFormField :label="t('fields.abbreviation')" name="abbreviation" required>
      <template #hint>
        <FieldHint
          :count="countLabel(state.abbreviation, RECORD_TABLE_ABBREVIATION_MAX)"
          :help="t('pages.records.tables.abbreviationHint')"
        />
      </template>
      <UInput
        v-model="state.abbreviation"
        :maxlength="RECORD_TABLE_ABBREVIATION_MAX"
        class="w-full"
      />
    </UFormField>
    <UFormField :label="t('fields.enabled')" name="enabled">
      <template #hint>
        <FieldHint :help="t('pages.records.tables.enabledHint')" />
      </template>
      <div class="flex h-8 items-center">
        <USwitch v-model="state.enabled" />
      </div>
    </UFormField>
  </div>
</template>
