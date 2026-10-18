import { useI18n } from 'vue-i18n';
import type { FormError } from '@nuxt/ui';

import type { FieldErrors } from '@/api/errors';
import { getAtPath } from '@/lib/form/path';

/** Matches a field path that ends in an index in brackets and captures the path of the list and the index. */
const LIST_ENTRY_PATTERN = /^(.+)\[(\d+)\]$/;

/** ValueKind groups the values whose rules are worded differently, such as the limit of max. */
type ValueKind = 'string' | 'number' | 'array';

/** valueKind returns the kind of field value, or undefined for a value without a specific wording. */
function valueKind(value: unknown): ValueKind | undefined {
  if (typeof value === 'string') return 'string';
  if (typeof value === 'number') return 'number';
  if (Array.isArray(value)) return 'array';
  return undefined;
}

/**
 * useValidationMessages maps field validation errors to localized messages that name the field,
 * such as "Name is required". The label comes from `fields.<name>` and the rule text from `validation.<tag>`.
 * A rule that limits characters, numbers, and entries alike, such as max, can have a text per kind of value,
 * such as `validation.max_string`, which is used when the checked values are known.
 * An unknown field shows its raw name and an unknown rule the generic `validation.invalid` message,
 * so a new server rule never shows a translation key. The forms edit a list in one field, so the error
 * of a list entry goes to the field of the list and names the position of the entry.
 */
export function useValidationMessages() {
  const { t, te } = useI18n();

  /** ruleKey returns the translation key of a rule, preferring the text for the kind of value. */
  function ruleKey(tag: string, value: unknown): string | undefined {
    const kind = valueKind(value);
    if (kind && te(`validation.${tag}_${kind}`)) return `validation.${tag}_${kind}`;
    if (te(`validation.${tag}`)) return `validation.${tag}`;
    return undefined;
  }

  /**
   * mapFieldErrors turns field errors into form errors with the message of the first rule of each field.
   * values are the checked values, such as the form state or the request body,
   * from which the kind of each failed value is read. Only the first failed entry of a list is reported.
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

  /**
   * errorTarget returns the form field that shows the error of a reported field, its label, and the checked value.
   * A list entry is shown at the field of its list. An indexed name whose base is not a list in the checked values,
   * such as the key of a map, keeps its raw name.
   */
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

  /** labelFor returns the translated label of a field, or its raw name if it has none. */
  function labelFor(field: string): string {
    const fieldKey = `fields.${field}`;
    return te(fieldKey) ? t(fieldKey) : field;
  }

  return {
    mapFieldErrors,
  };
}
