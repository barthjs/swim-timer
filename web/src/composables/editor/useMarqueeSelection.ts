import { ref } from 'vue';

import { usePointerDrag } from '@/composables/usePointerDrag';
import { isAdditiveSelect, type EditorSelection } from '@/composables/editor/useEditorSelection';
import { boundsIntersect, type Bounds, type Point } from '@/lib/editor/geometry';

// The value of PointerEvent.button for the primary button.
const BUTTON_PRIMARY = 0;

export type MarqueeSelectionOptions<K> = {
  // Converts a position in the browser window into content units.
  toContent: (clientX: number, clientY: number) => Point;
  // The items the marquee can select, with their bounds in content units.
  items: () => Iterable<[K, Bounds]>;
  selection: EditorSelection<K>;
};

export type Marquee = {
  start: Point;
  current: Point;
};

/** marqueeBounds returns the area a marquee spans, independent of the direction it was drawn in. */
export function marqueeBounds(marquee: Marquee): Bounds {
  return {
    minX: Math.min(marquee.start.x, marquee.current.x),
    maxX: Math.max(marquee.start.x, marquee.current.x),
    minY: Math.min(marquee.start.y, marquee.current.y),
    maxY: Math.max(marquee.start.y, marquee.current.y),
  };
}

/**
 * useMarqueeSelection selects items by drawing a rectangle on the empty canvas, in content units.
 * Every item whose bounds the rectangle touches is selected. With Shift, Ctrl, or Cmd held at the start,
 * the previous selection is kept and extended.
 */
export function useMarqueeSelection<K>(options: MarqueeSelectionOptions<K>) {
  const marquee = ref<Marquee | null>(null);
  let base = new Set<K>();

  const pointerDrag = usePointerDrag({
    onMove: (_dx, _dy, event) => {
      if (!marquee.value) return;
      marquee.value.current = options.toContent(event.clientX, event.clientY);
      select(marquee.value);
    },
    onEnd: () => {
      marquee.value = null;
    },
  });

  /** select selects the items inside the marquee in addition to the selection it started with. */
  function select(area: Marquee): void {
    const bounds = marqueeBounds(area);
    const next = new Set(base);
    for (const [key, itemBounds] of options.items()) {
      if (boundsIntersect(itemBounds, bounds)) next.add(key);
    }
    options.selection.set(next);
  }

  /** startMarquee begins a marquee selection with the primary pointer button. */
  function startMarquee(event: PointerEvent): void {
    if (event.button !== BUTTON_PRIMARY) return;

    const position = options.toContent(event.clientX, event.clientY);
    marquee.value = {
      start: position,
      current: position,
    };
    base = isAdditiveSelect(event) ? new Set(options.selection.selected.value) : new Set();
    options.selection.set(base);
    pointerDrag.start(event);
  }

  return {
    marquee,
    startMarquee,
  };
}
