import { ref } from 'vue';

// Theme colors accepted by Nuxt UI buttons.
type UiColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';

export type ConfirmOptions = {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  color?: UiColor;
  icon?: string;
};

export type ConfirmState = ConfirmOptions & { open: boolean };

/** Shared by every caller, since a single AppConfirmModal in the root component renders it. */
const state = ref<ConfirmState>({
  open: false,
  title: '',
});

// Holds the pending promise resolver. Confirms are sequential, so a single resolver is enough.
let resolver: ((value: boolean) => void) | null = null;

/**
 * useConfirm opens a confirmation dialog and resolves once it is answered, so a caller can write
 * `if (await confirm({ title, description }))`. A single AppConfirmModal in the root component renders it.
 */
export function useConfirm() {
  /** confirm opens the dialog and resolves to true once it is confirmed. */
  const confirm = (options: ConfirmOptions): Promise<boolean> =>
    new Promise<boolean>((resolve) => {
      // Resolve a previous, still-open confirm as canceled before replacing it
      resolver?.(false);
      resolver = resolve;
      state.value = {
        ...options,
        open: true,
      };
    });

  /** settle closes the dialog and resolves the pending confirmation. */
  const settle = (value: boolean) => {
    state.value = {
      ...state.value,
      open: false,
    };
    const resolve = resolver;
    resolver = null;
    resolve?.(value);
  };

  return {
    state,
    confirm,
    settle,
  };
}
