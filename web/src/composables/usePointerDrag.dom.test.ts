import { afterEach, describe, expect, it, vi } from 'vitest';

import { withSetup } from '@/test/withSetup';

import { usePointerDrag, type PointerDragHandlers } from '@/composables/usePointerDrag';

/** pointer creates a pointer event of the given type for one pointer. */
function pointer(type: string, init: PointerEventInit = {}): PointerEvent {
  return new PointerEvent(type, {
    pointerId: 1,
    ...init,
  });
}

/**
 * press dispatches a pointerdown on a fresh element and returns that event and the element,
 * so the drag starts with a real currentTarget for pointer capture.
 */
function press(start: (event: PointerEvent) => void, clientX: number, clientY: number) {
  const element = document.createElement('div');
  document.body.append(element);
  const setPointerCapture = vi.spyOn(element, 'setPointerCapture').mockImplementation(() => {});
  const hasPointerCapture = vi.spyOn(element, 'hasPointerCapture').mockReturnValue(true);
  const releasePointerCapture = vi
    .spyOn(element, 'releasePointerCapture')
    .mockImplementation(() => {});

  element.addEventListener('pointerdown', (event) => start(event as PointerEvent));
  element.dispatchEvent(
    pointer('pointerdown', {
      clientX,
      clientY,
    }),
  );

  return {
    element,
    setPointerCapture,
    hasPointerCapture,
    releasePointerCapture,
  };
}

describe('usePointerDrag', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  describe('start', () => {
    it('reports deltas against the start position', () => {
      const onStart = vi.fn<(event: PointerEvent) => void>();
      const onMove = vi.fn<PointerDragHandlers['onMove']>();
      const { result } = withSetup(() =>
        usePointerDrag({
          onStart,
          onMove,
        }),
      );

      const { setPointerCapture } = press(result.start, 100, 50);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 130,
          clientY: 40,
        }),
      );

      expect(onStart).toHaveBeenCalledOnce();
      expect(setPointerCapture).toHaveBeenCalledWith(1);
      expect(onMove).toHaveBeenCalledWith(30, -10, expect.any(PointerEvent));
    });

    it('works without optional handlers', () => {
      const onMove = vi.fn<PointerDragHandlers['onMove']>();
      const { result } = withSetup(() =>
        usePointerDrag({
          onMove,
        }),
      );

      press(result.start, 0, 0);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 5,
          clientY: 5,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(onMove).toHaveBeenCalledWith(5, 5, expect.any(PointerEvent));
    });

    it.each(['pointerup', 'pointercancel'])('ends the gesture on %s', (type) => {
      const onMove = vi.fn<PointerDragHandlers['onMove']>();
      const onEnd = vi.fn<(event: PointerEvent) => void>();
      const { result } = withSetup(() =>
        usePointerDrag({
          onMove,
          onEnd,
        }),
      );

      const { releasePointerCapture } = press(result.start, 0, 0);
      window.dispatchEvent(pointer(type));
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 10,
          clientY: 10,
        }),
      );
      window.dispatchEvent(pointer(type));

      expect(onEnd).toHaveBeenCalledOnce();
      expect(releasePointerCapture).toHaveBeenCalledWith(1);
      expect(onMove).not.toHaveBeenCalled();
    });

    it('does not release a capture the element no longer holds', () => {
      const { result } = withSetup(() =>
        usePointerDrag({
          onMove: vi.fn<PointerDragHandlers['onMove']>(),
        }),
      );
      const { hasPointerCapture, releasePointerCapture } = press(result.start, 0, 0);
      hasPointerCapture.mockReturnValue(false);

      window.dispatchEvent(pointer('pointerup'));

      expect(releasePointerCapture).not.toHaveBeenCalled();
    });

    it('stops listening when the component unmounts', () => {
      const onMove = vi.fn<PointerDragHandlers['onMove']>();
      const { result, unmount } = withSetup(() =>
        usePointerDrag({
          onMove,
        }),
      );

      press(result.start, 0, 0);
      unmount();
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 5,
          clientY: 5,
        }),
      );

      expect(onMove).not.toHaveBeenCalled();
    });
  });
});
