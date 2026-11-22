import { computed, onScopeDispose, ref, shallowRef, toRaw, watch } from 'vue';

// How many checkpoints the history keeps unless the caller sets another limit
const DEFAULT_MAX_HISTORY = 50;

// How long a change has to rest before it becomes a checkpoint of its own
const AUTO_COMMIT_DELAY_MS = 400;

export type History = ReturnType<typeof useHistory>;

export type HistoryOptions = {
  maxHistory?: number;
  // Holds back the automatic checkpoint, for example during a drag that commits once it ends
  paused?: () => boolean;
};

/**
 * Keeps an undo and redo history of a reactive object as full snapshots, restored in place,
 * where callers commit their own actions and every other change becomes a checkpoint once it rests.
 */
export function useHistory(target: object, options: HistoryOptions = {}) {
  const maxHistory = options.maxHistory ?? DEFAULT_MAX_HISTORY;
  // Each entry is already a cloned plain object, and a deep ref would wrap it in a proxy structuredClone cannot clone
  const history = shallowRef<object[]>([]);
  const historyIndex = ref(-1);

  /**
   * Copies the current state. The target is usually a reactive proxy,
   * which structuredClone cannot clone, so the raw object is copied.
   */
  function snapshot(): object {
    return structuredClone(toRaw(target));
  }

  let pending: ReturnType<typeof setTimeout> | null = null;

  /** Drops the scheduled automatic checkpoint. */
  function cancelPending(): void {
    if (pending === null) return;
    clearTimeout(pending);
    pending = null;
  }

  /** Starts a fresh history at the target's current state. */
  function reset(): void {
    cancelPending();
    history.value = [snapshot()];
    historyIndex.value = 0;
  }

  /** Pushes the current state as a history checkpoint, unless nothing changed since the last one. */
  function commit(): void {
    cancelPending();
    const current = snapshot();
    const last = history.value[historyIndex.value];
    if (last && snapshotsEqual(last, current)) return;

    const truncated = [...history.value.slice(0, historyIndex.value + 1), current];
    if (truncated.length > maxHistory) {
      truncated.shift();
    } else {
      historyIndex.value++;
    }
    history.value = truncated;
  }

  const canUndo = computed(() => historyIndex.value > 0);
  const canRedo = computed(() => historyIndex.value < history.value.length - 1);

  /** Restores the previous checkpoint. */
  function undo(): void {
    move(-1);
  }

  /** Restores the checkpoint that was undone last. */
  function redo(): void {
    move(1);
  }

  /** Shows the neighboring checkpoint after recording a pending change, so the change is never lost. */
  function move(offset: -1 | 1): void {
    if (pending !== null) commit();
    const index = historyIndex.value + offset;
    const entry = history.value[index];
    if (!entry) return;
    historyIndex.value = index;
    Object.assign(target, structuredClone(entry));
  }

  /** Records a resting change, unless a gesture started meanwhile that commits on its own. */
  function autoCommit(): void {
    pending = null;
    if (!options.paused?.()) commit();
  }

  // Restoring a snapshot changes the target as well, which the commit then skips as unchanged
  watch(
    () => target,
    () => {
      if (history.value.length === 0 || options.paused?.()) return;
      cancelPending();
      pending = setTimeout(autoCommit, AUTO_COMMIT_DELAY_MS);
    },
    {
      deep: true,
    },
  );

  onScopeDispose(cancelPending);

  return {
    reset,
    commit,
    undo,
    redo,
    canUndo,
    canRedo,
  };
}

/** Reports whether two snapshots hold the same state. */
function snapshotsEqual(a: object, b: object): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}
