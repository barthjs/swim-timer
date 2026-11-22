<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import FormModal from '@/components/form/FormModal.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useFormValidation } from '@/composables/form/useFormValidation';
import { usePresetForm } from '@/composables/preset/usePresetForm';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { PRESET_NAME_MAX } from '@/lib/preset/lanes';

// The lane counts a new preset can start with
const LANE_COUNTS = [2, 4, 6, 8, 10];
const DEFAULT_LANE_COUNT = 8;

// Manages the stored presets, in one row with the page title
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

/** Shows the dialog for a new preset with empty fields. */
function openNewPreset(): void {
  newPreset.name = '';
  newPreset.laneCount = DEFAULT_LANE_COUNT;
  newPresetOpen.value = true;
}

/** Starts a draft with the entered name and lane count. */
async function addPreset(): Promise<void> {
  const name = newPreset.name.trim();
  if (!name) return;

  if (await presetForm.addPreset(name, newPreset.laneCount)) {
    newPresetOpen.value = false;
  }
}
</script>

<template>
  <!-- The controls take part in the row of the page header, which places them beside its buttons. -->
  <div class="contents">
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
      class="w-full sm:w-56"
    >
      <!-- A draft is not in the list yet, so the menu names it and marks it instead of showing the placeholder. -->
      <template v-if="presetForm.isDraft.value" #default>
        <span class="flex min-w-0 items-center gap-2">
          <span class="truncate">{{ presetForm.preset.name || t('pages.editor.newPreset') }}</span>
          <UBadge :label="t('pages.editor.draft')" color="info" size="sm" />
        </span>
      </template>
    </USelectMenu>
    <DocsLink page="editor" />
    <UButton
      :label="t('pages.editor.new')"
      :aria-label="t('pages.editor.new')"
      icon="i-lucide-plus"
      color="neutral"
      variant="outline"
      :ui="{ label: 'hidden sm:inline' }"
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

    <div class="bg-accented mx-1 hidden h-5 w-px sm:block"></div>

    <UFileUpload
      v-slot="{ open }"
      :model-value="null"
      reset
      accept=".json,application/json"
      :preview="false"
      :dropzone="false"
      @update:model-value="(file: File | null | undefined) => void presetForm.importPreset(file)"
      @reject="(files: File[]) => void presetForm.importPreset(files[0])"
    >
      <UButton
        :label="t('pages.editor.importPreset')"
        :aria-label="t('pages.editor.importPreset')"
        icon="i-lucide-file-up"
        color="neutral"
        variant="outline"
        :ui="{ label: 'hidden sm:inline' }"
        @click="open()"
      />
    </UFileUpload>
    <UButton
      :label="t('pages.editor.downloadPreset')"
      :aria-label="t('pages.editor.downloadPreset')"
      icon="i-lucide-file-down"
      color="neutral"
      variant="outline"
      :ui="{ label: 'hidden sm:inline' }"
      :disabled="!presetForm.loaded.value"
      @click="presetForm.downloadPreset"
    />

    <FormModal
      v-model:open="newPresetOpen"
      :title="t('pages.editor.newPreset')"
      form-id="new-preset-form"
      :submit-label="t('pages.editor.addPreset')"
      @cancel="newPresetOpen = false"
    >
      <UForm
        id="new-preset-form"
        :validate-on="FORM_VALIDATE_ON"
        :state="newPreset"
        :validate="validate"
        class="flex flex-col gap-4"
        @submit="addPreset"
      >
        <UFormField :label="t('fields.name')" name="name" required>
          <UInput v-model="newPreset.name" :maxlength="PRESET_NAME_MAX" class="w-full" />
        </UFormField>
        <UFormField :label="t('pages.editor.laneCount')" name="laneCount">
          <template #hint>
            <HelpHint :text="t('pages.editor.help.laneCount')" />
          </template>
          <USelect v-model="newPreset.laneCount" :items="LANE_COUNTS" class="w-full" />
        </UFormField>
      </UForm>
    </FormModal>
  </div>
</template>
