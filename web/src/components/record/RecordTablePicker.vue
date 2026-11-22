<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import { recordTableRepository } from '@/api/repositories/recordTable';
import type { TableFormat } from '@/api/types/import';
import type { RecordTableResponse } from '@/api/types/recordTable';
import ExportDialog from '@/components/export/ExportDialog.vue';
import RecordTableFormModal from '@/components/record/RecordTableFormModal.vue';
import RecordTableImportModal from '@/components/record/RecordTableImportModal.vue';
import RecordTableOrderModal from '@/components/record/RecordTableOrderModal.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useRecordTexts } from '@/composables/record/useRecordTexts';
import { RECORD_COLUMNS } from '@/lib/record/record';

// Chooses the record table whose records are shown, and manages the record tables in a menu next to it:
// a new table, an import into the chosen one, its export, the order of the check, and its deletion.
// A pool has a handful of tables, so a choice saves the room of a table of them.
// Every change is saved right away and reported with changed, so the page reloads the tables and their records.
const props = defineProps<{
  tables: RecordTableResponse[];
}>();

// The id of the chosen table, empty for every table
const tableId = defineModel<string>({
  required: true,
});

const emit = defineEmits<{
  changed: [];
}>();

// The select value of every table, since Reka UI rejects an empty string
const ALL_TABLES = 'all';

const { t } = useI18n();
const { columnLabel } = useRecordTexts();
const { confirmDelete } = useDeleteConfirm();

const selected = computed(() => props.tables.find((table) => table.id === tableId.value) ?? null);

const choice = computed({
  get: () => tableId.value || ALL_TABLES,
  set: (value: string) => {
    tableId.value = value === ALL_TABLES ? '' : value;
  },
});

const items = computed(() => [
  {
    label: t('pages.records.tables.all'),
    value: ALL_TABLES,
  },
  ...props.tables.map((table, index) => ({
    label: [
      `${index + 1}.`,
      table.abbreviation,
      table.name,
      table.enabled ? '' : `(${t('pages.records.tables.disabled')})`,
    ]
      .filter(Boolean)
      .join(' '),
    value: table.id,
  })),
]);

const editOpen = ref(false);
const importOpen = ref(false);
const importInto = ref<RecordTableResponse | null>(null);
const exportOpen = ref(false);
const orderOpen = ref(false);

/** Shows the dialog that imports a file into a table, or into a new one without a table. */
function openImport(table: RecordTableResponse | null): void {
  importInto.value = table;
  importOpen.value = true;
}

const exportColumns = computed(() =>
  RECORD_COLUMNS.map((key) => ({
    key,
    label: columnLabel(key),
  })),
);

/** Loads the records of the chosen table in the format and with the columns of the export. */
async function exportTable(format: TableFormat, columns: string[]): Promise<Blob> {
  const table = selected.value;
  if (!table) return Promise.reject(new Error('no record table chosen'));
  return recordTableRepository.export(table.id, {
    format,
    columns,
  });
}

const { mutate: deleteTable } = useMutation(
  async (table: RecordTableResponse) => recordTableRepository.delete(table.id),
  {
    confirm: async (table) =>
      confirmDelete({
        title: t('pages.records.tables.deleteTitle'),
        description: t('pages.records.tables.deleteDescription', {
          name: table.name,
        }),
      }),
    success: (_, table) =>
      t('pages.records.tables.deleted', {
        name: table.name,
      }),
    onSuccess: () => {
      tableId.value = '';
      emit('changed');
    },
  },
);

// The actions of the chosen table are disabled while every table is shown
const menuItems = computed<DropdownMenuItem[][]>(() => {
  const table = selected.value;
  return [
    [
      {
        label: t('pages.records.tables.new'),
        icon: 'i-lucide-plus',
        onSelect: () => openImport(null),
      },
    ],
    [
      {
        label: t('pages.records.tables.edit'),
        icon: 'i-lucide-pencil',
        disabled: !table,
        onSelect: () => {
          editOpen.value = true;
        },
      },
      {
        label: t('pages.records.tables.importAction'),
        icon: 'i-lucide-upload',
        disabled: !table,
        onSelect: () => openImport(table),
      },
      {
        label: t('pages.records.tables.exportAction'),
        icon: 'i-lucide-download',
        disabled: !table,
        onSelect: () => {
          exportOpen.value = true;
        },
      },
    ],
    [
      {
        label: t('pages.records.tables.order'),
        icon: 'i-lucide-arrow-down-up',
        disabled: props.tables.length < 2,
        onSelect: () => {
          orderOpen.value = true;
        },
      },
    ],
    [
      {
        label: t('common.delete'),
        icon: 'i-lucide-trash-2',
        color: 'error',
        disabled: !table,
        onSelect: () => table && void deleteTable(table),
      },
    ],
  ];
});
</script>

<template>
  <!-- On narrow screens the choice takes the free width -->
  <div class="w-full min-w-0 sm:w-auto">
    <UFieldGroup class="flex w-full">
      <USelectMenu
        v-model="choice"
        :items="items"
        value-key="value"
        :search-input="false"
        :aria-label="t('pages.records.tables.choose')"
        icon="i-lucide-table-2"
        class="min-w-0 flex-1 sm:w-64 sm:flex-none"
      />
      <UDropdownMenu :items="menuItems" :content="{ align: 'end' }">
        <UButton
          icon="i-lucide-ellipsis-vertical"
          color="neutral"
          variant="outline"
          :aria-label="t('pages.records.tables.actions')"
        />
      </UDropdownMenu>
    </UFieldGroup>

    <RecordTableOrderModal v-model:open="orderOpen" :tables="tables" @saved="emit('changed')" />
    <RecordTableFormModal v-model:open="editOpen" :table="selected" @saved="emit('changed')" />
    <RecordTableImportModal
      v-model:open="importOpen"
      :table="importInto"
      @imported="emit('changed')"
    />
    <ExportDialog
      v-model:open="exportOpen"
      :title="t('pages.records.tables.exportTitle', { name: selected?.name ?? '' })"
      :columns="exportColumns"
      :file-name="`${selected?.abbreviation ?? ''}.csv`"
      :load="exportTable"
    />
  </div>
</template>
