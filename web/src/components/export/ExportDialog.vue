<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { TableEncoding, TableSeparator, type TableFormat } from '@/api/types/import';
import { useMutation } from '@/composables/form/useMutation';
import { downloadBlob } from '@/lib/browser/download';

export type ExportColumn = {
  key: string;
  label: string;
};

// Downloads a table as a file in a dialog, with the encoding, the separator, and the columns to export.
// The defaults suit a spreadsheet: UTF-8, which the server marks for it, separated by semicolons,
// with every column. The file is loaded before it is saved, so a failed export shows its error
// instead of saving it as the file.
const props = defineProps<{
  title: string;
  columns: ExportColumn[];
  fileName: string;
  // Loads the table in the format and with the columns, every column without any
  load: (format: TableFormat, columns: string[]) => Promise<Blob>;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();

const encoding = ref<TableEncoding>(TableEncoding.Utf8);
const separator = ref<TableSeparator>(TableSeparator.Semicolon);
const selected = ref<string[]>([]);

// Each opening starts with every column
watch(open, (isOpen) => {
  if (isOpen) selected.value = props.columns.map((column) => column.key);
});

const encodingItems = computed(() =>
  Object.values(TableEncoding).map((value) => ({
    label: t(`tableFormat.encodings.${value}`),
    value,
  })),
);

const separatorItems = computed(() =>
  Object.values(TableSeparator).map((value) => ({
    label: t(`tableFormat.separators.${value}`),
    value,
  })),
);

const columnItems = computed(() =>
  props.columns.map((column) => ({
    label: column.label,
    value: column.key,
  })),
);

const { mutate: download, pending: downloading } = useMutation(
  async () => {
    // The columns in the order of the table, every column as none
    const keys = props.columns
      .map((column) => column.key)
      .filter((key) => selected.value.includes(key));
    return props.load(
      {
        encoding: encoding.value,
        separator: separator.value,
      },
      keys.length === props.columns.length ? [] : keys,
    );
  },
  {
    onSuccess: (blob) => {
      downloadBlob(props.fileName, blob);
      open.value = false;
    },
  },
);
</script>

<template>
  <UModal v-model:open="open" :title="title" :ui="{ content: 'sm:max-w-xl' }">
    <template #body>
      <div class="flex flex-col gap-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <UFormField :label="t('tableFormat.encoding')" name="encoding">
            <USelect v-model="encoding" :items="encodingItems" class="w-full" />
          </UFormField>
          <UFormField :label="t('tableFormat.separator')" name="separator">
            <USelect v-model="separator" :items="separatorItems" class="w-full" />
          </UFormField>
        </div>

        <UFormField :label="t('export.columns')" name="columns">
          <UCheckboxGroup
            v-model="selected"
            :items="columnItems"
            :ui="{ fieldset: 'grid grid-cols-1 gap-2 sm:grid-cols-2' }"
          />
        </UFormField>
      </div>
    </template>

    <template #footer>
      <div class="flex w-full items-center justify-end gap-2">
        <UButton variant="ghost" color="neutral" :disabled="downloading" @click="open = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-lucide-download"
          :loading="downloading"
          :disabled="selected.length === 0"
          @click="void download()"
        >
          {{ t('export.download') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
