<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { recordingRepository } from '@/api/repositories/recording';
import type { RecordingResponse } from '@/api/types';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import { useApiError } from '@/composables/useApiError';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/useConfigForm';
import { useMutation } from '@/composables/form/useMutation';
import { countLabel } from '@/lib/form/validation';

// The transmission speeds the timing devices support.
const BAUD_RATES = [9600, 19200, 38400, 57600, 115200];

// The connection to the timing device, the simulator that replaces it, and the recordings it can replay.
const { t } = useI18n();
const toast = useToast();
const { messageFor } = useApiError();
const { state, fieldProps } = useConfigForm();

const recordings = ref<RecordingResponse[]>([]);
const fileInput = useTemplateRef<HTMLInputElement>('fileInput');

// A configured recording that is not listed, such as one set in the environment, shows the name in its path
const selectedRecordingName = computed(
  () =>
    recordings.value.find((recording) => recording.path === state.timing.replayPath)?.name ??
    state.timing.replayPath.split('/').pop(),
);

/** clearRecording replays no recording. */
function clearRecording(): void {
  state.timing.replayPath = '';
}

/** selectRecording replays the recording stored at the given path. */
function selectRecording(path: string): void {
  state.timing.replayPath = path;
}

/** loadRecordings reads the stored recordings. */
async function loadRecordings(): Promise<void> {
  try {
    recordings.value = await recordingRepository.list();
  } catch (err) {
    toast.add({
      title: messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  }
}

onMounted(loadRecordings);

const { mutate: uploadRecording, pending: uploading } = useMutation(
  (file: File) => recordingRepository.create(file),
  {
    onSuccess: async (recording) => {
      await loadRecordings();
      state.timing.replayPath = recording.path;
    },
  },
);

const { mutate: deleteRecording } = useMutation(
  (name: string) => recordingRepository.delete(name),
  {
    onSuccess: async (_result, name) => {
      if (state.timing.replayPath.endsWith(`/${name}`)) {
        state.timing.replayPath = '';
      }
      await loadRecordings();
    },
  },
);

/** onRecordingFileChange uploads the chosen file and resets the input, so the same file can be chosen again. */
function onRecordingFileChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (file) void uploadRecording(file);
}
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <UFormField :label="t('fields.timing.serialPort')" name="timing.serialPort">
        <template #hint>
          <ConfigFieldHint
            path="timing.serialPort"
            :help="t('pages.config.timing.serialPortHint')"
            :count="countLabel(state.timing.serialPort, CONFIG_LIMITS.timingSerialPort)"
          />
        </template>
        <UInput
          v-model="state.timing.serialPort"
          v-bind="fieldProps('timing.serialPort')"
          :placeholder="t('pages.config.timing.serialPortPlaceholder')"
          :maxlength="CONFIG_LIMITS.timingSerialPort"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.timing.baudRate')" name="timing.baudRate">
        <template #hint>
          <ConfigFieldHint path="timing.baudRate" :help="t('pages.config.timing.baudRateHint')" />
        </template>
        <USelect
          v-model="state.timing.baudRate"
          v-bind="fieldProps('timing.baudRate')"
          :items="BAUD_RATES"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.timing.poolLength')" name="timing.poolLength">
        <template #hint>
          <ConfigFieldHint
            path="timing.poolLength"
            :help="t('pages.config.timing.poolLengthHint')"
          />
        </template>
        <UInputNumber
          v-model="state.timing.poolLength"
          v-bind="fieldProps('timing.poolLength')"
          :min="0"
          :max="1000000"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.timing.simulateEvent')" name="timing.simulateEvent">
        <template #hint>
          <ConfigFieldHint
            path="timing.simulateEvent"
            :help="t('pages.config.timing.simulateEventHint')"
          />
        </template>
        <UInputNumber
          v-model="state.timing.simulateEvent"
          v-bind="fieldProps('timing.simulateEvent')"
          :min="0"
          :max="1000000"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.timing.replayPath')" name="timing.replayPath">
        <template #hint>
          <ConfigFieldHint
            path="timing.replayPath"
            :help="t('pages.config.timing.replayPathHint')"
          />
        </template>
        <USelectMenu
          v-model="state.timing.replayPath"
          v-bind="fieldProps('timing.replayPath')"
          value-key="path"
          label-key="name"
          :items="recordings"
          :placeholder="t('pages.config.timing.recordingNone')"
          class="w-full"
        >
          <template #leading>
            <UButton
              v-if="state.timing.replayPath"
              icon="i-lucide-x"
              size="xs"
              color="neutral"
              variant="ghost"
              :aria-label="t('pages.config.timing.recordingClear')"
              @click.stop="clearRecording"
            />
          </template>
          <template #default="{ ui }">
            <span
              v-if="selectedRecordingName"
              data-slot="value"
              :class="ui.value?.({ class: undefined })"
            >
              {{ selectedRecordingName }}
            </span>
            <span v-else data-slot="placeholder" :class="ui.placeholder?.({ class: undefined })">
              {{ t('pages.config.timing.recordingNone') }}
            </span>
          </template>
        </USelectMenu>
      </UFormField>
      <UFormField :label="t('fields.timing.recording')" name="timing.recording">
        <template #hint>
          <ConfigFieldHint path="timing.recording" :help="t('pages.config.timing.recordingHint')" />
        </template>
        <USwitch v-model="state.timing.recording" v-bind="fieldProps('timing.recording')" />
      </UFormField>
    </div>

    <UCard :ui="{ body: 'flex flex-col gap-3' }" class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <span class="text-highlighted font-medium">
              {{ t('pages.config.timing.recordingLibrary') }}
            </span>
            <HelpHint :text="t('pages.config.timing.recordingLibraryHint')" />
          </div>
          <UButton
            :label="t('pages.config.timing.uploadRecording')"
            icon="i-lucide-upload"
            size="sm"
            :loading="uploading"
            @click="fileInput?.click()"
          />
          <input
            ref="fileInput"
            type="file"
            accept=".jsonl"
            class="hidden"
            @change="onRecordingFileChange"
          />
        </div>
      </template>

      <p v-if="!recordings.length" class="text-muted text-sm">
        {{ t('pages.config.timing.recordingEmpty') }}
      </p>
      <div
        v-for="recording in recordings"
        :key="recording.path"
        class="border-default flex items-center justify-between gap-2 rounded-md border px-3 py-2"
      >
        <span class="truncate text-sm">{{ recording.name }}</span>
        <div class="flex shrink-0 items-center gap-2">
          <UBadge v-if="recording.current" color="neutral" variant="outline">
            {{ t('pages.config.timing.recordingCurrent') }}
          </UBadge>
          <UBadge v-if="state.timing.replayPath === recording.path" color="primary" variant="solid">
            {{ t('pages.config.timing.recordingActive') }}
          </UBadge>
          <UButton
            v-else
            :label="t('pages.config.timing.recordingSelect')"
            size="xs"
            color="neutral"
            variant="outline"
            @click="selectRecording(recording.path)"
          />
          <UButton
            icon="i-lucide-download"
            size="xs"
            color="neutral"
            variant="ghost"
            :to="recordingRepository.downloadUrl(recording.name)"
            target="_blank"
            :aria-label="t('pages.config.timing.recordingDownload')"
          />
          <UButton
            icon="i-lucide-file-spreadsheet"
            size="xs"
            color="neutral"
            variant="ghost"
            :to="recordingRepository.exportUrl(recording.name)"
            target="_blank"
            :aria-label="t('pages.config.timing.recordingExport')"
          />
          <UButton
            v-if="!recording.current"
            icon="i-lucide-trash-2"
            size="xs"
            color="error"
            variant="ghost"
            :aria-label="t('pages.config.timing.recordingDelete')"
            @click="deleteRecording(recording.name)"
          />
        </div>
      </div>
    </UCard>
  </UCard>
</template>
