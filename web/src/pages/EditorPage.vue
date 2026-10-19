<script setup lang="ts">
import { onMounted, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave } from 'vue-router';
import type { Form, FormSchema } from '@nuxt/ui';

import EditorSidebar from '@/components/editor/EditorSidebar.vue';
import EditorToolbar from '@/components/editor/EditorToolbar.vue';
import LaneSettings from '@/components/preset-editor/LaneSettings.vue';
import PresetCanvas from '@/components/preset-editor/PresetCanvas.vue';
import PresetHelpModal from '@/components/preset-editor/PresetHelpModal.vue';
import PresetPicker from '@/components/preset-editor/PresetPicker.vue';
import PresetSettings from '@/components/preset-editor/PresetSettings.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useEditorShortcuts } from '@/composables/editor/useEditorShortcuts';
import { useEditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import { useFormValidation } from '@/composables/form/useFormValidation';
import { providePresetForm } from '@/composables/usePresetForm';
import { useSaveShortcut } from '@/composables/form/useSaveShortcut';
import { useUnloadGuard } from '@/composables/useUnloadGuard';
import { shortcutDefinition } from '@/lib/editor/shortcuts';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { PRESET_COMMENT_MAX, PRESET_NAME_MAX, type EditorPreset } from '@/lib/lanes';
import { PRESET_EDITOR_SHORTCUTS } from '@/lib/presetEditorShortcuts';
import { useLiveStore } from '@/stores/live';

// How far the arrow keys move the selected lanes, in pixels of the reference resolution.
const NUDGE_STEP = 1;
const NUDGE_STEP_LARGE = 10;

// The zoom factor of one click on the zoom buttons.
const ZOOM_STEP = 1.25;

const { t } = useI18n();

const form = useTemplateRef<Form<FormSchema>>('form');
const canvas = useTemplateRef<InstanceType<typeof PresetCanvas>>('canvas');
const presetForm = providePresetForm(form);
const { editor } = presetForm;
const { selection, history, backgroundImage } = editor;

// The view mode shows the background image with the lanes as the AR overlay draws them.
// A drag or a marquee keeps the keys from changing the lanes under the pointer.
const workspace = useEditorWorkspace({
  storageKey: 'editor',
  onEnterViewMode: () => selection.clear(),
  onEscape: () => selection.clear(),
  onTab: (step) => selection.selectNext(step),
  busy: () => canvas.value?.dragging ?? false,
});
const { sidebarOpen, viewMode, helpOpen, fullscreen, whenEditing } = workspace;

// The hidden file input that picks the background image of the preset.
const imageInput = useTemplateRef<HTMLInputElement>('imageInput');

// The rules repeat the required fields and maximum lengths the server checks for a preset.
const { validate } = useFormValidation<EditorPreset>(() => ({
  name: {
    required: 'required',
    max: PRESET_NAME_MAX,
  },
  frameWidth: {
    required: 'required',
  },
  frameHeight: {
    required: 'required',
  },
  textSize: {
    required: 'required',
  },
  textAnchor: {
    required: 'required',
  },
  comment: {
    max: PRESET_COMMENT_MAX,
  },
}));

// Another preset starts fitted, since the zoom and the pan of the last one rarely suit it
watch(
  () => presetForm.openCount.value,
  () => canvas.value?.resetZoom(),
);

/** onImageChange shows the chosen file as the background image and keeps it for the preset. */
function onImageChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] ?? null;
  // Reset the input so choosing the same file again triggers another change
  input.value = '';
  if (file) void presetForm.setBackgroundImage(file);
}

const liveStore = useLiveStore();

onMounted(async () => {
  liveStore.connect();
  await presetForm.loadPresetNames();

  // Open the first preset right away, so the editor is not empty when presets exist
  const first = presetForm.presetNames.value[0];
  if (first && !presetForm.loaded.value) {
    await presetForm.selectPreset(first);
  }
});

// The digit keys select a lane, 0 selects lane 10.
const laneShortcuts = Object.fromEntries(
  ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0'].map((digit) => [
    digit,
    whenEditing(() => selection.selectOnly(digit === '0' ? '10' : digit)),
  ]),
);

useEditorShortcuts({
  workspace,
  history,
  canvas: () => canvas.value,
  selectAll: () => {
    if (presetForm.loaded.value) selection.selectAll();
  },
  pickImage: () => imageInput.value?.click(),
  nudge: presetForm.nudge,
  nudgeStep: NUDGE_STEP,
  nudgeStepLarge: NUDGE_STEP_LARGE,
  extra: {
    [shortcutDefinition(PRESET_EDITOR_SHORTCUTS.lock)]: whenEditing(() => presetForm.toggleLock()),
    [shortcutDefinition(PRESET_EDITOR_SHORTCUTS.align)]: whenEditing(() => presetForm.align()),
    ...laneShortcuts,
  },
});

// Ctrl+S also saves while a field of the cards has the focus.
const { kbds: saveKbds } = useSaveShortcut(form);

useUnloadGuard(() => presetForm.isDirty.value || presetForm.isDraft.value);

onBeforeRouteLeave(() => presetForm.confirmDiscard());
</script>

<template>
  <!--
    From lg up, the admin layout gives the page the height between header and footer, so the canvas size follows
    the layout and not the scroll position. Below lg, the canvas and the cards stack and the page scrolls.
  -->
  <div class="flex flex-col gap-3 lg:min-h-0 lg:flex-1">
    <input ref="imageInput" type="file" accept="image/*" class="hidden" @change="onImageChange" />

    <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
      <div class="flex items-center gap-1">
        <PresetPicker />
        <DocsLink page="editor" />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <UButton
          v-if="presetForm.isDirty.value"
          :label="t('common.reset')"
          icon="i-lucide-rotate-ccw"
          color="neutral"
          variant="outline"
          @click="presetForm.resetPreset"
        />
        <UTooltip :text="t('common.save')" :kbds="saveKbds">
          <UButton
            type="submit"
            form="preset-form"
            :label="t('common.save')"
            icon="i-lucide-save"
            :loading="presetForm.saving.value"
            :disabled="!presetForm.loaded.value"
          />
        </UTooltip>
      </div>
    </div>

    <!-- In fullscreen, the workspace covers the page, so the admin header and the navigation are out of the way. -->
    <div
      class="bg-default flex flex-col gap-3 lg:min-h-0 lg:flex-1 lg:flex-row"
      :class="{
        'fixed inset-0 z-50 overflow-y-auto p-3 lg:overflow-hidden': fullscreen.isFullscreen.value,
      }"
    >
      <div class="flex min-w-0 flex-col gap-2 lg:min-h-0 lg:flex-1">
        <EditorToolbar
          :workspace="workspace"
          :history="history"
          :zoom="canvas?.actualZoom ?? 1"
          :has-selection="selection.selected.value.size > 0"
          :align-count="presetForm.movableLanes.value.length"
          :image="backgroundImage"
          :image-label="t('pages.editor.backgroundImage')"
          :guides="editor.guides"
          @zoom-in="canvas?.zoomBy(ZOOM_STEP)"
          @zoom-out="canvas?.zoomBy(1 / ZOOM_STEP)"
          @reset-zoom="canvas?.resetZoom()"
          @zoom-to-selection="canvas?.zoomToSelection()"
          @zoom-to="(factor) => canvas?.zoomToActual(factor)"
          @align="presetForm.alignSelection"
          @distribute="presetForm.distributeSelection"
          @pick-image="imageInput?.click()"
          @clear-image="void presetForm.setBackgroundImage(null)"
        />

        <UAlert
          v-if="backgroundImage.url.value && backgroundImage.mismatch.value"
          :title="t('pages.editor.imageMismatchTitle')"
          :description="t('pages.editor.imageMismatch')"
          icon="i-lucide-triangle-alert"
          color="warning"
          variant="subtle"
          :ui="{ root: 'px-3 py-2', title: 'text-sm', description: 'text-xs' }"
        />

        <PresetCanvas ref="canvas" :view="workspace.view" :view-mode="viewMode" />
      </div>

      <!-- The cards stay mounted while hidden, since the save button submits their form. -->
      <EditorSidebar v-show="sidebarOpen" storage-key="editor">
        <UForm
          :validate-on="FORM_VALIDATE_ON"
          v-if="presetForm.loaded.value"
          id="preset-form"
          ref="form"
          :state="presetForm.preset"
          :validate="validate"
          class="flex flex-col gap-3 lg:min-h-full"
          @submit="void presetForm.savePreset()"
        >
          <PresetSettings class="shrink-0" />
          <LaneSettings class="shrink-0" />
        </UForm>
        <UCard v-else :ui="{ root: 'bg-elevated' }">
          <p class="text-muted text-sm">{{ t('pages.editor.emptyState') }}</p>
        </UCard>
      </EditorSidebar>
    </div>

    <PresetHelpModal v-model:open="helpOpen" />
  </div>
</template>
