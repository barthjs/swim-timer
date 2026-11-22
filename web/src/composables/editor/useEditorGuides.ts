import { ref, toRaw, watch } from 'vue';

import { usePointerDrag } from '@/composables/editor/usePointerDrag';
import type { Point, Size } from '@/lib/editor/geometry';
import { RULER_THICKNESS } from '@/lib/editor/ruler';
import type { AxisTargets, SnapLine } from '@/lib/editor/snap';
import { isRecord } from '@/lib/types/guards';

export type EditorGuidesOptions = {
  // The prefix of the keys the browser keeps the guides under, such as "editor"
  storageKey: string;
  // The document the guides belong to, or null for one that is not saved yet and keeps them in memory
  document: () => string | null;
};

/** What dragging a guide needs from the canvas. */
export type GuideView = {
  content: Size;
  // Converts a position in the browser window into a position in the area and into content units
  toArea: (clientX: number, clientY: number) => Point;
  toContent: (clientX: number, clientY: number) => Point;
};

export type EditorGuides = ReturnType<typeof useEditorGuides>;

/**
 * Holds the guides a user drags from the rulers for items to snap to,
 * kept in the browser under the name of a saved document.
 */
export function useEditorGuides(options: EditorGuidesOptions) {
  const lines = ref<SnapLine[]>([]);
  // The guide that follows the pointer, by its index in lines, and the canvas it is dragged on
  const dragged = ref<number | null>(null);
  let view: GuideView | null = null;

  watch(options.document, (name) => load(name), {
    immediate: true,
  });
  watch(lines, () => save(options.document()), {
    deep: true,
  });

  const pointerDrag = usePointerDrag({
    onMove: (_dx, _dy, event) => moveTo(event),
    onEnd: (event) => drop(event),
  });

  /** Returns the key the guides of a document are kept under. */
  function keyOf(name: string): string {
    return `${options.storageKey}.guides.${name}`;
  }

  /** Shows the guides kept for a document, or none for a document that is not saved yet. */
  function load(name: string | null): void {
    lines.value = name === null ? [] : read(keyOf(name));
  }

  /** Keeps the guides of a saved document. */
  function save(name: string | null): void {
    if (name === null) return;
    write(keyOf(name), lines.value);
  }

  /**
   * Keeps the guides under the name a document was saved with.
   * A document saved under a new name passes its earlier one, so no copy stays behind under it.
   */
  function persist(name: string, previous: string | null = null): void {
    write(keyOf(name), lines.value);
    if (previous !== null && previous !== name) forget(keyOf(previous));
  }

  /** Deletes the guides of a deleted document. */
  function remove(name: string): void {
    forget(keyOf(name));
  }

  /** Returns the guides as lines items snap to. */
  function targets(): AxisTargets {
    return {
      x: lines.value.filter((line) => line.axis === 'x').map((line) => line.value),
      y: lines.value.filter((line) => line.axis === 'y').map((line) => line.value),
    };
  }

  /**
   * Adds a guide across a ruler at the pointer and lets it follow the pointer.
   * The horizontal ruler at the top measures x, so its guides are vertical lines at a value of x,
   * and the vertical ruler places horizontal lines at a value of y.
   */
  function startFromRuler(
    event: PointerEvent,
    ruler: 'horizontal' | 'vertical',
    canvas: GuideView,
  ): void {
    const position = canvas.toContent(event.clientX, event.clientY);
    const line: SnapLine =
      ruler === 'horizontal'
        ? {
            axis: 'x',
            value: position.x,
          }
        : {
            axis: 'y',
            value: position.y,
          };
    lines.value = [...lines.value, line];
    start(event, lines.value.length - 1, canvas);
  }

  /** Lets a guide follow the pointer. */
  function startMove(event: PointerEvent, index: number, canvas: GuideView): void {
    start(event, index, canvas);
  }

  /** Begins a gesture on a guide. */
  function start(event: PointerEvent, index: number, canvas: GuideView): void {
    event.preventDefault();
    dragged.value = index;
    view = canvas;
    pointerDrag.start(event);
  }

  /** Puts the dragged guide at the pointer. */
  function moveTo(event: PointerEvent): void {
    const line = dragged.value === null ? undefined : lines.value[dragged.value];
    if (!line || !view) return;
    const position = view.toContent(event.clientX, event.clientY);
    line.value = line.axis === 'x' ? position.x : position.y;
  }

  /** Ends the gesture and removes the guide if it lies on its ruler or outside the content. */
  function drop(event: PointerEvent): void {
    const index = dragged.value;
    const canvas = view;
    dragged.value = null;
    view = null;
    if (index === null || !canvas) return;
    const line = lines.value[index];
    if (!line) return;

    // A vertical guide belongs to the ruler at the top, a horizontal one to the ruler on the left
    const area = canvas.toArea(event.clientX, event.clientY);
    const onRuler = line.axis === 'x' ? area.y < RULER_THICKNESS : area.x < RULER_THICKNESS;
    const length = line.axis === 'x' ? canvas.content.width : canvas.content.height;
    if (onRuler || line.value < 0 || line.value > length) removeAt(index);
  }

  /** Removes a guide by its index. */
  function removeAt(index: number): void {
    lines.value = lines.value.filter((_, position) => position !== index);
  }

  /** Removes every guide of the document. */
  function clear(): void {
    lines.value = [];
  }

  return {
    lines,
    dragged,
    persist,
    remove,
    targets,
    startFromRuler,
    startMove,
    removeAt,
    clear,
  };
}

/** Returns the guides kept under a key, or none if nothing valid is kept there. */
function read(key: string): SnapLine[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(key) ?? '[]');
    if (!Array.isArray(value)) return [];
    return value.filter(isSnapLine);
  } catch {
    return [];
  }
}

/** Keeps guides under a key if the browser allows it, and deletes the key for none. */
function write(key: string, lines: readonly SnapLine[]): void {
  try {
    if (lines.length === 0) {
      localStorage.removeItem(key);
      return;
    }
    localStorage.setItem(key, JSON.stringify(toRaw(lines)));
  } catch {
    // Without storage, the guides only live until the page is left
  }
}

/** Deletes what is kept under a key if the browser allows it. */
function forget(key: string): void {
  try {
    localStorage.removeItem(key);
  } catch {
    // Without storage, there is nothing to delete
  }
}

/** Reports whether a kept value is a guide. */
function isSnapLine(value: unknown): value is SnapLine {
  return (
    isRecord(value) && (value.axis === 'x' || value.axis === 'y') && typeof value.value === 'number'
  );
}
