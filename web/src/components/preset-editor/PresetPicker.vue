<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import HelpHint from '@/components/form/HelpHint.vue';
import { useFormValidation } from '@/composables/form/useFormValidation';
import { usePresetForm } from '@/composables/usePresetForm';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { PRESET_NAME_MAX } from '@/lib/lanes';

// The lane counts a new preset can start with.
const LANE_COUNTS = [2, 4, 6, 8, 10];
const DEFAULT_LANE_COUNT = 8;

// Manages the stored presets, in one row with the page title.
const { t } = useI18n();
const presetForm = usePresetForm();

type NewPreset = { name: string; laneCount: number };

const newPreset = reactive<NewPreset>({
  name: '',
  laneCount: DEFAULT_LANE_COUNT,
});
const newPresetOpen = ref(false);

const { validate } = useFormValidation<NewPreset>(() => ({
  name: {
    required: 'required',
    max: PRESET_NAME_MAX,
  },
}));

const presetSelection = computed<string | undefined>({
  get: () => presetForm.savedName.value ?? undefined,
  set: (name) => {
    if (name) void presetForm.selectPreset(name);
  },
});

/** openNewPreset shows the dialog for a new preset with empty fields. */
function openNewPreset(): void {
  newPreset.name = '';
  newPreset.laneCount = DEFAULT_LANE_COUNT;
  newPresetOpen.value = true;
}

/** addPreset starts a draft with the entered name and lane count. */
async function addPreset(): Promise<void> {
  const name = newPreset.name.trim();
  if (!name) return;

  if (await presetForm.addPreset(name, newPreset.laneCount)) {
    newPresetOpen.value = false;
  }
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <USelectMenu
      v-model="presetSelection"
      :items="presetForm.presetNames.value"
      :search-input="{ placeholder: t('pages.editor.searchPreset') }"
      :placeholder="
        presetForm.presetNames.value.length
          ? t('pages.editor.choosePreset')
          : t('pages.editor.noPresets')
      "
      :disabled="!presetForm.presetNames.value.length"
      :aria-label="t('pages.editor.preset')"
      class="w-56 max-w-full"
    />
    <UButton
      :label="t('pages.editor.new')"
      icon="i-lucide-plus"
      color="neutral"
      variant="outline"
      @click="openNewPreset"
    />
    <UTooltip :text="t('common.delete')">
      <UButton
        icon="i-lucide-trash-2"
        color="error"
        variant="outline"
        :aria-label="t('common.delete')"
        :disabled="!presetForm.savedName.value"
        :loading="presetForm.deleting.value"
        @click="presetForm.deletePreset"
      />
    </UTooltip>

    <div class="bg-accented mx-1 hidden h-5 w-px sm:block" />

    <UFileUpload
      v-slot="{ open }"
      :model-value="null"
      reset
      accept=".json,application/json"
      :preview="false"
      :dropzone="false"
      @update:model-value="(file: File | null | undefined) => void presetForm.importPreset(file)"
    >
      <UButton
        :label="t('pages.editor.importPreset')"
        icon="i-lucide-file-up"
        color="neutral"
        variant="outline"
        @click="open()"
      />
    </UFileUpload>
    <UButton
      :label="t('pages.editor.downloadPreset')"
      icon="i-lucide-file-down"
      color="neutral"
      variant="outline"
      :disabled="!presetForm.loaded.value"
      @click="presetForm.downloadPreset"
    />

    <UModal v-model:open="newPresetOpen" :title="t('pages.editor.newPreset')">
      <template #body>
        <UForm
          :validate-on="FORM_VALIDATE_ON"
          id="new-preset-form"
          :state="newPreset"
          :validate="validate"
          class="flex flex-col gap-4"
          @submit="addPreset"
        >
          <UFormField :label="t('fields.name')" name="name" required>
            <UInput
              v-model="newPreset.name"
              :maxlength="PRESET_NAME_MAX"
              autofocus
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('pages.editor.laneCount')" name="laneCount">
            <template #hint>
              <HelpHint :text="t('pages.editor.help.laneCount')" />
            </template>
            <USelect v-model="newPreset.laneCount" :items="LANE_COUNTS" class="w-full" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton
            :label="t('common.cancel')"
            color="neutral"
            variant="outline"
            @click="newPresetOpen = false"
          />
          <UButton
            type="submit"
            form="new-preset-form"
            :label="t('pages.editor.addPreset')"
            icon="i-lucide-plus"
          />
        </div>
      </template>
    </UModal>
  </div>
</template>
