import { ref, type Ref } from 'vue';

export type EditorVisibility<K> = ReturnType<typeof useEditorVisibility<K>>;

/**
 * useEditorVisibility holds the items of an editor that are hidden for a while, such as to reach one that another
 * covers. A hidden item is neither drawn nor selectable, and nothing snaps to it. Hiding is an aid of the editor
 * and not part of the document. The set is always replaced instead of changed in place, so every change triggers
 * reactivity.
 */
export function useEditorVisibility<K>() {
  const hidden: Ref<Set<K>> = ref(new Set<K>()) as Ref<Set<K>>;

  /** isHidden reports whether an item is hidden. */
  function isHidden(key: K): boolean {
    return hidden.value.has(key);
  }

  /** toggle hides an item or shows it again. */
  function toggle(key: K): void {
    const next = new Set(hidden.value);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    hidden.value = next;
  }

  /** showAll shows every item again, such as when another document opens. */
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
