import { nextTick, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SortableEvent, SortableOptions } from 'sortablejs';

import { useSortableList } from '@/composables/ui/useSortableList';
import { withSetup } from '@/test/withSetup';

const { created } = vi.hoisted(() => ({
  created: [] as { element: HTMLElement; options: SortableOptions; destroy: () => void }[],
}));

vi.mock('sortablejs', () => ({
  default: class {
    destroy = vi.fn<() => void>();

    constructor(element: HTMLElement, options: SortableOptions) {
      created.push({
        element,
        options,
        destroy: this.destroy,
      });
    }
  },
}));

/** Returns a list element with the given number of items. */
function listOf(count: number): HTMLElement {
  const list = document.createElement('div');
  for (let i = 0; i < count; i++) {
    const item = document.createElement('div');
    item.dataset.index = String(i);
    list.append(item);
  }
  return list;
}

/** Reports to Sortable that the item at oldIndex was dropped at newIndex after moving it there. */
function drop(list: HTMLElement, oldIndex: number, newIndex: number): void {
  const item = list.children[oldIndex] as HTMLElement;
  list.removeChild(item);
  list.insertBefore(item, list.children[newIndex] ?? null);
  created.at(-1)?.options.onEnd?.({
    oldIndex,
    newIndex,
    item,
    from: list,
  } as unknown as SortableEvent);
}

let unmount = () => {};

/** Builds the sortable list in a mounted component. */
function setup(
  list: HTMLElement | null,
  onMove = vi.fn<(oldIndex: number, newIndex: number) => void>(),
) {
  const mounted = withSetup(() => useSortableList(ref(list), onMove));
  // A test may unmount the component itself, which afterEach must not repeat
  unmount = () => {
    mounted.unmount();
    unmount = () => {};
  };
  return {
    ...mounted.result,
    onMove,
  };
}

describe('useSortableList', () => {
  beforeEach(() => {
    created.length = 0;
  });

  afterEach(() => {
    unmount();
  });

  describe('start', () => {
    it('makes the list sortable by its handles once it is rendered', async () => {
      const list = listOf(3);
      const sortable = setup(list);

      sortable.start();
      await nextTick();

      expect(created).toHaveLength(1);
      expect(created[0]?.element).toBe(list);
      expect(created[0]?.options.handle).toBe('[data-drag-handle]');
    });

    it('reports a move and puts the dragged element back for Vue to move it', async () => {
      const list = listOf(3);
      const sortable = setup(list);
      sortable.start();
      await nextTick();

      drop(list, 0, 2);

      expect(sortable.onMove).toHaveBeenCalledWith(0, 2);
      expect([...list.children].map((item) => (item as HTMLElement).dataset.index)).toEqual([
        '0',
        '1',
        '2',
      ]);
    });

    it('ignores a drop at the same place', async () => {
      const list = listOf(3);
      const sortable = setup(list);
      sortable.start();
      await nextTick();

      drop(list, 1, 1);

      expect(sortable.onMove).not.toHaveBeenCalled();
    });

    it('sets up the list only once', async () => {
      const sortable = setup(listOf(2));
      sortable.start();
      await nextTick();

      sortable.start();
      await nextTick();

      expect(created).toHaveLength(1);
    });

    it('does nothing without a rendered list', async () => {
      const sortable = setup(null);

      sortable.start();
      await nextTick();

      expect(created).toHaveLength(0);
    });
  });

  describe('stop', () => {
    it('removes the drag handling', async () => {
      const sortable = setup(listOf(2));
      sortable.start();
      await nextTick();

      sortable.stop();

      expect(created[0]?.destroy).toHaveBeenCalledOnce();
    });
  });

  it('stops with the component', async () => {
    const sortable = setup(listOf(2));
    sortable.start();
    await nextTick();

    unmount();

    expect(created[0]?.destroy).toHaveBeenCalledOnce();
  });
});
