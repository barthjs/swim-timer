import { computed, reactive, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';
import { useEventListener } from '@vueuse/core';

import { useCanvasViewport } from '@/composables/editor/useCanvasViewport';
import type { EditorGuides, GuideView } from '@/composables/editor/useEditorGuides';
import type { EditorSelection } from '@/composables/editor/useEditorSelection';
import { useMarqueeSelection } from '@/composables/editor/useMarqueeSelection';
import type { Bounds } from '@/lib/editor/geometry';

/** EDITOR_SURFACE_ATTRIBUTE marks the element the content is drawn on, where a press starts a marquee. */
export const EDITOR_SURFACE_ATTRIBUTE = 'data-editor-surface';

export type EditorCanvasOptions<K> = {
  // The guides of the document, which the rulers add and the canvas draws.
  guides: EditorGuides;
  // The size of the content in its own units, such as pixels or millimeters.
  contentWidth: MaybeRefOrGetter<number>;
  contentHeight: MaybeRefOrGetter<number>;
  // The free space around the fitted content, which keeps it clear of the rulers.
  padding: number;
  // The screen pixels one content unit spans at 100 percent, such as 1 for pixels or about 3.78 for millimeters.
  actualScale: number;
  minZoom?: number;
  maxZoom?: number;
  selection: EditorSelection<K>;
  // The items the marquee can select, with their bounds in content units.
  items: () => Iterable<[K, Bounds]>;
  // Whether a gesture on an item runs, such as a drag, which the rulers and the guides follow.
  gesture: () => boolean;
  // The view mode blocks editing, so a press starts no marquee.
  viewMode: MaybeRefOrGetter<boolean>;
};

export type EditorCanvas = ReturnType<typeof useEditorCanvas>;

/**
 * useEditorCanvas holds the state of the canvas of an editor: the area with zoom and pan, the position of the pointer,
 * the dragging of guides, and the marquee selection. EditorCanvasFrame draws it, and the editor draws its content
 * inside.
 */
export function useEditorCanvas<K>(options: EditorCanvasOptions<K>) {
  const area = shallowRef<HTMLElement | null>(null);
  const viewport = useCanvasViewport({
    area,
    contentWidth: options.contentWidth,
    contentHeight: options.contentHeight,
    padding: options.padding,
    minZoom: options.minZoom,
    maxZoom: options.maxZoom,
  });

  const { guides } = options;

  const { marquee, startMarquee } = useMarqueeSelection({
    toContent: viewport.clientToContent,
    items: options.items,
    selection: options.selection,
  });

  // The pointer position in the window. Pointer events are tracked instead of mouse events,
  // since a drag prevents the compatibility mouse events and holds the pointer capture.
  const pointer = reactive({
    x: 0,
    y: 0,
  });
  useEventListener(window, ['pointermove', 'pointerdown'], (event: PointerEvent) => {
    pointer.x = event.clientX;
    pointer.y = event.clientY;
  });
  const areaPointer = computed(() => viewport.clientToArea(pointer.x, pointer.y));
  const contentPointer = computed(() => viewport.clientToContent(pointer.x, pointer.y));

  const dragging = computed(
    () => options.gesture() || !!marquee.value || guides.dragged.value !== null,
  );

  /** onAreaPointerDown starts a marquee selection on the empty content or the area around it. */
  function onAreaPointerDown(event: PointerEvent): void {
    if (toValue(options.viewMode)) return;
    const target = event.target;
    if (
      target !== area.value &&
      !(target instanceof Element && target.hasAttribute(EDITOR_SURFACE_ATTRIBUTE))
    ) {
      return;
    }
    startMarquee(event);
  }

  /** guideView returns what dragging a guide needs from the canvas. */
  function guideView(): GuideView {
    return {
      content: {
        width: toValue(options.contentWidth),
        height: toValue(options.contentHeight),
      },
      toArea: viewport.clientToArea,
      toContent: viewport.clientToContent,
    };
  }

  /** startGuideFromRuler adds a guide at the pointer on a ruler and lets it follow the pointer. */
  function startGuideFromRuler(event: PointerEvent, ruler: 'horizontal' | 'vertical'): void {
    guides.startFromRuler(event, ruler, guideView());
  }

  /** startGuideMove lets a guide follow the pointer. */
  function startGuideMove(event: PointerEvent, index: number): void {
    guides.startMove(event, index, guideView());
  }

  /** setArea takes the element of the area, which the frame hands over once it is mounted. */
  function setArea(element: unknown): void {
    area.value = element instanceof HTMLElement ? element : null;
  }

  // The zoom relative to the actual size of the content, which the toolbar shows in percent
  const actualZoom = computed(() => viewport.scale.value / options.actualScale);

  /** zoomToActual zooms to a share of the actual size, such as 1 for 100 percent. */
  function zoomToActual(factor: number): void {
    viewport.zoomToScale(options.actualScale * factor);
  }

  /** zoomToBounds fills the area with a part of the content, if there is one. */
  function zoomToBounds(bounds: Bounds | null): void {
    if (bounds) viewport.zoomToBounds(bounds);
  }

  return {
    area,
    viewport,
    guides,
    marquee,
    areaPointer,
    contentPointer,
    dragging,
    actualZoom,
    zoomToActual,
    onAreaPointerDown,
    startGuideFromRuler,
    startGuideMove,
    setArea,
    zoomToBounds,
  };
}
