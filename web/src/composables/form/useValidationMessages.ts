import { useI18n } from 'vue-i18n';
import type { FormError } from '@nuxt/ui';

import type { FieldErrors } from '@/api/client/errors';
import { getAtPath } from '@/lib/form/path';

/** Matches a field path that ends in an index in brackets and captures the path of the list and the index. */
const LIST_ENTRY_PATTERN = /^(.+)\[(\d+)\]$/;

/** Groups the values whose rules are worded differently. */
type ValueKind = 'string' | 'number' | 'array';

/** Returns the kind of field value, or undefined for a value without a specific wording. */
function valueKind(value: unknown): ValueKind | undefined {
  if (typeof value === 'string') return 'string';
  if (typeof value === 'number') return 'number';
  if (Array.isArray(value)) return 'array';
  return undefined;
}

/**
 * Maps field validation errors to messages that name the field from `fields.<name>`
 * with the rule text from `validation.<tag>`, or from `validation.<tag>_<kind>` for the kind of the checked value.
 */
export function useValidationMessages() {
  const { t, te } = useI18n();

  /** Returns the translation key of a rule, preferring the text for the kind of value. */
  function ruleKey(tag: string, value: unknown): string | undefined {
    const kind = valueKind(value);
    if (kind && te(`validation.${tag}_${kind}`)) return `validation.${tag}_${kind}`;
    if (te(`validation.${tag}`)) return `validation.${tag}`;
    return undefined;
  }

  /**
   * Turns field errors into form errors with the message of the first rule of each field
   * and of the first failed entry of a list, reading the kind of each value from the checked values.
   */
  const mapFieldErrors = (errors: FieldErrors, values?: unknown): FormError[] => {
    const result: FormError[] = [];

    for (const field in errors) {
      const first = errors[field]?.[0];
      if (!first) continue;

      const target = errorTarget(field, values);
      if (result.some((error) => error.name === target.name)) continue;

      const key = ruleKey(first.tag, target.value);
      const message = key
        ? t(key, {
            field: target.label,
            param: first.param ?? '',
          })
        : t('validation.invalid', {
            field: target.label,
          });

      result.push({
        name: target.name,
        message,
      });
    }

    return result;
  };

  /** Returns the form field that shows the error of a reported field, its label, and the checked value. */
  function errorTarget(
    field: string,
    values: unknown,
  ): {
    name: string;
    label: string;
    value: unknown;
  } {
    const entry = LIST_ENTRY_PATTERN.exec(field);
    const list = entry?.[1];
    const items = list ? getAtPath(values, list) : undefined;
    if (!entry?.[2] || !list || !Array.isArray(items)) {
      return {
        name: field,
        label: labelFor(field),
        value: getAtPath(values, field),
      };
    }

    const index = Number(entry[2]);
    return {
      name: list,
      label: t('validation.listEntry', {
        field: labelFor(list),
        position: index + 1,
      }),
      value: (items as unknown[])[index],
    };
  }

  /** Returns the translated label of a field, or its raw name if it has none. */
  function labelFor(field: string): string {
    const fieldKey = `fields.${field}`;
    return te(fieldKey) ? t(fieldKey) : field;
  }

  return {
    mapFieldErrors,
  };
}
