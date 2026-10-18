import { computed, reactive, ref, toValue, watch, type MaybeRefOrGetter } from 'vue';
import { useElementBounding, useEventListener, type MaybeElementRef } from '@vueuse/core';

import { usePointerDrag } from '@/composables/usePointerDrag';
import type { Bounds, Point } from '@/lib/editor/geometry';

/** The zoom range relative to the fitted size. */
export const DEFAULT_MIN_ZOOM = 0.25;
export const DEFAULT_MAX_ZOOM = 8;

/**
 * How strongly the wheel zooms per pixel of scroll distance. A touchpad pinch reports small steps
 * and zooms at about the speed of the browser's own pinch zoom. One step is limited,
 * so the large steps of a mouse wheel zoom by about a fifth each.
 */
const WHEEL_ZOOM_SPEED = 0.01;
const WHEEL_ZOOM_STEP_MAX = 20;

/** The pixels of one line, for wheel events that scroll by lines instead of pixels. */
const WHEEL_LINE_HEIGHT = 16;

/** The deltaMode values of a WheelEvent. */
const DOM_DELTA_LINE = 1;
const DOM_DELTA_PAGE = 2;

/** The values of PointerEvent.button that pan. */
const BUTTON_PRIMARY = 0;
const BUTTON_MIDDLE = 1;
const BUTTON_SECONDARY = 2;

export type CanvasViewportOptions = {
  // The element the content is fitted into, which receives the wheel events.
  area: MaybeElementRef<HTMLElement | null | undefined>;
  // The size of the content in its own units. Only its aspect ratio decides the fitted size.
  contentWidth: MaybeRefOrGetter<number>;
  contentHeight: MaybeRefOrGetter<number>;
  // The free space kept around the fitted content, in pixels of the screen.
  padding?: MaybeRefOrGetter<number>;
  minZoom?: number;
  maxZoom?: number;
};

/** ViewTransform places content in the area: its top left corner lies at x, y and one unit spans scale pixels. */
export type ViewTransform = {
  x: number;
  y: number;
  scale: number;
};

/**
 * useCanvasViewport fits content of a given aspect ratio into an area, with zoom and pan on top.
 * The content is drawn in its own units and scaled to the screen by contentStyle. Positions in the area
 * are pixels from its top left corner, so the pan is where the top left corner of the content lies in the area.
 * While neither zoom nor pan were changed, the content stays fitted and centered when the area
 * or the content changes size.
 *
 * Dragging with the right or the middle button, or with the left button while Space is held,
 * pans like in image editors and leaves the left button for the content.
 * The wheel and two fingers on a touchpad pan as well, Shift with the wheel pans sideways,
 * and Ctrl or Cmd with the wheel zooms around the pointer, as a pinch on a touchpad does.
 */
export function useCanvasViewport(options: CanvasViewportOptions) {
  const minZoom = options.minZoom ?? DEFAULT_MIN_ZOOM;
  const maxZoom = options.maxZoom ?? DEFAULT_MAX_ZOOM;

  const bounding = useElementBounding(options.area);
  const zoom = ref(1);
  const pan = reactive<Point>({
    x: 0,
    y: 0,
  });
  // Whether the content follows the size of the area, until the user zooms or pans
  const fitted = ref(true);

  // The screen pixels per content unit at zoom 1, which fit the content into the area
  const fitScale = computed(() => {
    const padding = toValue(options.padding) ?? 0;
    const width = toValue(options.contentWidth);
    const height = toValue(options.contentHeight);
    const availableWidth = bounding.width.value - 2 * padding;
    const availableHeight = bounding.height.value - 2 * padding;
    if (width <= 0 || height <= 0 || availableWidth <= 0 || availableHeight <= 0) return 1;
    return Math.min(availableWidth / width, availableHeight / height);
  });

  // The screen pixels per content unit with the zoom applied
  const scale = computed(() => fitScale.value * zoom.value);

  const transform = computed<ViewTransform>(() => ({
    x: pan.x,
    y: pan.y,
    scale: scale.value,
  }));

  const contentStyle = computed(() => ({
    width: `${toValue(options.contentWidth)}px`,
    height: `${toValue(options.contentHeight)}px`,
    transform: `translate(${pan.x}px, ${pan.y}px) scale(${scale.value})`,
    transformOrigin: '0 0',
  }));

  /** clampZoom limits a zoom factor to the supported range. */
  function clampZoom(value: number): number {
    return Math.min(maxZoom, Math.max(minZoom, value));
  }

  /** center places the content in the middle of the area without changing the zoom. */
  function center(): void {
    pan.x = (bounding.width.value - toValue(options.contentWidth) * scale.value) / 2;
    pan.y = (bounding.height.value - toValue(options.contentHeight) * scale.value) / 2;
  }

  // The pan follows a size change at once, so the content never shows up at a stale position
  watch(
    [fitScale, bounding.width, bounding.height],
    () => {
      if (fitted.value) center();
    },
    {
      immediate: true,
      flush: 'sync',
    },
  );

  /** zoomAt zooms by a factor and keeps the point at x, y in the area in place. */
  function zoomAt(x: number, y: number, factor: number): void {
    const next = clampZoom(zoom.value * factor);
    const ratio = next / zoom.value;
    pan.x = x - (x - pan.x) * ratio;
    pan.y = y - (y - pan.y) * ratio;
    zoom.value = next;
    fitted.value = false;
  }

  /** zoomBy zooms by a factor around the middle of the area. */
  function zoomBy(factor: number): void {
    zoomAt(bounding.width.value / 2, bounding.height.value / 2, factor);
  }

  /** zoomToScale zooms around the middle of the area until one content unit spans a number of screen pixels. */
  function zoomToScale(target: number): void {
    if (target <= 0 || scale.value <= 0) return;
    zoomBy(target / scale.value);
  }

  /** resetZoom fits the content into the area again and keeps it fitted. */
  function resetZoom(): void {
    zoom.value = 1;
    fitted.value = true;
    center();
  }

  /**
   * zoomToBounds zooms and pans so that a part of the content fills the area within the padding.
   * Bounds without an extent, such as those of a single point, keep the zoom and are centered.
   */
  function zoomToBounds(bounds: Bounds): void {
    const padding = toValue(options.padding) ?? 0;
    const width = bounds.maxX - bounds.minX;
    const height = bounds.maxY - bounds.minY;
    const availableWidth = bounding.width.value - 2 * padding;
    const availableHeight = bounding.height.value - 2 * padding;
    if (width > 0 && height > 0 && availableWidth > 0 && availableHeight > 0) {
      const target = Math.min(availableWidth / width, availableHeight / height);
      zoom.value = clampZoom(target / fitScale.value);
    }
    pan.x = bounding.width.value / 2 - ((bounds.minX + bounds.maxX) / 2) * scale.value;
    pan.y = bounding.height.value / 2 - ((bounds.minY + bounds.maxY) / 2) * scale.value;
    fitted.value = false;
  }

  /** panBy moves the content by a distance in screen pixels. */
  function panBy(dx: number, dy: number): void {
    pan.x += dx;
    pan.y += dy;
    fitted.value = false;
  }

  // Whether Space is held, which turns the left button into a pan, and whether a pan drag runs
  const spacePressed = ref(false);
  const panning = ref(false);

  let panStart: Point = {
    x: 0,
    y: 0,
  };
  const panDrag = usePointerDrag({
    onStart: () => {
      panStart = {
        x: pan.x,
        y: pan.y,
      };
      panning.value = true;
    },
    onMove: (dx, dy) => {
      pan.x = panStart.x + dx;
      pan.y = panStart.y + dy;
      fitted.value = false;
    },
    onEnd: () => {
      panning.value = false;
    },
  });

  /** startPan moves the content with the pointer until the pointer is released. */
  function startPan(event: PointerEvent): void {
    event.preventDefault();
    panDrag.start(event);
  }

  /** clientToArea converts a position in the browser window into a position in the area. */
  function clientToArea(clientX: number, clientY: number): Point {
    return {
      x: clientX - bounding.left.value,
      y: clientY - bounding.top.value,
    };
  }

  /** clientToContent converts a position in the browser window into content units. */
  function clientToContent(clientX: number, clientY: number): Point {
    const point = clientToArea(clientX, clientY);
    return {
      x: (point.x - pan.x) / scale.value,
      y: (point.y - pan.y) / scale.value,
    };
  }

  /** contentToArea converts a position in content units into a position in the area. */
  function contentToArea(x: number, y: number): Point {
    return {
      x: pan.x + x * scale.value,
      y: pan.y + y * scale.value,
    };
  }

  /** isPanTrigger reports whether a pointer press pans instead of reaching the content. */
  function isPanTrigger(event: PointerEvent): boolean {
    if (event.button === BUTTON_MIDDLE || event.button === BUTTON_SECONDARY) return true;
    return event.button === BUTTON_PRIMARY && spacePressed.value;
  }

  // The capture phase starts the pan before an element of the content can take the press
  useEventListener(
    options.area,
    'pointerdown',
    (event: PointerEvent) => {
      if (!isPanTrigger(event)) return;
      event.stopPropagation();
      startPan(event);
    },
    {
      capture: true,
    },
  );

  // The right button pans, so the context menu of the browser stays closed on the area
  useEventListener(options.area, 'contextmenu', (event: MouseEvent) => {
    event.preventDefault();
  });

  useEventListener(window, 'keydown', (event: KeyboardEvent) => {
    if (event.code !== 'Space' || isInteractiveTarget(event.target)) return;
    event.preventDefault();
    spacePressed.value = true;
  });
  useEventListener(window, 'keyup', (event: KeyboardEvent) => {
    if (event.code === 'Space') spacePressed.value = false;
  });
  // A key released in another window never reaches this one
  useEventListener(window, 'blur', () => {
    spacePressed.value = false;
  });

  /** wheelDelta returns the scroll distance of a wheel event in pixels. */
  function wheelDelta(event: WheelEvent): Point {
    let factor = 1;
    if (event.deltaMode === DOM_DELTA_LINE) factor = WHEEL_LINE_HEIGHT;
    if (event.deltaMode === DOM_DELTA_PAGE) factor = bounding.height.value;
    return {
      x: event.deltaX * factor,
      y: event.deltaY * factor,
    };
  }

  // A pinch on a touchpad arrives as a wheel event with ctrlKey set
  useEventListener(
    options.area,
    'wheel',
    (event: WheelEvent) => {
      event.preventDefault();
      const delta = wheelDelta(event);

      if (event.ctrlKey || event.metaKey) {
        const step = Math.min(WHEEL_ZOOM_STEP_MAX, Math.max(-WHEEL_ZOOM_STEP_MAX, delta.y));
        const point = clientToArea(event.clientX, event.clientY);
        zoomAt(point.x, point.y, Math.exp(-step * WHEEL_ZOOM_SPEED));
        return;
      }

      // Some browsers turn Shift with the wheel into a sideways scroll themselves, others do not
      if (event.shiftKey && delta.x === 0) {
        panBy(-delta.y, 0);
        return;
      }
      panBy(-delta.x, -delta.y);
    },
    {
      passive: false,
    },
  );

  return {
    areaWidth: bounding.width,
    areaHeight: bounding.height,
    zoom,
    pan,
    spacePressed,
    panning,
    fitScale,
    scale,
    transform,
    contentStyle,
    center,
    zoomAt,
    zoomBy,
    zoomToScale,
    resetZoom,
    zoomToBounds,
    panBy,
    startPan,
    clientToArea,
    clientToContent,
    contentToArea,
  };
}

/**
 * isInteractiveTarget reports whether a key event belongs to a control that uses the Space key itself,
 * such as a text field or a button.
 */
function isInteractiveTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  return (
    target.closest(
      'input, textarea, select, button, a[href], [role="button"], [role="checkbox"], ' +
        '[role="switch"], [role="combobox"], [role="menuitem"], [role="option"], [role="tab"]',
    ) !== null
  );
}
