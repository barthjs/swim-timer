<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { ChevronLeft, RefreshCcw, Upload } from 'lucide-vue-next';

import { raceDataApi } from '@/api/raceDataApi.ts';

import BaseButton from '@/components/BaseButton.vue';

import { useToastStore } from '@/stores/toastStore';

import type { RaceDataFile } from '@/types';
import { getErrorMessageKey, isApiError } from '@/types/api.ts';

const raceFiles = ref<RaceDataFile[]>([]);
const isLoading = ref(true);
const isReloading = ref(false);
const isUploadingMultiple = ref(false);
const multipleFileInput = ref<HTMLInputElement | null>(null);

const { t } = useI18n();

const toastStore = useToastStore();

const fetchStatus = async () => {
  try {
    await raceDataApi.getStatus().then((statuses: Record<string, boolean>) => {
      raceFiles.value = Object.entries(statuses).map(
        ([id, exists]): RaceDataFile => ({
          id,
          exists,
          isUploading: false,
        }),
      );
    });
  } catch (err) {
    const message = isApiError(err) ? t(getErrorMessageKey(err)) : t('raceData.fetchError');
    toastStore.error(message);
    console.error(err);
  } finally {
    isLoading.value = false;
  }
};

const handleUpload = async (fileId: string, event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  const raceFile = raceFiles.value.find((f) => f.id === fileId);
  if (raceFile) raceFile.isUploading = true;

  try {
    await raceDataApi.uploadFile(fileId, file);
    toastStore.success(t('raceData.uploadSuccess', { name: t(`raceData.fileNames.${fileId}`) }));
    await fetchStatus();
  } catch (err) {
    const message = isApiError(err)
      ? t(getErrorMessageKey(err))
      : t('raceData.uploadError', { name: t(`raceData.fileNames.${fileId}`) });
    toastStore.error(message);
    console.error(err);
  } finally {
    if (raceFile) raceFile.isUploading = false;
    target.value = '';
  }
};

const handleMultipleUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  const files = Array.from(target.files || []);
  if (files.length === 0) return;

  isUploadingMultiple.value = true;
  try {
    const results = await raceDataApi.uploadFiles(files);

    let successCount = 0;
    results.forEach((res) => {
      if (res.matched && !res.error) {
        successCount++;
      } else if (!res.matched) {
        toastStore.error(
          `${t('raceData.uploadError', { name: res.filename })}: Not an allowed file`,
          undefined,
          undefined,
          true,
        );
      } else if (res.error) {
        toastStore.error(
          `${t('raceData.uploadError', { name: res.filename })}: ${res.error}`,
          undefined,
          undefined,
          true,
        );
      }
    });

    if (successCount > 0) {
      toastStore.success(t('raceData.uploadMultipleSuccess', { count: successCount }));
      await fetchStatus();
    }
  } catch (err) {
    const message = isApiError(err) ? err.message : t('raceData.uploadMultipleError');
    toastStore.error(message);
    console.error(err);
  } finally {
    isUploadingMultiple.value = false;
    target.value = '';
  }
};

const reloadStore = async () => {
  isReloading.value = true;

  try {
    await raceDataApi.reloadRaceData();
    await fetchStatus();
    toastStore.success(t('raceData.reloadSuccess'));
  } catch (err) {
    const message = isApiError(err) ? t(getErrorMessageKey(err)) : t('raceData.reloadError');
    toastStore.error(message);
    console.error(err);
  } finally {
    isReloading.value = false;
  }
};

onMounted(fetchStatus);
</script>

<template>
  <div class="mx-auto w-full max-w-4xl space-y-6">
    <div class="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
      <div class="flex items-center gap-2">
        <RouterLink :to="{ name: 'home' }">
          <BaseButton
            variant="ghost"
            :icon="ChevronLeft"
            size="icon"
            :title="$t('common.backToHome')"
          />
        </RouterLink>
        <h1 class="text-3xl font-bold">{{ $t('raceData.title') }}</h1>
      </div>

      <div class="flex items-center gap-2">
        <BaseButton
          variant="primary"
          @click="multipleFileInput?.click()"
          :disabled="isUploadingMultiple"
        >
          <Upload class="h-4 w-4 sm:mr-2" :class="{ 'animate-spin': isUploadingMultiple }" />
          <span class="hidden sm:inline">{{ $t('raceData.uploadMultiple') }}</span>
        </BaseButton>
        <input
          ref="multipleFileInput"
          type="file"
          class="hidden"
          multiple
          accept=".csv,.txt"
          @change="handleMultipleUpload"
        />

        <BaseButton variant="primary" @click="reloadStore" :disabled="isReloading">
          <RefreshCcw class="h-4 w-4 sm:mr-2" :class="{ 'animate-spin': isReloading }" />
          <span class="hidden sm:inline">{{ $t('raceData.reload') }}</span>
        </BaseButton>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div
        v-for="file in raceFiles"
        :key="file.id"
        class="bg-surface border-subtle group flex flex-col rounded-xl border p-5 transition-colors"
      >
        <div class="mb-4 flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div
              :class="[
                'h-3 w-3 rounded-full',
                file.exists ? 'bg-success shadow-[0_0_12px_rgba(16,185,129,0.4)]' : 'bg-subtle',
              ]"
            ></div>
            <div>
              <h3 class="text-lg font-semibold">
                {{ $t(`raceData.fileNames.${file.id}`) }}
              </h3>
              <code class="text-muted text-xs">{{ file.id }}</code>
            </div>
          </div>

          <span
            class="bg-base border-subtle rounded border px-2 py-1 font-mono text-xs"
            :class="file.exists ? 'text-success' : 'text-muted'"
          >
            {{ file.exists ? $t('raceData.available') : $t('raceData.missing') }}
          </span>
        </div>

        <div class="border-subtle mt-auto flex items-center justify-between border-t pt-4">
          <p class="text-sm">
            {{ file.exists ? $t('raceData.fileReady') : $t('raceData.fileRequired') }}
          </p>

          <label
            class="bg-primary hover:bg-primary-hover flex cursor-pointer items-center gap-2 rounded-lg px-4 py-2"
          >
            <input
              type="file"
              class="hidden"
              accept=".csv,.txt"
              @change="handleUpload(file.id, $event)"
              :disabled="file.isUploading"
            />
            <Upload v-if="!file.isUploading" class="h-4 w-4" />
            <RefreshCcw v-else class="h-4 w-4 animate-spin" />
            <span>{{ $t('raceData.upload') }}</span>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>
