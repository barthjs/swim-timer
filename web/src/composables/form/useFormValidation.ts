import type { Form, FormError, FormSchema } from '@nuxt/ui';

import { useValidationMessages } from '@/composables/form/useValidationMessages';
import { checkFields, type FormRules } from '@/lib/form/validation';

/**
 * FormErrorTarget is the part of a Nuxt UI form instance that receives the field errors of a failed request.
 * Every form that shows server errors is passed as this type.
 */
export type FormErrorTarget = Pick<Form<FormSchema>, 'setErrors' | 'clear'>;

/**
 * useFormValidation builds the validate function of a UForm from declared rules. The rules are a function
 * of the state, so a field can be required depending on another one. Failed rules produce the same tags
 * as the server's validation errors and get their messages from the same translations.
 */
export function useFormValidation<T extends object>(rules: (state: T) => FormRules<T>) {
  const { mapFieldErrors } = useValidationMessages();

  /** validate checks the state and returns the errors for the form fields. */
  function validate(state: T): FormError[] {
    return mapFieldErrors(checkFields(state, rules(state)), state);
  }

  return {
    validate,
  };
}
