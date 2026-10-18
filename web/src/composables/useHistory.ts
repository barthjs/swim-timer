import { computed, onScopeDispose, ref, shallowRef, toRaw, watch } from 'vue';

// How many checkpoints the history keeps unless the caller sets another limit.
const DEFAULT_MAX_HISTORY = 50;

// How long a change has to rest before it becomes a checkpoint of its own.
const AUTO_COMMIT_DELAY_MS = 400;

export type History = ReturnType<typeof useHistory<object>>;

export type HistoryOptions = {
  maxHistory?: number;
  // Holds back the automatic checkpoint, for example during a drag that commits once it ends.
  paused?: () => boolean;
};

/**
 * useHistory keeps an undo and redo history of a reactive object as full snapshots.
 * A commit without a change since the last entry adds no checkpoint. Undo and redo assign the snapshot
 * to the target in place, so its identity stays stable. Callers commit their own actions at once.
 * Every other change, such as a typed value or a series of nudges, becomes one checkpoint once it rests,
 * and undo or redo record a pending change first, so it is never lost.
 *
 *
 * The history is a shallowRef, since each entry is already a cloned plain object.
 * A deep ref would wrap the entries in reactive proxies, which structuredClone cannot clone.
 */
export function useHistory<T extends object>(target: T, options: HistoryOptions = {}) {
  const maxHistory = options.maxHistory ?? DEFAULT_MAX_HISTORY;
  const history = shallowRef<T[]>([]);
  const historyIndex = ref(-1);

  /** snapshotsEqual reports whether two snapshots hold the same state. */
  function snapshotsEqual(a: T, b: T): boolean {
    return JSON.stringify(a) === JSON.stringify(b);
  }

  /**
   * snapshot copies the current state. The target is usually a reactive proxy, which structuredClone cannot clone,
   * so the raw object is copied.
   */
  function snapshot(): T {
    return structuredClone(toRaw(target));
  }

  let pending: ReturnType<typeof setTimeout> | null = null;

  /** cancelPending drops the scheduled automatic checkpoint. */
  function cancelPending(): void {
    if (pending === null) return;
    clearTimeout(pending);
    pending = null;
  }

  /** reset starts a fresh history at the target's current state. */
  function reset(): void {
    cancelPending();
    history.value = [snapshot()];
    historyIndex.value = 0;
  }

  /** commit pushes the current state as a history checkpoint, unless nothing changed since the last one. */
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

  /** undo restores the previous checkpoint. */
  function undo(): void {
    move(-1);
  }

  /** redo restores the checkpoint that was undone last. */
  function redo(): void {
    move(1);
  }

  /** move shows the neighboring checkpoint after recording a pending change, so the change is never lost. */
  function move(offset: -1 | 1): void {
    if (pending !== null) commit();
    const index = historyIndex.value + offset;
    const entry = history.value[index];
    if (!entry) return;
    historyIndex.value = index;
    Object.assign(target, structuredClone(entry));
  }

  /** autoCommit records a resting change, unless a gesture started meanwhile that commits on its own. */
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
