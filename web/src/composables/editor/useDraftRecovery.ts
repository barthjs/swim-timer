import { onScopeDispose, toRaw, watch, type Ref } from 'vue';

// How long changes rest before the browser keeps them as a draft
const SAVE_DELAY_MS = 1000;

export type DraftRecoveryOptions<T> = {
  // The prefix of the keys the browser keeps the drafts under, such as "editor"
  storageKey: string;
  // The saved document the draft belongs to. A document that is not saved yet keeps no draft.
  document: () => string | null;
  state: T;
  // Whether the state has changes that are not saved
  dirty: Ref<boolean>;
  // Asks whether to restore a draft, which offer calls before it restores one
  confirm: () => Promise<boolean>;
};

/** Keeps the unsaved changes of a saved document in the browser until they are saved or discarded. */
export function useDraftRecovery<T extends object>(options: DraftRecoveryOptions<T>) {
  let pending: ReturnType<typeof setTimeout> | null = null;

  /** Returns the key the draft of a document is kept under. */
  function keyOf(name: string): string {
    return `${options.storageKey}.draft.${name}`;
  }

  /** Drops the scheduled save of the draft. */
  function cancelPending(): void {
    if (pending === null) return;
    clearTimeout(pending);
    pending = null;
  }

  /** Keeps the current state as the draft of the open document, or deletes the draft without changes. */
  function save(): void {
    pending = null;
    const name = options.document();
    if (name === null) return;
    try {
      if (!options.dirty.value) {
        localStorage.removeItem(keyOf(name));
        return;
      }
      localStorage.setItem(keyOf(name), JSON.stringify(toRaw(options.state)));
    } catch {
      // Without storage, the changes only live until the page is left
    }
  }

  /** Asks to restore a kept draft that differs from the state, deletes a declined one, and reports whether it restored it. */
  async function offer(): Promise<boolean> {
    const name = options.document();
    if (name === null) return false;

    let stored: string | null = null;
    try {
      stored = localStorage.getItem(keyOf(name));
    } catch {
      return false;
    }
    if (stored === null || stored === JSON.stringify(toRaw(options.state))) return false;

    if (!(await options.confirm())) {
      discard(name);
      return false;
    }
    try {
      Object.assign(options.state, JSON.parse(stored));
    } catch {
      discard(name);
      return false;
    }
    return true;
  }

  /** Deletes the draft of a document. */
  function discard(name: string): void {
    cancelPending();
    try {
      localStorage.removeItem(keyOf(name));
    } catch {
      // Without storage, there is nothing to delete
    }
  }

  watch(
    () => options.state,
    () => {
      cancelPending();
      pending = setTimeout(save, SAVE_DELAY_MS);
    },
    {
      deep: true,
    },
  );

  // Saved or discarded changes leave nothing to recover
  watch(options.dirty, (dirty) => {
    if (dirty) return;
    const name = options.document();
    if (name !== null) discard(name);
  });

  onScopeDispose(cancelPending);

  return {
    offer,
    discard,
  };
}
