import { onUnmounted } from 'vue';
import { useEventListener } from '@vueuse/core';

export type PointerDragHandlers = {
  onStart?: (event: PointerEvent) => void;
  onMove: (deltaX: number, deltaY: number, event: PointerEvent) => void;
  onEnd?: (event: PointerEvent) => void;
};

/** Reports the movement of the captured pointer in client pixels relative to where start was called. */
export function usePointerDrag(handlers: PointerDragHandlers) {
  let active = false;
  let startClientX = 0;
  let startClientY = 0;
  let pointerId: number | null = null;
  let captureTarget: Element | null = null;
  // Removes the window listeners of the running gesture. They are only registered while dragging.
  let removeListeners: (() => void) | null = null;

  /** Reports the distance from the start of the gesture. */
  function onPointerMove(event: PointerEvent): void {
    if (!active) return;
    handlers.onMove(event.clientX - startClientX, event.clientY - startClientY, event);
  }

  /** Removes the window listeners of the running gesture. */
  function stopListening(): void {
    removeListeners?.();
    removeListeners = null;
  }

  /** Ends the gesture and releases the pointer. */
  function stop(event: PointerEvent): void {
    if (!active) return;
    active = false;
    if (captureTarget && pointerId !== null && captureTarget.hasPointerCapture(pointerId)) {
      captureTarget.releasePointerCapture(pointerId);
    }
    stopListening();
    handlers.onEnd?.(event);
  }

  /** Begins a gesture at the position of the event. */
  function start(event: PointerEvent): void {
    active = true;
    startClientX = event.clientX;
    startClientY = event.clientY;
    pointerId = event.pointerId;
    captureTarget = event.currentTarget instanceof Element ? event.currentTarget : null;
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
