<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import HelpHint from '@/components/form/HelpHint.vue';
import { usePresetForm } from '@/composables/usePresetForm';
import { useShortcutLabels } from '@/composables/useShortcutLabels';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { PRESET_EDITOR_SHORTCUTS } from '@/lib/presetEditorShortcuts';
import { CORNERS } from '@/lib/lanes';

// Pixel coordinates show at most two decimal places, which only appear after the reference resolution changed.
// Typed values snap to whole pixels.
const FORMAT_OPTIONS = {
  maximumFractionDigits: 2,
};

// The corner coordinates of the single selected lane in pixels of the reference resolution, and its lock.
// The field names use the notation of the server's validation errors, lanes[1].tl.x,
// while the dirty check addresses the same value by its dot-separated path in the state, lanes.1.tl.x.
const { t } = useI18n();
const presetForm = usePresetForm();
const { dirtyForm } = presetForm;
const { keyLabels } = useShortcutLabels();
</script>

<template>
  <div
    v-if="presetForm.editor.selection.active.value && presetForm.activeLane.value"
    class="flex flex-col gap-3"
  >
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-1">
        <h3 class="text-highlighted font-semibold">
          {{ t('pages.editor.lane', { number: presetForm.editor.selection.active.value }) }}
        </h3>
        <HelpHint :text="t('pages.editor.help.lane')" />
      </div>
      <UTooltip
        :text="
          presetForm.activeLane.value.locked ? t('pages.editor.unlock') : t('pages.editor.lock')
        "
        :kbds="keyLabels(shortcutKeys(PRESET_EDITOR_SHORTCUTS.lock))"
      >
        <UButton
          :icon="presetForm.activeLane.value.locked ? 'i-lucide-lock' : 'i-lucide-lock-open'"
          color="neutral"
          :variant="presetForm.activeLane.value.locked ? 'solid' : 'outline'"
          :aria-label="
            presetForm.activeLane.value.locked ? t('pages.editor.unlock') : t('pages.editor.lock')
          "
          @click="presetForm.toggleLock"
        />
      </UTooltip>
    </div>

    <div
      v-if="!presetForm.activeLane.value.locked"
      class="grid grid-cols-[auto_1fr_1fr] items-center gap-x-2 gap-y-1"
    >
      <span></span>
      <span class="text-toned text-center text-xs font-medium">
        {{ t('pages.editor.coordinateX') }}
      </span>
      <span class="text-toned text-center text-xs font-medium">
        {{ t('pages.editor.coordinateY') }}
      </span>

      <template v-for="corner in CORNERS" :key="corner">
        <span class="text-default text-sm font-medium">
          {{ t(`pages.editor.corners.${corner}`) }}
        </span>
        <UFormField :name="`lanes[${presetForm.editor.selection.active.value}].${corner}.x`">
          <UInputNumber
            v-model="presetForm.activeLane.value[corner].x"
            orientation="vertical"
            v-bind="
              dirtyForm.fieldProps(`lanes.${presetForm.editor.selection.active.value}.${corner}.x`)
            "
            :min="0"
            :max="presetForm.preset.frameWidth"
            :format-options="FORMAT_OPTIONS"
            :increment="false"
            :decrement="false"
            class="w-full"
            :ui="{ base: 'px-2 text-center font-mono' }"
            @update:model-value="() => presetForm.editor.history.commit()"
          />
        </UFormField>
        <UFormField :name="`lanes[${presetForm.editor.selection.active.value}].${corner}.y`">
          <UInputNumber
            v-model="presetForm.activeLane.value[corner].y"
            orientation="vertical"
            v-bind="
              dirtyForm.fieldProps(`lanes.${presetForm.editor.selection.active.value}.${corner}.y`)
            "
            :min="0"
            :max="presetForm.preset.frameHeight"
            :format-options="FORMAT_OPTIONS"
            :increment="false"
            :decrement="false"
            class="w-full"
            :ui="{ base: 'px-2 text-center font-mono' }"
            @update:model-value="() => presetForm.editor.history.commit()"
          />
        </UFormField>
      </template>
    </div>
  </div>
</template>
