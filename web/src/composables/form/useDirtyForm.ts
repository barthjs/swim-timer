import { computed, ref, toRaw } from 'vue';

import { getAtPath, type Paths } from '@/lib/form/path';

/**
 * useDirtyForm tracks which fields of a reactive state object have changed since the last capture(),
 * so a form can highlight edited fields and offer a reset to the captured baseline.
 * Fields are addressed by the same dot-separated path used as a UFormField's name.
 */
export function useDirtyForm<T extends Record<string, unknown>>(state: T) {
  const snapshot = ref<Partial<T>>({});

  /** cloneValue copies a value, so the baseline does not change with the state. */
  function cloneValue<V>(value: V): V {
    return value !== null && typeof value === 'object'
      ? (JSON.parse(JSON.stringify(value)) as V)
      : value;
  }

  /**
   * capture records the current state as the baseline fields are compared against.
   * Call it once the state holds a value worth returning to, for example after a successful load or save.
   */
  function capture(): void {
    const raw = toRaw(state);
    const copy = {} as Partial<T>;
    for (const key of Object.keys(raw) as (keyof T)[]) {
      copy[key] = cloneValue(raw[key]);
    }
    snapshot.value = copy;
  }

  /** isFieldDirty reports whether a field differs from the baseline. */
  function isFieldDirty(path: Paths<T>): boolean {
    return (
      JSON.stringify(getAtPath(state, path)) !== JSON.stringify(getAtPath(snapshot.value, path))
    );
  }

  // Read through the reactive state so this recomputes as fields change
  const isDirty = computed(() => JSON.stringify(state) !== JSON.stringify(snapshot.value));

  /** reset restores every top-level field of state to its captured baseline value. */
  function reset(): void {
    for (const key of Object.keys(snapshot.value) as (keyof T)[]) {
      state[key] = cloneValue(snapshot.value[key] as T[keyof T]);
    }
  }

  /** fieldProps spreads onto a Nuxt UI input to ring-highlight it in warning color while dirty. */
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
