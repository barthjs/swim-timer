import { computed, shallowRef, toRaw } from 'vue';

import { getAtPath, type Paths } from '@/lib/form/path';
import { objectKeys } from '@/lib/types/guards';

/**
 * Tracks which fields of a reactive state object have changed since the last capture(),
 * so a form can highlight edited fields and offer a reset to the captured baseline.
 * Fields are addressed by the same dot-separated path used as a UFormField's name.
 */
export function useDirtyForm<T extends Record<string, unknown>>(state: T) {
  const snapshot = shallowRef<Partial<T>>({});

  /** Records the current state as the baseline fields are compared against, after a load or a save. */
  function capture(): void {
    const raw = toRaw(state);
    const copy: Partial<T> = {};
    for (const key of objectKeys(raw)) {
      copy[key] = cloneValue(raw[key]);
    }
    snapshot.value = copy;
  }

  /** Reports whether a field differs from the baseline. */
  function isFieldDirty(path: Paths<T>): boolean {
    return (
      JSON.stringify(getAtPath(state, path)) !== JSON.stringify(getAtPath(snapshot.value, path))
    );
  }

  // Read through the reactive state so this recomputes as fields change
  const isDirty = computed(() => JSON.stringify(state) !== JSON.stringify(snapshot.value));

  /** Restores every top-level field of state to its captured baseline value. */
  function reset(): void {
    for (const key of objectKeys(snapshot.value)) {
      const field: keyof T = key;
      // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- capture copies every top-level field of the state
      state[field] = cloneValue(snapshot.value[field] as T[keyof T]);
    }
  }

  /** Spreads onto a Nuxt UI input to ring-highlight it in warning color while dirty. */
  function fieldProps(path: Paths<T>): { color: 'warning' | undefined; highlight: boolean } {
    const dirty = isFieldDirty(path);
    return {
      color: dirty ? 'warning' : undefined,
      highlight: dirty,
    };
  }

  return {
    isDirty,
    isFieldDirty,
    fieldProps,
    capture,
    reset,
  };
}

/** Copies a value, so the baseline does not change with the state. */
function cloneValue<V>(value: V): V {
  return value !== null && typeof value === 'object'
    ? // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- a JSON copy keeps the shape of plain data
      (JSON.parse(JSON.stringify(value)) as V)
    : value;
}
