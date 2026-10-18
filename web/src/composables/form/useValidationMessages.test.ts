import { describe, expect, it, vi } from 'vitest';

import { useValidationMessages } from '@/composables/form/useValidationMessages';

const messages: Record<string, string> = {
  'fields.password': 'Password',
  'fields.origins': 'Origins',
  'validation.listEntry': '{field}, entry {position}',
  'validation.max_string': '{field} allows at most {param} characters',
  'validation.min': '{field} needs at least {param}',
  'validation.min_string': '{field} needs at least {param} characters',
  'validation.invalid': '{field} is invalid',
};

/** format returns the message of a key with its parameters filled in. */
function format(key: string, params: Record<string, string | number> = {}): string {
  let text = messages[key] ?? key;
  for (const [name, value] of Object.entries(params)) {
    text = text.replace(`{${name}}`, String(value));
  }
  return text;
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: format,
    te: (key: string) => key in messages,
  }),
}));

describe('useValidationMessages', () => {
  describe('mapFieldErrors', () => {
    it('names the field and fills in the rule parameter', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors({
        password: [
          {
            tag: 'min',
            param: '8',
          },
        ],
      });

      expect(errors).toEqual([
        {
          name: 'password',
          message: 'Password needs at least 8',
        },
      ]);
    });

    it('prefers the text for the kind of the checked value', () => {
      const { mapFieldErrors } = useValidationMessages();
      const fields = {
        'account.password': [
          {
            tag: 'min',
            param: '8',
          },
        ],
      };

      const errors = mapFieldErrors(fields, {
        account: {
          password: 'short',
        },
      });

      expect(errors).toEqual([
        {
          name: 'account.password',
          message: 'account.password needs at least 8 characters',
        },
      ]);
    });

    it('falls back to the general text if the kind has none', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors(
        {
          count: [
            {
              tag: 'min',
              param: '8',
            },
          ],
        },
        {
          count: 2,
        },
      );

      expect(errors).toEqual([
        {
          name: 'count',
          message: 'count needs at least 8',
        },
      ]);
    });

    it('shows the error of a list entry at the field of the list and names its position', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors(
        {
          'origins[2]': [
            {
              tag: 'max',
              param: '255',
            },
          ],
        },
        {
          origins: ['a', 'b', 'c'],
        },
      );

      expect(errors).toEqual([
        {
          name: 'origins',
          message: 'Origins, entry 3 allows at most 255 characters',
        },
      ]);
    });

    it('reports only the first failed entry of a list', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors(
        {
          'origins[0]': [
            {
              tag: 'invalid',
            },
          ],
          'origins[4]': [
            {
              tag: 'invalid',
            },
          ],
        },
        {
          origins: ['a', 'b', 'c', 'd', 'e'],
        },
      );

      expect(errors).toEqual([
        {
          name: 'origins',
          message: 'Origins, entry 1 is invalid',
        },
      ]);
    });

    it('uses only the first error of a field and skips fields without errors', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors({
        password: [
          {
            tag: 'invalid',
          },
          {
            tag: 'min',
            param: '8',
          },
        ],
        name: [],
      });

      expect(errors).toEqual([
        {
          name: 'password',
          message: 'Password is invalid',
        },
      ]);
    });

    it('falls back to the raw field key and the generic rule message', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors({
        'items[1].nested.field': [
          {
            tag: 'unknown_rule',
          },
        ],
      });

      expect(errors).toEqual([
        {
          name: 'items[1].nested.field',
          message: 'items[1].nested.field is invalid',
        },
      ]);
    });

    it('keeps the raw name of an indexed field whose base is a map', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors(
        {
          'lanes[0]': [
            {
              tag: 'invalid',
            },
          ],
        },
        {
          lanes: {
            0: 'a',
          },
        },
      );

      expect(errors).toEqual([
        {
          name: 'lanes[0]',
          message: 'lanes[0] is invalid',
        },
      ]);
    });

    it('keeps the raw name of an indexed field without checked values', () => {
      const { mapFieldErrors } = useValidationMessages();

      const errors = mapFieldErrors({
        'origins[0]': [
          {
            tag: 'invalid',
          },
        ],
      });

      expect(errors).toEqual([
        {
          name: 'origins[0]',
          message: 'origins[0] is invalid',
        },
      ]);
    });
  });
});
