import { nextTick, onBeforeUnmount, type Ref } from 'vue';
import Sortable from 'sortablejs';

/**
 * Lets the items of a list be dragged by their data-drag-handle element between start and stop
 * and reports every move to onMove, which changes the state the list renders.
 */
export function useSortableList(
  list: Readonly<Ref<HTMLElement | null>>,
  onMove: (oldIndex: number, newIndex: number) => void,
) {
  let sortable: Sortable | null = null;

  /** Makes the list sortable once it is rendered. */
  function start(): void {
    void nextTick(() => {
      if (!list.value || sortable) return;
      sortable = new Sortable(list.value, {
        handle: '[data-drag-handle]',
        animation: 150,
        forceFallback: true,
        delay: 150,
        delayOnTouchOnly: true,
        ghostClass: 'opacity-30',
        onEnd: (event) => {
          const { oldIndex, newIndex, item, from } = event;
          if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) return;
          // Sortable moved the element itself. It goes back to its place, so only Vue changes the DOM
          // and renders the new order from the state.
          from.removeChild(item);
          from.insertBefore(item, from.children[oldIndex] ?? null);
          onMove(oldIndex, newIndex);
        },
      });
    });
  }

  /** Removes the drag handling. */
  function stop(): void {
    sortable?.destroy();
    sortable = null;
  }

  onBeforeUnmount(stop);

  return {
    start,
    stop,
  };
}
