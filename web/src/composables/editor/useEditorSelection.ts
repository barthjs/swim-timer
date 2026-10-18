import { computed, ref, type Ref } from 'vue';

export type EditorSelection<K> = ReturnType<typeof useEditorSelection<K>>;

/**
 * isAdditiveSelect reports whether a press adds to the selection or removes from it instead of replacing it,
 * which Shift, Ctrl, and Cmd do.
 */
export function isAdditiveSelect(
  event: Pick<MouseEvent, 'shiftKey' | 'ctrlKey' | 'metaKey'>,
): boolean {
  return event.shiftKey || event.ctrlKey || event.metaKey;
}

/**
 * useEditorSelection holds the selected items of an editor by their keys. The keys come in the order
 * the items are stepped through, and the single selected item is the active one whose details are shown.
 * The set is always replaced instead of changed in place, so every change triggers reactivity.
 */
export function useEditorSelection<K>(keys: () => readonly K[]) {
  const selected: Ref<Set<K>> = ref(new Set<K>()) as Ref<Set<K>>;

  // The item whose details are shown, which requires exactly one selected item
  const active = computed<K | null>(() => {
    if (selected.value.size !== 1) return null;
    const [key] = selected.value;
    return key ?? null;
  });

  /** has reports whether an item is selected. */
  function has(key: K): boolean {
    return selected.value.has(key);
  }

  /** set replaces the selection with the given items. */
  function set(next: Iterable<K>): void {
    selected.value = new Set(next);
  }

  /** clear selects no item. */
  function clear(): void {
    selected.value = new Set();
  }

  /** selectOnly replaces the selection with one item, if it exists. */
  function selectOnly(key: K): void {
    if (!keys().includes(key)) return;
    selected.value = new Set([key]);
  }

  /** toggle adds an item to the selection or removes it. */
  function toggle(key: K): void {
    const next = new Set(selected.value);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    selected.value = next;
  }

  /** selectAll selects every item. */
  function selectAll(): void {
    selected.value = new Set(keys());
  }

  /**
   * press selects an item the pointer pressed. An additive press toggles it, a plain press on an unselected item
   * selects only that one, and a plain press on a selected item keeps the selection, so a group moves together.
   */
  function press(key: K, additive: boolean): void {
    if (additive) {
      toggle(key);
      return;
    }
    if (!has(key)) selectOnly(key);
  }

  /**
   * selectNext selects the item after the active one, or before it for a negative step, wrapping around
   * at the ends. Without an active item, it starts at the first or the last item.
   */
  function selectNext(step: 1 | -1): void {
    const all = keys();
    if (all.length === 0) return;

    const current = active.value === null ? -1 : all.indexOf(active.value);
    if (current === -1) {
      selected.value = new Set([step > 0 ? all[0]! : all[all.length - 1]!]);
      return;
    }
    selected.value = new Set([all[(current + step + all.length) % all.length]!]);
  }

  return {
    selected,
    active,
    has,
    set,
    clear,
    selectOnly,
    toggle,
    selectAll,
    press,
    selectNext,
  };
}
