/**
 * pointer creates a pointer event as the browser dispatches it, for the primary button and pointer.
 */
export function pointer(type: string, init: PointerEventInit = {}): PointerEvent {
  return new PointerEvent(type, {
    button: 0,
    pointerId: 1,
    bubbles: true,
    ...init,
  });
}
