import type { Form, FormError, FormSchema } from '@nuxt/ui';

import { useValidationMessages } from '@/composables/form/useValidationMessages';
import { checkFields, type FormRules } from '@/lib/form/validation';

/**
 * The part of a Nuxt UI form instance that receives the field errors of a failed request.
 * Every form that shows server errors is passed as this type.
 */
export type FormErrorTarget = Pick<Form<FormSchema>, 'setErrors' | 'clear'>;

/** Builds the validate function of a UForm from rules of the state, with the messages of the server's tags. */
export function useFormValidation<T extends object>(rules: (state: T) => FormRules<T>) {
  const { mapFieldErrors } = useValidationMessages();

  /** Checks the state and returns the errors for the form fields. */
  function validate(state: T): FormError[] {
    return mapFieldErrors(checkFields(state, rules(state)), state);
  }

  return {
    validate,
  };
}
