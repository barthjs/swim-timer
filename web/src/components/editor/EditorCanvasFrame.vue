<script setup lang="ts">
import { computed } from 'vue';

import EditorLoupe from '@/components/editor/EditorLoupe.vue';
import EditorRuler from '@/components/editor/EditorRuler.vue';
import PointerGuides from '@/components/editor/PointerGuides.vue';
import SelectionMarquee from '@/components/editor/SelectionMarquee.vue';
import SnapGuides from '@/components/editor/SnapGuides.vue';
import type { EditorCanvas } from '@/composables/editor/useEditorCanvas';
import { marqueeBounds } from '@/composables/editor/useMarqueeSelection';
import type { Size } from '@/lib/editor/geometry';
import type { SnapFeedback } from '@/lib/editor/snap';

// The area of an editor with zoom and pan and the aids drawn above the content: the snap guides, the rulers
// with the guides dragged out of them, the pointer guides, the marquee, and the loupe. The default slot draws
// the content, whose root element carries EDITOR_SURFACE_ATTRIBUTE. The loupe slot draws the content again
// in content units while the loupe is shown.
const props = defineProps<{
  canvas: EditorCanvas;
  // The size of the content in its own units.
  content: Size;
  feedback: SnapFeedback;
  // The unit of the content, which the rulers and the distances are labeled in, such as "px" or "mm".
  unit: string;
  showRulers: boolean;
  showGuides: boolean;
  // Whether the loupe is shown, such as while a corner is dragged.
  showLoupe: boolean;
  viewMode: boolean;
}>();

defineSlots<{
  default: () => unknown;
  loupe: (props: { strokeWidth: number }) => unknown;
}>();

const viewport = computed(() => props.canvas.viewport);
const rulersShown = computed(() => props.showRulers && !props.viewMode);

// The guides at their place in the area
const guides = computed(() =>
  props.canvas.guides.lines.value.map((line) => {
    const { x, y, scale } = viewport.value.transform.value;
    return {
      axis: line.axis,
      style:
        line.axis === 'x'
          ? {
              left: `${x + line.value * scale}px`,
            }
          : {
              top: `${y + line.value * scale}px`,
            },
    };
  }),
);
const pointerGuidesShown = computed(
  () => props.showGuides && props.canvas.dragging.value && !props.viewMode,
);
</script>

<template>
  <div
    :ref="canvas.setArea"
    class="bg-muted border-default relative overflow-hidden rounded-md border select-none"
    :class="{
      'cursor-grab': viewport.spacePressed.value && !viewport.panning.value,
      'cursor-grabbing': viewport.panning.value,
    }"
    @pointerdown="canvas.onAreaPointerDown"
  >
    <slot />

    <SnapGuides
      :feedback="feedback"
      :content="content"
      :view="viewport.transform.value"
      :unit="unit"
    />

    <template v-if="rulersShown">
      <!-- Each guide spans the whole area and has a wider strip to grab it by than its line shows. -->
      <div
        v-for="(guide, index) in guides"
        :key="`guide-${index}`"
        class="group absolute z-[4] flex touch-none"
        :class="
          guide.axis === 'x'
            ? 'inset-y-0 w-2 -translate-x-1/2 cursor-col-resize justify-center'
            : 'inset-x-0 h-2 -translate-y-1/2 cursor-row-resize items-center'
        "
        :style="guide.style"
        @pointerdown.left.stop="canvas.startGuideMove($event, index)"
        @dblclick="canvas.guides.removeAt(index)"
      >
        <div
          class="bg-info group-hover:bg-primary"
          :class="guide.axis === 'x' ? 'h-full w-px' : 'h-px w-full'"
        ></div>
      </div>

      <EditorRuler
        orientation="horizontal"
        :length="viewport.areaWidth.value"
        :offset="viewport.pan.x"
        :scale="viewport.scale.value"
        :pointer="canvas.dragging.value ? canvas.areaPointer.value.x : null"
        @guide-down="(event) => canvas.startGuideFromRuler(event, 'horizontal')"
      />
      <EditorRuler
        orientation="vertical"
        :length="viewport.areaHeight.value"
        :offset="viewport.pan.y"
        :scale="viewport.scale.value"
        :pointer="canvas.dragging.value ? canvas.areaPointer.value.y : null"
        @guide-down="(event) => canvas.startGuideFromRuler(event, 'vertical')"
      />
    </template>

    <PointerGuides
      v-if="pointerGuidesShown"
      :x="canvas.areaPointer.value.x"
      :y="canvas.areaPointer.value.y"
      :content-x="canvas.contentPointer.value.x"
      :content-y="canvas.contentPointer.value.y"
    />

    <SelectionMarquee
      v-if="canvas.marquee.value"
      :bounds="marqueeBounds(canvas.marquee.value)"
      :view="viewport.transform.value"
    />

    <EditorLoupe
      v-if="showLoupe"
      v-slot="{ strokeWidth }"
      :pointer="canvas.areaPointer.value"
      :content-pointer="canvas.contentPointer.value"
      :area="{
        width: viewport.areaWidth.value,
        height: viewport.areaHeight.value,
      }"
      :content="content"
      :scale="viewport.scale.value"
    >
      <slot name="loupe" :stroke-width="strokeWidth" />
    </EditorLoupe>
  </div>
</template>
