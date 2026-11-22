import { onMounted, ref, watch, type Ref } from 'vue';

/**
 * Keeps a value of one viewer in the localStorage of the browser, read once the component mounts,
 * and as long as the component lives without a key or storage.
 */
export function useStoredState<T>(
  key: string | undefined,
  fallback: () => T,
  parse: (raw: unknown) => T | undefined,
): Ref<T> {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- a stored value holds no refs, so ref keeps its type
  const state = ref(fallback()) as Ref<T>;

  // Reading on mount lets a prerendered page hydrate with the fallback first
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

/** Returns the parsed JSON stored under a key, or undefined without a readable value. */
function readStored(key: string): unknown {
  try {
    const raw = localStorage.getItem(key);
    if (raw === null) return undefined;
    return JSON.parse(raw) as unknown;
  } catch {
    return undefined;
  }
}
