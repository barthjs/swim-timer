import { describe, expect, it } from 'vitest';

import { checkFields, countLabel, FORM_VALIDATE_ON } from '@/lib/form/validation';

type State = {
  name: string;
  width: number | null;
  nested: { field: string };
  tags: string[];
};

/** state returns a valid state with the given fields replaced. */
function state(overrides: Partial<State> = {}): State {
  return {
    name: 'Name',
    width: 1920,
    nested: {
      field: 'value',
    },
    tags: [],
    ...overrides,
  };
}

describe('FORM_VALIDATE_ON', () => {
  it('leaves out the blur, so focusing and leaving a field shows no error', () => {
    const events = FORM_VALIDATE_ON;

    expect(events).toEqual(['input', 'change']);
  });
});

describe('countLabel', () => {
  it('shows the used and the allowed characters', () => {
    const label = countLabel('Name 😀', 100);

    expect(label).toBe('6/100');
  });
});

describe('checkFields', () => {
  it('accepts a state that passes every rule', () => {
    const errors = checkFields(state(), {
      name: {
        required: 'required',
        max: 10,
      },
      width: {
        required: 'required',
      },
    });

    expect(errors).toEqual({});
  });

  it('counts characters instead of UTF-16 code units', () => {
    const errors = checkFields(
      state({
        name: '😀😀😀',
      }),
      {
        name: {
          max: 3,
        },
      },
    );

    expect(errors).toEqual({});
  });

  it('skips the maximum for an optional empty field', () => {
    const errors = checkFields(
      state({
        name: '',
      }),
      {
        name: {
          max: 10,
        },
      },
    );

    expect(errors).toEqual({});
  });

  it.each([
    {
      name: 'an empty string',
      overrides: {
        name: '',
      },
      field: 'name',
    },
    {
      name: 'a zero',
      overrides: {
        width: 0,
      },
      field: 'width',
    },
    {
      name: 'a cleared number',
      overrides: {
        width: null,
      },
      field: 'width',
    },
  ])('reports $name as missing', ({ overrides, field }) => {
    const errors = checkFields(state(overrides), {
      name: {
        required: 'required',
      },
      width: {
        required: 'required',
      },
    });

    expect(errors).toEqual({
      [field]: [
        {
          tag: 'required',
        },
      ],
    });
  });

  it('reports the tag of a conditional rule', () => {
    const errors = checkFields(
      state({
        nested: {
          field: '',
        },
      }),
      {
        'nested.field': {
          required: 'required_unless',
        },
      },
    );

    expect(errors).toEqual({
      'nested.field': [
        {
          tag: 'required_unless',
        },
      ],
    });
  });

  it('reports a text over the maximum with the limit as parameter', () => {
    const errors = checkFields(
      state({
        name: 'Example name',
      }),
      {
        name: {
          max: 10,
        },
      },
    );

    expect(errors).toEqual({
      name: [
        {
          tag: 'max',
          param: '10',
        },
      ],
    });
  });

  it('reports only the missing value of an empty required field', () => {
    const errors = checkFields(
      state({
        name: '',
      }),
      {
        name: {
          required: 'required',
          max: 0,
        },
      },
    );

    expect(errors).toEqual({
      name: [
        {
          tag: 'required',
        },
      ],
    });
  });

  it('reports each list entry over the maximum by its index', () => {
    const errors = checkFields(
      state({
        tags: ['short', 'much too long', 'ok', 'also too long'],
      }),
      {
        tags: {
          entryMax: 5,
        },
      },
    );

    expect(errors).toEqual({
      'tags[1]': [
        {
          tag: 'max',
          param: '5',
        },
      ],
      'tags[3]': [
        {
          tag: 'max',
          param: '5',
        },
      ],
    });
  });
});
