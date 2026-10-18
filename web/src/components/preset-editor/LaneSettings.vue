<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorItemList, { type EditorListItem } from '@/components/editor/EditorItemList.vue';
import EditorPanel from '@/components/editor/EditorPanel.vue';
import EditorTransformFields from '@/components/editor/EditorTransformFields.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import LaneCornerFields from '@/components/preset-editor/LaneCornerFields.vue';
import { usePresetForm } from '@/composables/usePresetForm';
import { useShortcutLabels } from '@/composables/useShortcutLabels';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { PRESET_EDITOR_SHORTCUTS } from '@/lib/presetEditorShortcuts';
import { unionBounds } from '@/lib/editor/geometry';
import { LANE_GAP_MAX, LANE_GAP_MIN, laneBounds } from '@/lib/lanes';

// The automatic alignment with the gap between the lanes, the list of the lanes, and the corners of the selected lane
// or the position and size of several selected ones.
const { t } = useI18n();
const presetForm = usePresetForm();
const { preset, dirtyForm, editor } = presetForm;
const { selection, visibility } = editor;
const { keyLabels } = useShortcutLabels();

// The lanes in the order of their numbers
const items = computed<EditorListItem<string>[]>(() =>
  Object.keys(preset.lanes)
    .sort((a, b) => Number(a) - Number(b))
    .map((key) => ({
      key,
      label: t('pages.editor.lane', {
        number: key,
      }),
      locked: preset.lanes[key]!.locked,
      hidden: visibility.isHidden(key),
    })),
);

// The bounds of the selected lanes that can move, which the transform fields show
const movableBounds = computed(() => unionBounds(presetForm.movableLanes.value.map(laneBounds)));

/** onSelect selects a lane of the list, alone or in addition to the selection. */
function onSelect(key: string, additive: boolean): void {
  if (additive) {
    selection.toggle(key);
    return;
  }
  selection.selectOnly(key);
}

/** onToggleLock locks or unlocks a lane of the list. */
function onToggleLock(key: string): void {
  const lane = preset.lanes[key];
  if (lane) presetForm.setLocked(key, !lane.locked);
}

/** onLaneGap changes the gap between the lanes, which distributes them again. */
function onLaneGap(laneGap: number | null | undefined): void {
  if (typeof laneGap === 'number') presetForm.setLaneGap(laneGap);
}
</script>

<template>
  <EditorPanel :title="t('pages.editor.lanes')" storage-key="editor.panel.lanes">
    <template #summary>
      <HelpHint :text="t('pages.editor.help.alignLanes')" />
    </template>

    <!-- In a narrow sidebar the button moves below the field and takes the whole width. -->
    <div class="flex flex-col gap-2 @xs:flex-row @xs:items-end">
      <UFormField :label="t('fields.laneGap')" name="laneGap" class="min-w-0 flex-1">
        <template #hint>
          <HelpHint :text="t('pages.editor.help.laneGap')" />
        </template>
        <UInputNumber
          :model-value="preset.laneGap"
          v-bind="dirtyForm.fieldProps('laneGap')"
          :min="LANE_GAP_MIN"
          :max="LANE_GAP_MAX"
          class="w-full"
          @update:model-value="onLaneGap"
        />
      </UFormField>
      <UTooltip
        :text="t('pages.editor.alignLanes')"
        :kbds="keyLabels(shortcutKeys(PRESET_EDITOR_SHORTCUTS.align))"
      >
        <UButton
          :label="t('pages.editor.alignLanes')"
          icon="i-lucide-align-vertical-space-around"
          class="w-full justify-center @xs:w-auto"
          @click="presetForm.align"
        />
      </UTooltip>
    </div>

    <EditorItemList
      :items="items"
      :selected="selection.selected.value"
      @select="onSelect"
      @toggle-lock="onToggleLock"
      @toggle-hidden="visibility.toggle"
    />

    <USeparator />

    <LaneCornerFields v-if="selection.active.value" />
    <template v-else-if="selection.selected.value.size > 1">
      <p class="text-muted text-sm">
        {{ t('pages.editor.multipleSelected', { n: selection.selected.value.size }) }}
      </p>
      <EditorTransformFields
        v-if="movableBounds"
        :bounds="movableBounds"
        unit="px"
        @change="presetForm.transformSelection"
      />
    </template>
    <p v-else class="text-muted text-sm">{{ t('pages.editor.selectLaneHint') }}</p>
  </EditorPanel>
</template>
