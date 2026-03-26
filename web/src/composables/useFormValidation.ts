import { computed, reactive } from 'vue';
import { useI18n } from 'vue-i18n';

interface ValidationError {
  rule: string;
  params?: Record<string, string>;
}

export function useFormValidation() {
  const { t } = useI18n();
  const errors = reactive<Record<string, string>>({});

  const getErrorMessage = (error: ValidationError | string): string => {
    if (typeof error === 'string') {
      error = { rule: error };
    }

    const key = `validation.${error.rule}`;
    const params = error.params || {};
    return t(key, params);
  };

  const setErrors = (fieldErrors: Record<string, ValidationError | string>) => {
    // Clear previous errors
    Object.keys(errors).forEach((key) => {
      delete errors[key];
    });

    // Set new errors with translated messages
    Object.entries(fieldErrors).forEach(([field, error]) => {
      errors[field] = getErrorMessage(error);
    });
  };

  const clearErrors = () => {
    Object.keys(errors).forEach((key) => {
      delete errors[key];
    });
  };

  const clearFieldError = (field: string) => {
    delete errors[field];
  };

  const hasErrors = computed(() => Object.keys(errors).length > 0);

  return {
    errors,
    setErrors,
    clearErrors,
    clearFieldError,
    hasErrors,
  };
}
