<script setup lang="ts">
import { computed } from 'vue';

import EditorCanvasFrame from '@/components/editor/EditorCanvasFrame.vue';
import LaneCanvas from '@/components/preset-editor/LaneCanvas.vue';
import LanePreview from '@/components/preset-editor/LanePreview.vue';
import { EDITOR_SURFACE_ATTRIBUTE, useEditorCanvas } from '@/composables/editor/useEditorCanvas';
import type { EditorViewOptions } from '@/composables/editor/useEditorWorkspace';
import { useLaneDrag } from '@/composables/useLaneDrag';
import { usePresetForm } from '@/composables/usePresetForm';
import { frameOf, laneBounds, liveFirstLane, polygonPoints } from '@/lib/lanes';
import { useLiveStore } from '@/stores/live';

// The free space around the fitted canvas, which keeps it clear of the rulers.
const CANVAS_PADDING = 32;

// The reference resolution of the open preset as the editing canvas, with zoom and pan.
const props = defineProps<{
  view: EditorViewOptions;
  // Shows the lanes as the AR overlay draws them, without the view options, and blocks editing.
  viewMode: boolean;
}>();

const presetForm = usePresetForm();
const { preset, editor, loaded } = presetForm;
const { selection, visibility, backgroundImage } = editor;

// The lanes that are drawn, without the hidden ones
const visibleLanes = computed(() =>
  Object.fromEntries(Object.entries(preset.lanes).filter(([key]) => !visibility.isHidden(key))),
);

// The live heat tells from which number the pool counts its lanes,
// so the labels and the preview show the lanes of the heat as the AR overlay maps them.
const liveStore = useLiveStore();
const firstLane = computed(() => liveFirstLane(liveStore.state));

const canvas = useEditorCanvas({
  guides: editor.guides,
  contentWidth: () => preset.frameWidth,
  contentHeight: () => preset.frameHeight,
  padding: CANVAS_PADDING,
  // One pixel of the reference resolution is one pixel of the screen at 100 percent
  actualScale: 1,
  selection,
  // Locked and hidden lanes keep their place, so the marquee leaves them out
  items: () =>
    Object.entries(visibleLanes.value)
      .filter(([, lane]) => !lane.locked)
      .map(([key, lane]) => [key, laneBounds(lane)]),
  gesture: () => editor.dragging.value,
  viewMode: () => props.viewMode,
});
const { viewport } = canvas;

const laneDrag = useLaneDrag(
  {
    snapping: () => props.view.snapping.value,
    toFrame: viewport.clientToContent,
    scale: viewport.scale,
    // The guides only snap while the rulers show them
    guides: () =>
      props.view.rulers.value
        ? editor.guides.targets()
        : {
            x: [],
            y: [],
          },
  },
  presetForm,
);

const gridShown = computed(() => props.view.grid.value && !props.viewMode);
const previewShown = computed(() => props.view.preview.value || props.viewMode);

// --editor-px is one screen pixel in pixels of the reference resolution,
// so outlines, labels, and handles keep their size at every zoom.
const canvasStyle = computed(() => ({
  ...viewport.contentStyle.value,
  '--editor-px': `${1 / viewport.scale.value}px`,
}));

/** zoomToSelection fills the canvas with the selected lanes. */
function zoomToSelection(): void {
  canvas.zoomToBounds(presetForm.selectionBounds.value);
}

defineExpose({
  actualZoom: canvas.actualZoom,
  dragging: canvas.dragging,
  zoomBy: viewport.zoomBy,
  resetZoom: viewport.resetZoom,
  zoomToActual: canvas.zoomToActual,
  zoomToSelection,
});
</script>

<template>
  <!--
    From lg up, the canvas takes the space the flex layout gives it. On narrow screens
    it has the aspect ratio of the reference resolution, so no empty space remains around the image.
  -->
  <EditorCanvasFrame
    :canvas="canvas"
    :content="frameOf(preset)"
    :feedback="laneDrag.feedback.value"
    unit="px"
    :show-rulers="view.rulers.value"
    :show-guides="view.guides.value"
    :show-loupe="view.loupe.value && laneDrag.mode.value === 'corner'"
    :view-mode="viewMode"
    class="aspect-(--frame-aspect) max-h-[80vh] min-h-48 lg:aspect-auto lg:max-h-none lg:min-h-0 lg:flex-1"
    :style="{
      '--frame-aspect': `${preset.frameWidth} / ${preset.frameHeight}`,
    }"
  >
    <div
      v-bind="{ [EDITOR_SURFACE_ATTRIBUTE]: '' }"
      class="bg-default absolute top-0 left-0 shadow-lg"
      :class="{ 'editor-grid': gridShown }"
      :style="canvasStyle"
    >
      <img
        v-if="backgroundImage.url.value"
        :src="backgroundImage.url.value"
        alt=""
        class="pointer-events-none absolute inset-0 size-full object-fill"
        :style="{ opacity: backgroundImage.opacity.value }"
        @load="backgroundImage.onLoad"
      />

      <LanePreview v-if="loaded && previewShown" :preset="preset" :first-lane="firstLane" />

      <LaneCanvas
        v-if="loaded && !viewMode"
        :lanes="visibleLanes"
        :selected-lanes="selection.selected.value"
        :active-lane-key="selection.active.value"
        :reverse-lanes="preset.reverseLanes"
        :first-lane="firstLane"
        :outline-only="view.preview.value"
        @lane-down="laneDrag.startLaneDrag"
        @corner-down="laneDrag.startCornerDrag"
      />
    </div>

    <template #loupe="{ strokeWidth }">
      <img
        v-if="backgroundImage.url.value"
        :src="backgroundImage.url.value"
        alt=""
        class="absolute inset-0 size-full object-fill"
      />
      <svg class="absolute inset-0 size-full overflow-visible">
        <polygon
          v-for="(lane, key) in preset.lanes"
          :key="key"
          :points="polygonPoints(lane)"
          class="loupe-outline"
          :stroke-width="strokeWidth"
        />
      </svg>
    </template>
  </EditorCanvasFrame>
</template>

<style scoped>
/* The grid spans 50 pixels of the reference resolution with lines of one screen pixel. */
.editor-grid {
  background-size: 50px 50px;
  background-image:
    linear-gradient(
      to right,
      var(--color-editor-grid) var(--editor-px),
      transparent var(--editor-px)
    ),
    linear-gradient(
      to bottom,
      var(--color-editor-grid) var(--editor-px),
      transparent var(--editor-px)
    );
}

.loupe-outline {
  fill: none;
  stroke: var(--color-editor-lane);
}
</style>
