import type { FormInputEvents } from '@nuxt/ui';

import type { FieldErrors } from '@/api/errors';
import { getAtPath, type Paths } from '@/lib/form/path';

/**
 * FORM_VALIDATE_ON are the events after which a form checks a field again. Every form passes them as validate-on.
 * A field that only got and lost the focus stays without an error. It is checked on submit, once it was changed
 * and left, and while typing in a field that was left before, so a shown error clears as soon as it is fixed.
 */
export const FORM_VALIDATE_ON: FormInputEvents[] = ['input', 'change'];

/**
 * RequiredTag is the tag the server reports for a missing value. Conditional rules report their own tag,
 * such as required_if, so the client shows the same message as the server.
 */
export type RequiredTag = 'required' | 'required_if' | 'required_unless' | 'required_with';

/** FieldRules are the checks the client runs for one field before it sends a form. */
export type FieldRules = {
  // The tag to report while the field is empty. Leave it out for an optional field.
  required?: RequiredTag;
  // The maximum number of characters.
  max?: number;
  // The maximum number of characters of each entry of a list of texts.
  entryMax?: number;
};

/** FormRules maps the name of each checked field to its rules. */
export type FormRules<T> = Partial<Record<Paths<T>, FieldRules>>;

/**
 * isEmpty reports whether a value counts as missing. It follows the zero values the server rejects
 * for a required field, so a value the client accepts is not rejected by the server for being empty.
 */
function isEmpty(value: unknown): boolean {
  if (value === undefined || value === null) return true;
  if (typeof value === 'string') return value === '';
  if (typeof value === 'number') return value === 0;
  if (typeof value === 'boolean') return !value;
  if (Array.isArray(value)) return value.length === 0;
  return false;
}

/** characterCount counts code points like the server, so an emoji counts as one character. */
export function characterCount(value: string): number {
  return Array.from(value).length;
}

/** countLabel shows how many of the allowed characters a text uses, as the hint of a field. */
export function countLabel(value: string, max: number): string {
  return `${characterCount(value)}/${max}`;
}

/**
 * checkFields runs the rules against state and reports each failed field in the format
 * of the server's validation errors. Only the first failed rule of a field is reported, like the server does.
 * A failed entry of a list is reported by its index, such as `origins[2]`, also like the server.
 */
export function checkFields<T extends object>(state: T, rules: FormRules<T>): FieldErrors {
  const errors: FieldErrors = {};

  for (const [name, rule] of Object.entries(rules) as [string, FieldRules | undefined][]) {
    if (!rule) continue;
    const value = getAtPath(state, name);

    if (rule.required && isEmpty(value)) {
      errors[name] = [
        {
          tag: rule.required,
        },
      ];
      continue;
    }
    if (rule.max !== undefined && typeof value === 'string' && characterCount(value) > rule.max) {
      errors[name] = [
        {
          tag: 'max',
          param: String(rule.max),
        },
      ];
      continue;
    }
    if (rule.entryMax !== undefined && Array.isArray(value)) {
      checkEntries(errors, name, value, rule.entryMax);
    }
  }

  return errors;
}

/** checkEntries reports every text entry of a list that is longer than max. */
function checkEntries(errors: FieldErrors, name: string, entries: unknown[], max: number): void {
  entries.forEach((entry, index) => {
    if (typeof entry !== 'string' || characterCount(entry) <= max) return;
    errors[`${name}[${index}]`] = [
      {
        tag: 'max',
        param: String(max),
      },
    ];
  });
}
