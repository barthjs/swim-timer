import { onUnmounted } from 'vue';
import { useEventListener } from '@vueuse/core';

export type PointerDragHandlers = {
  onStart?: (event: PointerEvent) => void;
  onMove: (deltaX: number, deltaY: number, event: PointerEvent) => void;
  onEnd?: (event: PointerEvent) => void;
};

/**
 * usePointerDrag reports pointer movement in client pixels relative to where start was called,
 * for moving, resizing, and panning. It captures the pointer, so the gesture keeps receiving events
 * when the pointer leaves the source element. Callers convert the deltas into their own coordinate space.
 */
export function usePointerDrag(handlers: PointerDragHandlers) {
  let active = false;
  let startClientX = 0;
  let startClientY = 0;
  let pointerId: number | null = null;
  let captureTarget: HTMLElement | null = null;
  // Removes the window listeners of the running gesture. They are only registered while dragging
  let removeListeners: (() => void) | null = null;

  /** onPointerMove reports the distance from the start of the gesture. */
  function onPointerMove(event: PointerEvent): void {
    if (!active) return;
    handlers.onMove(event.clientX - startClientX, event.clientY - startClientY, event);
  }

  /** stopListening removes the window listeners of the running gesture. */
  function stopListening(): void {
    removeListeners?.();
    removeListeners = null;
  }

  /** stop ends the gesture and releases the pointer. */
  function stop(event: PointerEvent): void {
    if (!active) return;
    active = false;
    if (captureTarget && pointerId !== null && captureTarget.hasPointerCapture(pointerId)) {
      captureTarget.releasePointerCapture(pointerId);
    }
    stopListening();
    handlers.onEnd?.(event);
  }

  /** start begins a gesture at the position of the event. */
  function start(event: PointerEvent): void {
    active = true;
    startClientX = event.clientX;
    startClientY = event.clientY;
    pointerId = event.pointerId;
    captureTarget = event.currentTarget as HTMLElement | null;
    captureTarget?.setPointerCapture(pointerId);

    stopListening();
    const removeMove = useEventListener(window, 'pointermove', onPointerMove);
    const removeEnd = useEventListener(window, ['pointerup', 'pointercancel'], stop);
    removeListeners = () => {
      removeMove();
      removeEnd();
    };

    handlers.onStart?.(event);
  }

  onUnmounted(stopListening);

  return {
    start,
  };
}
