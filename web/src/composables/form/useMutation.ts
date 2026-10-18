import { computed, type MaybeRefOrGetter, ref, toValue } from 'vue';
import { useToast } from '@nuxt/ui/composables';

import { ApiError } from '@/api/errors';
import { useApiError } from '@/composables/useApiError';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useValidationMessages } from '@/composables/form/useValidationMessages';

export type UseMutationOptions<TArgs extends unknown[], TResult> = {
  // Success toast title. A function receives the result and arguments so the message can name the affected entity.
  // Omitted means no success toast.
  success?: string | ((result: TResult, ...args: TArgs) => string);
  successIcon?: string;
  // Runs after a successful mutation, for example to advance to the next screen
  // or store the result. Awaited so the caller can sequence follow-up work.
  onSuccess?: (result: TResult, ...args: TArgs) => void | Promise<void>;
  // Runs on error before the default toast. Returning true marks the error
  // as fully handled and suppresses the default error toast.
  onError?: (error: unknown, ...args: TArgs) => boolean | undefined;
  // A form whose inline errors should receive 422 field errors. When set, a validation error is mapped
  // onto the form instead of surfacing as a toast.
  form?: MaybeRefOrGetter<FormErrorTarget | null | undefined>;
  // The values the field errors of a 422 refer to, usually the form state. The messages read the kind
  // of each failed value from it, so max is worded as characters for a text field.
  values?: MaybeRefOrGetter<object | null | undefined>;
  // Whether non-field errors surface as a toast. Disable when the caller renders the error itself
  // (such as an inline message via errorMessage). Defaults to true.
  toast?: boolean;
};

/**
 * useMutation runs a write request for a form or an action. It tracks whether the request is pending,
 * shows an optional success toast, and runs a follow-up hook. Validation errors go to the form
 * as field errors, and every other error is shown as a localized toast. Without a form, the toast
 * of a validation error names the first failed field.
 */
export function useMutation<TArgs extends unknown[], TResult>(
  fn: (...args: TArgs) => Promise<TResult>,
  options: UseMutationOptions<TArgs, TResult> = {},
) {
  const toast = useToast();
  const { messageFor } = useApiError();
  const { mapFieldErrors } = useValidationMessages();

  const pending = ref(false);
  const error = ref<unknown>(null);
  const errorMessage = computed(() => (error.value ? messageFor(error.value) : ''));

  /** resolveForm returns the form that receives field errors, if any. */
  const resolveForm = (): FormErrorTarget | null => toValue(options.form) ?? null;

  /** mutate runs the request and handles its success and errors. */
  async function mutate(...args: TArgs): Promise<TResult | undefined> {
    pending.value = true;
    error.value = null;
    // Clear errors from a previous attempt, so resubmitting a form does not show stale field messages
    // alongside the new ones.
    resolveForm()?.clear();

    try {
      const result = await fn(...args);

      if (options.success !== undefined) {
        const title =
          typeof options.success === 'function'
            ? options.success(result, ...args)
            : options.success;
        toast.add({
          title,
          icon: options.successIcon ?? 'i-lucide-check-circle',
          color: 'success',
        });
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
      pending.value = false;
    }
  }

  /** firstFieldError returns the message of the first failed field of a validation error, if any. */
  function firstFieldError(err: unknown): string | undefined {
    if (!(err instanceof ApiError) || !err.fields) return undefined;
    return mapFieldErrors(err.fields, toValue(options.values))[0]?.message;
  }

  return {
    mutate,
    pending,
    error,
    errorMessage,
  };
}
