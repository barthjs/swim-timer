<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';
import type { FileUploadItem } from '@nuxt/ui';

import { meetRepository } from '@/api/repositories/meet';
import { meetFileRepository } from '@/api/repositories/meetFile';
import { type MeetFile, type MeetFileResponse, type MeetFileUploadResponse } from '@/api/types';
import { ApiError } from '@/api/errors';
import HelpHint from '@/components/form/HelpHint.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useApiError } from '@/composables/useApiError';
import { useMutation } from '@/composables/form/useMutation';

// The extension the timing software exports its files with. The browser matches it regardless of case.
const ACCEPTED_FILES = '.txt';

const { t } = useI18n();
const { messageFor } = useApiError();
const toast = useToast();

const files = ref<MeetFileResponse[]>([]);
const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

const uploadingFile = ref<MeetFile | null>(null);

/** load reads which meet files are stored. */
async function load(): Promise<void> {
  try {
    files.value = await meetFileRepository.list();
    loadFailed.value = false;
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

const { mutate: upload } = useMutation(
  async (file: MeetFile, content: File) => {
    uploadingFile.value = file;
    try {
      await meetFileRepository.save(file, content);
    } finally {
      uploadingFile.value = null;
    }
  },
  {
    success: (_result, file) =>
      t('pages.meetData.uploadSuccess', {
        name: fileName(file),
      }),
    onSuccess: load,
  },
);

/**
 * reportUploadResults reports every file the server did not accept on its own,
 * so one unknown file does not hide the others.
 */
function reportUploadResults(results: MeetFileUploadResponse[]): void {
  const stored = results.filter((result) => !result.error);

  for (const result of results) {
    if (result.error) reportUploadError(result.filename, result.error);
  }

  if (stored.length > 0) {
    toast.add({
      title: t(
        'pages.meetData.uploadManySuccess',
        {
          count: stored.length,
        },
        stored.length,
      ),
      icon: 'i-lucide-check-circle',
      color: 'success',
    });
  }
}

/** reportUploadError reports one file that was not stored, with the reason as its description. */
function reportUploadError(filename: string, error: string): void {
  toast.add({
    title: t('pages.meetData.uploadError', {
      name: filename,
    }),
    description: t(`pages.meetData.uploadErrors.${error}`),
    icon: 'i-lucide-circle-alert',
    color: 'error',
    duration: 0,
  });
}

const { mutate: uploadMany, pending: uploadingMany } = useMutation(
  (contents: File[]) => meetFileRepository.upload(contents),
  {
    onSuccess: async (results) => {
      reportUploadResults(results);
      await load();
    },
  },
);

const { mutate: reload, pending: reloading } = useMutation(() => meetRepository.reload(), {
  success: () => t('pages.meetData.reloadSuccess'),
  onSuccess: load,
  onError: (err) => {
    // A file that cannot be parsed is reported with the parse error, since the admin has to fix the export
    toast.add({
      title: t('pages.meetData.reloadError'),
      description:
        err instanceof ApiError && err.code === 'bad_request' ? err.message : messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
      duration: 0,
    });
    return true;
  },
});

/** fileName returns the display name of a meet file. */
function fileName(file: MeetFile): string {
  return t(`pages.meetData.fileNames.${file}`);
}

/** onFileSelected uploads the file chosen for one meet file. */
function onFileSelected(file: MeetFile, content: File | FileUploadItem | null | undefined): void {
  if (content instanceof File) void upload(file, content);
}

/** onFilesSelected uploads several files at once. */
function onFilesSelected(contents: (File | FileUploadItem)[] | null | undefined): void {
  // The model type also allows custom file items, which a file dialog or a drop never produces
  const selected = contents?.filter((content) => content instanceof File) ?? [];
  if (selected.length) void uploadMany(selected);
}

/** onFilesRejected reports dropped files whose type the upload does not accept, which never reach the server. */
function onFilesRejected(contents: File[]): void {
  for (const content of contents) {
    reportUploadError(content.name, 'unsupported_type');
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-1">
        <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.meetData.title') }}</h1>
        <DocsLink page="meetData" />
      </div>

      <div class="flex items-center gap-1.5">
        <UButton
          :label="t('pages.meetData.reload')"
          icon="i-lucide-refresh-ccw"
          :loading="reloading"
          @click="reload()"
        />
        <HelpHint :text="t('pages.meetData.reloadHint')" />
      </div>
    </div>

    <p class="text-muted text-sm">{{ t('pages.meetData.hint') }}</p>

    <UFileUpload
      :model-value="null"
      multiple
      reset
      :accept="ACCEPTED_FILES"
      :preview="false"
      :disabled="uploadingMany"
      icon="i-lucide-upload"
      :label="t('pages.meetData.uploadMany')"
      :description="t('pages.meetData.uploadManyHint')"
      class="min-h-32 w-full"
      @update:model-value="onFilesSelected"
      @reject="onFilesRejected"
    />

    <div class="flex items-center gap-1">
      <h2 class="text-highlighted font-semibold">{{ t('fields.files') }}</h2>
      <HelpHint :text="t('pages.meetData.filesHint')" />
    </div>

    <div v-if="loading" class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <USkeleton v-for="i in 6" :key="i" class="h-20 w-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.meetData.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <UCard v-for="item in files" :key="item.file" :ui="{ body: 'flex items-center gap-4' }">
        <div
          class="flex size-10 shrink-0 items-center justify-center rounded-lg"
          :class="item.available ? 'bg-success/10 text-success' : 'bg-elevated text-muted'"
        >
          <UIcon
            :name="item.available ? 'i-lucide-file-check' : 'i-lucide-file-x'"
            class="size-5"
          />
        </div>

        <div class="min-w-0 flex-1">
          <p class="text-highlighted truncate font-semibold">{{ fileName(item.file) }}</p>
          <p class="text-muted truncate font-mono text-xs">
            {{ item.filename || t('pages.meetData.expectedFilename', { file: item.file }) }}
          </p>
        </div>

        <UBadge
          :label="item.available ? t('pages.meetData.available') : t('pages.meetData.missing')"
          :color="item.available ? 'success' : 'neutral'"
          variant="solid"
          class="shrink-0"
        />

        <UFileUpload
          v-slot="{ open }"
          :model-value="null"
          reset
          :accept="ACCEPTED_FILES"
          :preview="false"
          :dropzone="false"
          :disabled="uploadingFile !== null"
          class="shrink-0"
          @update:model-value="(content) => onFileSelected(item.file, content)"
        >
          <UButton
            :label="t('pages.meetData.upload')"
            icon="i-lucide-upload"
            color="neutral"
            variant="outline"
            :loading="uploadingFile === item.file"
            :disabled="uploadingFile !== null && uploadingFile !== item.file"
            @click="open()"
          />
        </UFileUpload>
      </UCard>
    </div>
  </div>
</template>
