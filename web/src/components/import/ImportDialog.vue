<script setup lang="ts" generic="TFields extends Record<string, unknown>">
import { computed, reactive, ref, useId, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { FileUploadItem, FormError } from '@nuxt/ui';

import { ApiError, type ApiFieldError } from '@/api/client/errors';
import {
  TableEncoding,
  TableSeparator,
  type ImportColumnResponse,
  type ImportMapping,
  type ImportPreviewResponse,
  type TableFormat,
} from '@/api/types/import';
import FormModal from '@/components/form/FormModal.vue';
import ImportStep from '@/components/import/ImportStep.vue';
import { useDirtyForm } from '@/composables/form/useDirtyForm';
import { useDiscardConfirm } from '@/composables/form/useDiscardConfirm';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { useValidationMessages } from '@/composables/form/useValidationMessages';
import { useApiError } from '@/composables/form/useApiError';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import {
  cleanImportMapping,
  emptyTableFormat,
  IMPORT_UNSET,
  importTargets,
  importValueSources,
  splitImportErrors,
  type ImportErrors,
} from '@/lib/import/mapping';
import { isOneOf } from '@/lib/types/guards';

// Imports a table from a file in a dialog with three numbered steps.
// The file is previewed as soon as it is chosen, with its encoding and separator detected.
// The columns of the file are then assigned to the columns of the target in a table,
// and the values a column with fixed or suggested values reads differently are translated,
// both starting from the guess of the server. Other fields of the form, such as a name, sit in the #fields slot above the steps
// and belong to fields, whose errors the form shows. A failed import lists the failed rows.
// Closing asks first once a file is chosen or a field changed.
const props = defineProps<{
  title: string;
  submitLabel: string;
  // The other fields of the form, owned by the parent and edited in the #fields slot
  fields: TFields;
  validate?: (state: TFields) => FormError[];
  // Returns the label of a column of the target by its key
  columnLabel: (key: string) => string;
  // Describes what a column of the target holds, empty for a column that needs no description
  columnHint?: (key: string) => string;
  // Describes a value of a column of the target, such as the name of an abbreviation, empty without one
  valueLabel?: (key: string, value: string) => string;
  // Reads the start of a file and guesses its mapping, until signal cancels the reading
  read: (file: File, format: TableFormat, signal: AbortSignal) => Promise<ImportPreviewResponse>;
  submit: (file: File, mapping: ImportMapping) => Promise<unknown>;
  // The title of the toast after a successful import
  success: string;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  imported: [];
}>();

// The extensions of the delimited files an import reads. The browser matches them regardless of case.
const ACCEPTED_FILES = '.csv,.tsv,.txt';

const { t } = useI18n();
const { messageFor } = useApiError();
const { mapFieldErrors } = useValidationMessages();
const formId = useId();
const form = useTemplateRef<FormErrorTarget>('form');

const file = ref<File | null>(null);
const format = reactive<TableFormat>(emptyTableFormat());
const preview = ref<ImportPreviewResponse | null>(null);
const previewing = ref(false);
const previewError = ref('');
const columns = ref<Record<string, string>>({});
const values = ref<Record<string, Record<string, string>>>({});
const errors = ref<ImportErrors | null>(null);
// The untranslated values are only marked after a first attempt to import, so a fresh preview is not red all over
const showUntranslated = ref(false);
// The encoding and the separator are detected, so their selects stay hidden until they are needed
const formatOpen = ref(false);
// Cancels the preview in flight, so a slower answer for an earlier file or format never replaces a later one
let previewController: AbortController | null = null;

// The parent resets its fields when it opens the dialog, before this watcher takes them as the baseline
const { isDirty: fieldsDirty, capture: captureFields } = useDirtyForm(props.fields);
const isDirty = computed(() => file.value !== null || fieldsDirty.value);

// Each opening starts without a file, and closing cancels the preview in flight
watch(open, (isOpen) => {
  if (!isOpen) {
    cancelPreview();
    return;
  }
  captureFields();
  file.value = null;
  Object.assign(format, emptyTableFormat());
  preview.value = null;
  previewError.value = '';
  columns.value = {};
  values.value = {};
  errors.value = null;
  showUntranslated.value = false;
  formatOpen.value = false;
});

const encodingItems = computed(
  (): { label: string; value: TableEncoding | typeof IMPORT_UNSET }[] => [
    {
      label: t('import.detect'),
      value: IMPORT_UNSET,
    },
    {
      label: t('tableFormat.encodings.utf-8'),
      value: TableEncoding.Utf8,
    },
    {
      label: t('tableFormat.encodings.windows-1252'),
      value: TableEncoding.Windows1252,
    },
  ],
);

const separatorItems = computed(
  (): { label: string; value: TableSeparator | typeof IMPORT_UNSET }[] => [
    {
      label: t('import.detect'),
      value: IMPORT_UNSET,
    },
    ...Object.values(TableSeparator).map((separator) => ({
      label: t(`tableFormat.separators.${separator}`),
      value: separator,
    })),
  ],
);

const TABLE_ENCODINGS = Object.values(TableEncoding);
const TABLE_SEPARATORS = Object.values(TableSeparator);

// The selects cannot hold an empty value, so a detected field is shown as IMPORT_UNSET
const encoding = computed({
  get: () => format.encoding || IMPORT_UNSET,
  set: (value: string) => {
    format.encoding = isOneOf(TABLE_ENCODINGS, value) ? value : '';
  },
});
const separator = computed({
  get: () => format.separator || IMPORT_UNSET,
  set: (value: string) => {
    format.separator = isOneOf(TABLE_SEPARATORS, value) ? value : '';
  },
});

const detectedFormat = computed(() => {
  const detected = preview.value?.format;
  if (!detected?.encoding || !detected.separator) return '';
  return t('import.detected', {
    encoding: t(`tableFormat.encodings.${detected.encoding}`),
    separator: t(`tableFormat.separators.${detected.separator}`),
  });
});

/** Previews a chosen file. */
function onFileSelected(content: File | FileUploadItem | null | undefined): void {
  file.value = content instanceof File ? content : null;
  void loadPreview();
}

/** Reports a dropped file whose type the import does not read, which never reaches the server. */
function onFileRejected(contents: File[]): void {
  file.value = null;
  preview.value = null;
  previewError.value = t('import.fileRejected', {
    name: contents[0]?.name ?? '',
  });
}

// A changed format reads the file again, since its header and values depend on it
watch(format, () => void loadPreview());

/**
 * Reads the start of the file and starts the mapping from the guess of the server.
 * It cancels the preview before, whose answer is then dropped.
 */
async function loadPreview(): Promise<void> {
  cancelPreview();
  errors.value = null;
  previewError.value = '';
  if (!file.value) {
    preview.value = null;
    return;
  }

  const controller = new AbortController();
  previewController = controller;
  previewing.value = true;
  try {
    const result = await props.read(
      file.value,
      {
        ...format,
      },
      controller.signal,
    );
    if (controller.signal.aborted) return;
    preview.value = result;
    columns.value = Object.fromEntries(
      result.columns.map((column) => [
        column.key,
        result.mapping.columns[column.key] ?? IMPORT_UNSET,
      ]),
    );
    values.value = structuredClone(result.mapping.values);
  } catch (err) {
    if (controller.signal.aborted) return;
    preview.value = null;
    const fileErrors = err instanceof ApiError ? err.fields?.file : undefined;
    previewError.value = fileErrors ? fieldMessage('file', fileErrors) : messageFor(err);
  } finally {
    if (previewController === controller) {
      previewController = null;
      previewing.value = false;
    }
  }
}

/** Cancels the preview in flight, if there is one. */
function cancelPreview(): void {
  previewController?.abort();
  previewController = null;
  previewing.value = false;
}

const headerItems = computed(() => [
  {
    label: t('import.unassigned'),
    value: IMPORT_UNSET,
  },
  ...(preview.value?.header ?? [])
    .filter((header) => header !== '')
    .map((header) => ({
      label: header,
      value: header,
    })),
]);

/** Returns the first value of the sample in the header assigned to a column, empty without one. */
function example(key: string): string {
  const header = columns.value[key];
  const index = preview.value?.header.indexOf(header ?? '') ?? -1;
  if (index < 0) return '';
  return preview.value?.sample.find((row) => row[index])?.[index] ?? '';
}

// The mapping as it is shown, with the values of a column translated that the server could not guess
const mapping = computed<ImportMapping>(() => ({
  format: {
    encoding: format.encoding || preview.value?.format.encoding || '',
    separator: format.separator || preview.value?.format.separator || '',
  },
  columns: columns.value,
  values: values.value,
}));

// The columns whose values can be translated, with the values of the file in their assigned header
const valueColumns = computed(() => {
  if (!preview.value) return [];
  const cleaned = cleanImportMapping(mapping.value);
  return preview.value.columns
    .map((column) => ({
      column,
      sources: importValueSources(column, cleaned, preview.value?.values ?? {}),
    }))
    .filter((item) => item.sources.length > 0);
});

/** Returns a value of a column followed by its description, if it has one. */
function targetLabel(key: string, value: string): string {
  const description = props.valueLabel?.(key, value);
  return description ? `${value} – ${description}` : value;
}

/** Returns the values a value of a file can be translated into, with keeping it for free text. */
function targetItems(column: ImportColumnResponse) {
  const targets = importTargets(column).map((target) => ({
    label: targetLabel(column.key, target),
    value: target,
  }));
  if (column.choices.length) return targets;
  return [
    {
      label: t('import.keepValue'),
      value: IMPORT_UNSET,
    },
    ...targets,
  ];
}

/**
 * Returns what a value of the file is translated into.
 * A value of free text is kept without a translation, IMPORT_UNSET,
 * while a value of a column with choices has none yet, undefined, and shows the placeholder.
 */
function valueTarget(column: ImportColumnResponse, value: string): string | undefined {
  const target = values.value[column.key]?.[value];
  if (target !== undefined && target !== IMPORT_UNSET) return target;
  return column.choices.length ? undefined : IMPORT_UNSET;
}

/** Translates a value of the file into a value of the column. */
function setValueTarget(column: ImportColumnResponse, value: string, target: string): void {
  values.value = {
    ...values.value,
    [column.key]: {
      ...values.value[column.key],
      [value]: target,
    },
  };
}

/** Returns the message of the first failed rule of a field. */
function fieldMessage(field: string, fieldErrors: ApiFieldError[]): string {
  return (
    mapFieldErrors({
      [field]: fieldErrors,
    })[0]?.message ?? ''
  );
}

/** Returns the message of a failed mapping of a column, empty without one. */
function columnError(key: string): string {
  const failed = errors.value?.columns[key];
  return failed ? fieldMessage(key, failed) : '';
}

/** Returns the message of a failed translation of the values of a column, empty without one. */
function valuesError(key: string): string {
  const failed = errors.value?.values[key];
  if (failed) return fieldMessage(key, failed);
  const missing = untranslated.value[key];
  return showUntranslated.value && missing
    ? t('import.untranslated', { values: missing.join(', ') }, missing.length)
    : '';
}

// The values of a column with choices that are neither a choice nor translated into one, by the key of the column.
// Every row with such a value would fail, so the import waits until they are translated.
const untranslated = computed(() => {
  const result: Record<string, string[]> = {};
  for (const { column, sources } of valueColumns.value) {
    if (!column.choices.length) continue;
    const missing = sources.filter(
      (source) => !column.choices.includes(source) && valueTarget(column, source) === undefined,
    );
    if (missing.length) result[column.key] = missing;
  }
  return result;
});

// The failed rows with the message of every failed column, the whole row under the field row
const rowErrors = computed(() =>
  (errors.value?.rows ?? []).map((row) => ({
    line: row.line,
    messages: Object.entries(row.fields).map(([key, fieldErrors]) =>
      fieldMessage(key === '' ? 'row' : key, fieldErrors),
    ),
  })),
);

const fileError = computed(() => {
  const failed = errors.value?.file;
  return failed !== undefined && failed.length > 0
    ? fieldMessage('file', failed)
    : previewError.value;
});

const { mutate: save, pending: saving } = useMutation(
  async () => {
    // The submit button is disabled until a file is previewed
    if (!file.value) throw new Error('no file to import');
    await props.submit(file.value, cleanImportMapping(mapping.value));
  },
  {
    success: () => props.success,
    onSuccess: () => {
      open.value = false;
      emit('imported');
    },
    onError: (err) => {
      if (!(err instanceof ApiError) || !err.fields) return undefined;
      errors.value = splitImportErrors(err.fields);
      form.value?.setErrors(mapFieldErrors(errors.value.form, props.fields));
      return true;
    },
  },
);

const { openProxy, attemptClose } = useDiscardConfirm(open, isDirty, saving);

/** Imports the file once the fields of the form are valid. */
function onSubmit(): void {
  errors.value = null;
  showUntranslated.value = true;
  if (Object.keys(untranslated.value).length) return;
  void save();
}
</script>

<template>
  <FormModal
    v-model:open="openProxy"
    :title="title"
    :form-id="formId"
    :submit-label="submitLabel"
    :loading="saving"
    :disabled="!preview || previewing"
    :ui="{ content: 'sm:max-w-4xl' }"
    @cancel="attemptClose"
  >
    <UForm
      :id="formId"
      ref="form"
      :validate-on="FORM_VALIDATE_ON"
      :state="fields"
      :validate="validate"
      class="flex flex-col gap-8"
      @submit="onSubmit"
    >
      <div v-if="$slots.fields" class="flex flex-col gap-4">
        <slot name="fields" />
      </div>

      <ImportStep
        :number="1"
        :title="t('import.steps.file')"
        :description="t('import.steps.fileHint')"
      >
        <UFormField name="file" :error="fileError || undefined">
          <UFileUpload
            :model-value="file"
            :accept="ACCEPTED_FILES"
            :preview="false"
            icon="i-lucide-file-spreadsheet"
            :label="file ? file.name : t('import.chooseFile')"
            :description="t('import.chooseFileHint')"
            class="min-h-24 w-full"
            @update:model-value="onFileSelected"
            @reject="onFileRejected"
          />
        </UFormField>

        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-muted text-xs">{{ detectedFormat }}</span>
          <UButton
            :label="formatOpen ? t('import.hideFormat') : t('import.changeFormat')"
            :trailing-icon="formatOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            color="neutral"
            variant="link"
            size="xs"
            @click="formatOpen = !formatOpen"
          />
        </div>

        <div v-if="formatOpen" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <UFormField :label="t('tableFormat.encoding')" name="encoding">
            <USelect v-model="encoding" :items="encodingItems" class="w-full" />
          </UFormField>
          <UFormField :label="t('tableFormat.separator')" name="separator">
            <USelect v-model="separator" :items="separatorItems" class="w-full" />
          </UFormField>
        </div>
      </ImportStep>

      <div v-if="previewing" class="flex flex-col gap-2">
        <USkeleton v-for="i in 6" :key="i" class="h-10 w-full" />
      </div>

      <template v-else-if="preview">
        <ImportStep
          :number="2"
          :title="t('import.steps.columns')"
          :description="t('import.steps.columnsHint')"
        >
          <div class="border-default divide-default divide-y rounded-md border">
            <div
              class="bg-elevated/50 text-muted grid-cols-import-columns hidden gap-4 px-3 py-2 text-xs font-medium sm:grid"
            >
              <span>{{ t('import.target') }}</span>
              <span>{{ t('import.source') }}</span>
              <span>{{ t('import.exampleValue') }}</span>
            </div>
            <div
              v-for="column in preview.columns"
              :key="column.key"
              class="sm:grid-cols-import-columns grid grid-cols-1 items-center gap-x-4 gap-y-1.5 px-3 py-2.5"
            >
              <div class="flex min-w-0 flex-col">
                <span class="text-highlighted text-sm font-medium">
                  {{ columnLabel(column.key) }}
                  <span v-if="column.required" class="text-error">*</span>
                </span>
                <span v-if="columnHint?.(column.key)" class="text-muted text-xs">
                  {{ columnHint(column.key) }}
                </span>
              </div>
              <UFormField
                :name="`mapping.columns.${column.key}`"
                :error="columnError(column.key) || undefined"
              >
                <USelect
                  v-model="columns[column.key]"
                  :items="headerItems"
                  :aria-label="columnLabel(column.key)"
                  class="w-full"
                />
              </UFormField>
              <span class="text-muted truncate font-mono text-xs" :title="example(column.key)">
                {{ example(column.key) }}
              </span>
            </div>
          </div>
        </ImportStep>

        <ImportStep
          v-if="valueColumns.length"
          :number="3"
          :title="t('import.steps.values')"
          :description="t('import.steps.valuesHint')"
        >
          <UFormField
            v-for="item in valueColumns"
            :key="item.column.key"
            :name="`mapping.values.${item.column.key}`"
            :error="valuesError(item.column.key) || undefined"
          >
            <div class="border-default divide-default divide-y rounded-md border">
              <div class="bg-elevated/50 flex flex-col px-3 py-2">
                <span class="text-highlighted text-sm font-medium">
                  {{ columnLabel(item.column.key) }}
                </span>
                <span class="text-muted text-xs">
                  {{
                    item.column.choices.length
                      ? t('import.fixedValues', {
                          values: item.column.choices.join(', '),
                        })
                      : t('import.suggestedValues')
                  }}
                </span>
              </div>
              <div
                v-for="source in item.sources"
                :key="source"
                class="grid-cols-import-values grid items-center gap-3 px-3 py-2"
              >
                <span class="text-default truncate font-mono text-sm" :title="source">
                  {{ source }}
                </span>
                <UIcon name="i-lucide-arrow-right" class="text-dimmed size-4" />
                <USelect
                  :model-value="valueTarget(item.column, source)"
                  :items="targetItems(item.column)"
                  :placeholder="t('import.chooseValue')"
                  :aria-label="`${columnLabel(item.column.key)}: ${source}`"
                  class="w-full"
                  @update:model-value="
                    (target: string) => setValueTarget(item.column, source, target)
                  "
                />
              </div>
            </div>
          </UFormField>
        </ImportStep>
      </template>

      <UAlert
        v-if="rowErrors.length"
        :title="t('import.rowErrors', { count: rowErrors.length }, rowErrors.length)"
        icon="i-lucide-circle-alert"
        color="error"
        variant="subtle"
      >
        <template #description>
          <ul class="max-h-48 overflow-y-auto">
            <li v-for="row in rowErrors" :key="row.line">
              <span class="font-semibold">{{ t('import.line', { line: row.line }) }}:</span>
              {{ row.messages.join(' ') }}
            </li>
          </ul>
        </template>
      </UAlert>
    </UForm>
  </FormModal>
</template>
