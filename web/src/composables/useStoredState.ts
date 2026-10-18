import { onMounted, ref, watch, type Ref } from 'vue';

/**
 * useStoredState keeps a value in the localStorage of the browser, for conveniences such as the layout of a table
 * that only concern one viewer on one device. The stored value is read once the component mounts,
 * so a prerendered page hydrates with the fallback first. Without a key, or where the browser denies the storage,
 * the value lives as long as the component.
 */
export function useStoredState<T>(
  key: string | undefined,
  fallback: () => T,
  parse: (raw: unknown) => T | undefined,
): Ref<T> {
  const state = ref(fallback()) as Ref<T>;

  onMounted(() => {
    if (!key) return;
    const stored = readStored(key);
    if (stored === undefined) return;
    const parsed = parse(stored);
    if (parsed !== undefined) state.value = parsed;
  });

  watch(
    state,
    (value) => {
      if (!key) return;
      try {
        localStorage.setItem(key, JSON.stringify(value));
      } catch {
        // A browser without storage access keeps the value for this visit only
      }
    },
    {
      deep: true,
    },
  );

  return state;
}

/** readStored returns the parsed JSON stored under a key, or undefined without a readable value. */
function readStored(key: string): unknown {
  try {
    const raw = localStorage.getItem(key);
    if (raw === null) return undefined;
    return JSON.parse(raw) as unknown;
  } catch {
    return undefined;
  }
}
