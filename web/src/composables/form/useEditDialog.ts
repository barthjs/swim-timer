import { type MaybeRefOrGetter, reactive, ref, shallowRef, watch, type Ref } from 'vue';

import { useDirtyForm } from '@/composables/form/useDirtyForm';
import { useDiscardConfirm } from '@/composables/form/useDiscardConfirm';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation, type UseMutationOptions } from '@/composables/form/useMutation';
import type { Paths } from '@/lib/form/path';

export type UseEditDialogOptions<TRow, TState extends Record<string, unknown>, TResult> = Pick<
  UseMutationOptions<[], TResult>,
  'toast' | 'onError'
> & {
  // Returns the values of the form for a record, or the defaults of a new one
  initial: (row: TRow | null) => TState;
  // A dialog that only edits leaves it out, and saving a new record then fails
  create?: (state: TState) => Promise<TResult>;
  // Receives isFieldDirty, so an update can send only the changed fields, and a dialog that only creates leaves it out
  update?: (
    row: TRow,
    state: TState,
    isFieldDirty: (path: Paths<TState>) => boolean,
  ) => Promise<TResult>;
  // The success toast of a new or an edited record
  success?: (isEdit: boolean) => string;
  // Runs after a successful save, once the dialog is closed
  onSaved?: (result: TResult, isEdit: boolean) => void | Promise<void>;
  // Runs whenever the dialog starts, such as to reset parts of the form that are not in the state
  onStart?: (row: TRow | null) => void;
  // The form that receives the field errors of a 422
  form?: MaybeRefOrGetter<FormErrorTarget | null | undefined>;
  // The open state of a dialog component, whose parent opens it.
  // The dialog then starts from row when it opens, or blank without one.
  open?: Ref<boolean>;
  row?: () => TRow | null | undefined;
};

/**
 * Drives a dialog that creates or edits a record and asks before unsaved changes are dropped,
 * opened with openCreate and openEdit or by the open model and record of a dialog component.
 */
export function useEditDialog<TRow, TState extends Record<string, unknown>, TResult>(
  options: UseEditDialogOptions<TRow, TState, TResult>,
) {
  const open = options.open ?? ref(false);
  const editing = shallowRef<TRow | null>(null);
  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the state holds no refs, so reactive keeps its type
  const state = reactive(options.initial(null)) as TState;
  const { isDirty, isFieldDirty, fieldProps, capture } = useDirtyForm(state);
  capture();

  /** Fills the form from the record or the defaults and takes it as the baseline of the changed fields. */
  function start(row: TRow | null): void {
    editing.value = row;
    Object.assign(state, options.initial(row));
    options.onStart?.(row);
    capture();
  }

  /** Opens the dialog for a new record. */
  function openCreate(): void {
    start(null);
    open.value = true;
  }

  /** Opens the dialog for an existing record. */
  function openEdit(row: TRow): void {
    start(row);
    open.value = true;
  }

  // A dialog component starts from its record, or blank, whenever its parent opens it
  if (options.open) {
    watch(open, (isOpen) => {
      if (isOpen) start(options.row?.() ?? null);
    });
  }

  /** Closes the dialog without asking. */
  function close(): void {
    open.value = false;
  }

  const success = options.success;
  const {
    mutate: save,
    pending: saving,
    error,
    errorMessage,
  } = useMutation(
    async () => {
      if (editing.value !== null && options.update) {
        return options.update(editing.value, state, isFieldDirty);
      }
      if (!options.create) return Promise.reject(new Error('the dialog only edits records'));
      return options.create(state);
    },
    {
      form: options.form,
      values: state,
      toast: options.toast,
      onError: options.onError,
      success: success ? () => success(editing.value !== null) : undefined,
      onSuccess: async (result) => {
        const isEdit = editing.value !== null;
        close();
        await options.onSaved?.(result, isEdit);
      },
    },
  );

  const { openProxy, attemptClose } = useDiscardConfirm(open, isDirty, saving);

  return {
    open: openProxy,
    editing,
    state,
    isDirty,
    isFieldDirty,
    fieldProps,
    openCreate,
    openEdit,
    close,
    attemptClose,
    save,
    saving,
    error,
    errorMessage,
  };
}
