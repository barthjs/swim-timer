<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorHelpFrame, { type HelpShortcut } from '@/components/editor/EditorHelpFrame.vue';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { PRESET_EDITOR_SHORTCUTS, type PresetEditorShortcut } from '@/lib/presetEditorShortcuts';

// Explains what the preset editor is for and the usual workflow, and lists its keyboard shortcuts.
const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();

// The number of steps of the guide in the locales.
const STEP_COUNT = 6;

const steps = computed(() =>
  Array.from(
    {
      length: STEP_COUNT,
    },
    (_, index) => t(`pages.editor.guide.steps.${index + 1}`),
  ),
);

// The shortcuts of the preset editor itself and how the lanes are selected and moved.
const shortcuts = computed<HelpShortcut[]>(() => [
  ...(Object.keys(PRESET_EDITOR_SHORTCUTS) as PresetEditorShortcut[]).map((shortcut) => ({
    keys: shortcutKeys(PRESET_EDITOR_SHORTCUTS[shortcut]),
    label: t(`pages.editor.guide.shortcuts.${shortcut}`),
  })),
  {
    keys: ['1', '…', '0'],
    label: t('pages.editor.guide.shortcuts.selectLane'),
  },
  {
    keys: ['arrows'],
    label: t('pages.editor.guide.shortcuts.nudge'),
  },
  {
    keys: ['shift', 'arrows'],
    label: t('pages.editor.guide.shortcuts.nudgeLarge'),
  },
  {
    keys: ['shift'],
    label: t('pages.editor.guide.shortcuts.shiftSelect'),
  },
]);
</script>

<template>
  <EditorHelpFrame v-model:open="open" :shortcuts="shortcuts">
    <template #guide>
      <p>{{ t('pages.editor.guide.intro') }}</p>

      <ol class="flex list-decimal flex-col gap-2 pl-5">
        <li v-for="(step, index) in steps" :key="index">{{ step }}</li>
      </ol>

      <div class="flex flex-col gap-2">
        <h3 class="text-highlighted font-semibold">
          {{ t('pages.editor.guide.presetsTitle') }}
        </h3>
        <p>{{ t('pages.editor.guide.presets') }}</p>
        <p>{{ t('pages.editor.guide.importExport') }}</p>
      </div>

      <div class="flex flex-col gap-2">
        <h3 class="text-highlighted font-semibold">
          {{ t('pages.editor.guide.viewTitle') }}
        </h3>
        <p class="whitespace-pre-line">{{ t('pages.editor.guide.view') }}</p>
      </div>
    </template>
  </EditorHelpFrame>
</template>
