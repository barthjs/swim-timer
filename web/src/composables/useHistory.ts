import { ref } from 'vue';

export function useHistory<T>(target: { value: T | null }) {
  const history = ref<string[]>([]);
  const historyIndex = ref(-1);

  const saveHistory = (): void => {
    if (target.value === null) return;
    const state = JSON.stringify(target.value);

    if (historyIndex.value >= 0 && history.value[historyIndex.value] === state) {
      return;
    }

    history.value = history.value.slice(0, historyIndex.value + 1);
    history.value.push(state);
    historyIndex.value++;
  };

  const undo = (): void => {
    if (historyIndex.value > 0 && target.value !== null) {
      historyIndex.value--;
      target.value = JSON.parse(history.value[historyIndex.value]!) as T;
    }
  };

  const redo = (): void => {
    if (historyIndex.value < history.value.length - 1 && target.value !== null) {
      historyIndex.value++;
      target.value = JSON.parse(history.value[historyIndex.value]!) as T;
    }
  };

  const clearHistory = (): void => {
    history.value = [];
    historyIndex.value = -1;
  };

  return { saveHistory, undo, redo, clearHistory };
}
