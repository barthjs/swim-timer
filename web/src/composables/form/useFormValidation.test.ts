import { describe, expect, it, vi } from 'vitest';

import { useFormValidation } from '@/composables/form/useFormValidation';

const messages: Record<string, string> = {
  'fields.name': 'Name',
  'fields.contact.email': 'Email',
  'validation.required_unless': '{field} is required',
  'validation.max': '{field} must be at most {param}',
  'validation.max_string': '{field} must be at most {param} characters',
};

/** format returns the message of a key with its parameters filled in. */
function format(key: string, params: Record<string, string> = {}): string {
  let text = messages[key] ?? key;
  for (const [name, value] of Object.entries(params)) {
    text = text.replace(`{${name}}`, value);
  }
  return text;
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: format,
    te: (key: string) => key in messages,
  }),
}));

type State = { name: string; contact: { method: string; email: string } };

describe('useFormValidation', () => {
  const { validate } = useFormValidation<State>((state) => ({
    name: {
      max: 5,
    },
    'contact.email': {
      required: state.contact.method !== 'none' ? 'required_unless' : undefined,
    },
  }));

  describe('validate', () => {
    it('returns the failed rules with the messages of the server errors', () => {
      const state: State = {
        name: 'Example',
        contact: {
          method: 'email',
          email: '',
        },
      };

      const errors = validate(state);

      expect(errors).toEqual([
        {
          name: 'name',
          message: 'Name must be at most 5 characters',
        },
        {
          name: 'contact.email',
          message: 'Email is required',
        },
      ]);
    });

    it('evaluates conditional rules against the current state', () => {
      const state: State = {
        name: 'abc',
        contact: {
          method: 'none',
          email: '',
        },
      };

      const errors = validate(state);

      expect(errors).toEqual([]);
    });
  });
});
