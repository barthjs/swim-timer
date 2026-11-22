import { computed, type MaybeRefOrGetter, reactive, ref, toValue, watch } from 'vue';
import { useToast, type Toast } from '@nuxt/ui/composables';

import { ApiError } from '@/api/client/errors';
import { useApiError } from '@/composables/form/useApiError';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useValidationMessages } from '@/composables/form/useValidationMessages';
import { changedPaths } from '@/lib/form/path';

export type UseMutationOptions<TArgs extends unknown[], TResult> = {
  // Names the target of a call, such as the row of a table that shares one mutation.
  // A call is only ignored while a call with the same key runs, so actions on other targets still run. Omitted means one key for all calls.
  key?: (...args: TArgs) => string;
  // Asks before the request, such as with a confirmation dialog, and the request only runs once it resolves to true
  confirm?: (...args: TArgs) => Promise<boolean>;
  // Title of a toast shown while the request runs, for a request that takes a while such as an upload.
  // It turns into the success toast, or closes before the error toast.
  loading?: string | ((...args: TArgs) => string);
  // Success toast title. A function receives the result and arguments so the message can name the affected entity.
  // Omitted means no success toast.
  success?: string | ((result: TResult, ...args: TArgs) => string);
  successIcon?: string;
  // Runs after a successful mutation, for example to advance to the next screen
  // or store the result. Awaited so the caller can sequence follow-up work.
  onSuccess?: (result: TResult, ...args: TArgs) => void | Promise<void>;
  // Runs on error before the default toast. Returning true marks the error as fully handled
  // and suppresses the default error toast.
  onError?: (error: unknown, ...args: TArgs) => boolean | undefined;
  // A form whose inline errors should receive 422 field errors.
  // When set, a validation error is mapped onto the form instead of surfacing as a toast.
  form?: MaybeRefOrGetter<FormErrorTarget | null | undefined>;
  // The values the field errors of a 422 refer to, usually the form state.
  // The messages read the kind of each failed value from it, so max is worded as characters for a text field.
  values?: MaybeRefOrGetter<object | null | undefined>;
  // Whether non-field errors surface as a toast. Disable when the caller renders the error itself
  // (such as an inline message via errorMessage). Defaults to true.
  toast?: boolean;
};

/** The key of every call when the options set no key function. */
const unkeyed = Symbol('unkeyed');

/**
 * Runs a write request for a form or an action after an optional confirmation,
 * with its pending state, toasts, and the validation errors at the fields of the form.
 */
export function useMutation<TArgs extends unknown[], TResult>(
  fn: (...args: TArgs) => Promise<TResult>,
  options: UseMutationOptions<TArgs, TResult> = {},
) {
  const toast = useToast();
  const { messageFor } = useApiError();
  const { mapFieldErrors } = useValidationMessages();

  const running = reactive(new Set<string | symbol>());
  const pending = computed(() => running.size > 0);
  const error = ref<unknown>(null);
  const errorMessage = computed(() =>
    error.value === null || error.value === undefined ? '' : messageFor(error.value),
  );

  /** Returns the form that receives field errors, if any. */
  const resolveForm = (): FormErrorTarget | null => toValue(options.form) ?? null;

  // A form shows its errors only after a submit, so an edited field drops its error until the next one
  if (options.form && options.values) {
    watch(
      () => JSON.stringify(toValue(options.values) ?? null),
      (after, before) => {
        const form = resolveForm();
        if (!form) return;
        for (const path of changedPaths(JSON.parse(before), JSON.parse(after))) {
          form.clear(path);
        }
      },
    );
  }

  /** Returns whether a call with the given key runs. */
  const isPending = (key: string): boolean => running.has(key);

  /**
   * Runs the request, after a confirmation if one is set, and handles its success and errors.
   * A call while a call with the same key runs is ignored, so a second click never sends the request twice.
   */
  async function mutate(...args: TArgs): Promise<TResult | undefined> {
    const key = options.key?.(...args) ?? unkeyed;
    if (running.has(key)) return undefined;
    if (options.confirm && !(await options.confirm(...args))) return undefined;
    // Another call with the same key may have started while the confirmation was open
    if (running.has(key)) return undefined;

    running.add(key);
    error.value = null;
    // Clear errors from a previous attempt, so resubmitting a form does not show stale field messages alongside the new ones
    resolveForm()?.clear();

    try {
      const request = fn(...args);
      const loading =
        typeof options.loading === 'function' ? options.loading(...args) : options.loading;
      const result = await (loading === undefined
        ? request
        : toast.promise(request, {
            loading: {
              title: loading,
            },
            success:
              options.success === undefined ? undefined : (value) => successToast(value, args),
          }));

      if (loading === undefined && options.success !== undefined) {
        toast.add(successToast(result, args));
      }

      await options.onSuccess?.(result, ...args);
      return result;
    } catch (err) {
      error.value = err;

      // Field-level validation errors belong inline on the form, not in a toast
      const form = resolveForm();
      if (form && err instanceof ApiError && err.fields) {
        form.setErrors(mapFieldErrors(err.fields, toValue(options.values)));
        return undefined;
      }

      if (options.onError?.(err, ...args) === true) {
        return undefined;
      }

      if (options.toast !== false) {
        toast.add({
          title: messageFor(err),
          description: firstFieldError(err),
          icon: 'i-lucide-circle-alert',
          color: 'error',
        });
      }
      return undefined;
    } finally {
      running.delete(key);
    }
  }

  /** Returns the toast of a successful request, titled by the success option. */
  function successToast(result: TResult, args: TArgs): Partial<Toast> {
    return {
      title:
        typeof options.success === 'function' ? options.success(result, ...args) : options.success,
      icon: options.successIcon ?? 'i-lucide-check-circle',
      color: 'success',
    };
  }

  /** Returns the message of the first failed field of a validation error, if any. */
  function firstFieldError(err: unknown): string | undefined {
    if (!(err instanceof ApiError) || !err.fields) return undefined;
    return mapFieldErrors(err.fields, toValue(options.values))[0]?.message;
  }

  return {
    mutate,
    pending,
    isPending,
    error,
    errorMessage,
  };
}
