import { shallowRef } from 'vue';

export type EditorVisibility<K> = ReturnType<typeof useEditorVisibility<K>>;

/** Holds the items an editor neither draws, selects, nor snaps to for a while, apart from the document. */
export function useEditorVisibility<K>() {
  // The set is always replaced instead of changed in place, so every change triggers reactivity
  const hidden = shallowRef(new Set<K>());

  /** Reports whether an item is hidden. */
  function isHidden(key: K): boolean {
    return hidden.value.has(key);
  }

  /** Hides an item or shows it again. */
  function toggle(key: K): void {
    const next = new Set(hidden.value);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    hidden.value = next;
  }

  /** Shows every item again. */
  function showAll(): void {
    hidden.value = new Set();
  }

  return {
    hidden,
    isHidden,
    toggle,
    showAll,
  };
}
