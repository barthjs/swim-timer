import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import type { ImportColumnResponse, ImportMapping } from '@/api/types/import';
import {
  cleanImportMapping,
  emptyTableFormat,
  IMPORT_UNSET,
  importTargets,
  importValueSources,
  splitImportErrors,
} from '@/lib/import/mapping';
import { fakeDistinct } from '@/test/fake';

// The keys of the columns of a target and the headers of a file, and values of a file and of a column
const [kind, name, time] = fakeDistinct(() => faker.database.column(), 3);
const [kindHeader, nameHeader] = fakeDistinct(() => faker.lorem.word(), 2);
const [small, large, medium, other] = fakeDistinct(() => faker.lorem.word(), 4);
const [choice, otherChoice, suggestion] = fakeDistinct(() => faker.lorem.word(), 3);

/** Returns a column of a target with the given choices and suggestions. */
function column(
  key: string,
  choices: string[] = [],
  suggestions: string[] = [],
): ImportColumnResponse {
  return {
    key,
    required: false,
    choices,
    suggestions,
  };
}

/** Returns a mapping with the given columns and values and a detected format. */
function mapping(
  columns: Record<string, string>,
  values: Record<string, Record<string, string>> = {},
): ImportMapping {
  return {
    format: emptyTableFormat(),
    columns,
    values,
  };
}

describe('emptyTableFormat', () => {
  it('returns a format that is detected', () => {
    const format = emptyTableFormat();

    expect(format).toEqual({
      encoding: '',
      separator: '',
    });
  });
});

describe('importTargets', () => {
  it('returns the choices of a column', () => {
    const targets = importTargets(column(kind, [choice, otherChoice], [suggestion]));

    expect(targets).toEqual([choice, otherChoice]);
  });

  it('returns the suggestions of a column without choices', () => {
    const targets = importTargets(column(kind, [], [suggestion]));

    expect(targets).toEqual([suggestion]);
  });

  it('returns nothing for a column of free text', () => {
    const targets = importTargets(column(name));

    expect(targets).toEqual([]);
  });
});

describe('importValueSources', () => {
  it('returns the values of the assigned header', () => {
    const sources = importValueSources(
      column(kind, [choice]),
      mapping({
        [kind]: kindHeader,
      }),
      {
        [kindHeader]: [small, large],
      },
    );

    expect(sources).toEqual([small, large]);
  });

  it('returns nothing for a column of free text or without a header', () => {
    const free = importValueSources(
      column(name),
      mapping({
        [name]: nameHeader,
      }),
      {
        [nameHeader]: [small],
      },
    );
    const unassigned = importValueSources(column(kind, [choice]), mapping({}), {
      [kindHeader]: [small],
    });

    expect(free).toEqual([]);
    expect(unassigned).toEqual([]);
  });
});

describe('cleanImportMapping', () => {
  it('leaves out the unassigned columns and the kept or unchanged values', () => {
    const cleaned = cleanImportMapping(
      mapping(
        {
          [kind]: kindHeader,
          [name]: IMPORT_UNSET,
          [time]: '',
        },
        {
          [kind]: {
            [small]: choice,
            [large]: large,
            [medium]: IMPORT_UNSET,
          },
          [name]: {
            [other]: choice,
          },
        },
      ),
    );

    expect(cleaned).toEqual(
      mapping(
        {
          [kind]: kindHeader,
        },
        {
          [kind]: {
            [small]: choice,
          },
        },
      ),
    );
  });
});

describe('splitImportErrors', () => {
  it('sorts the errors by what they belong to', () => {
    const errors = splitImportErrors({
      [name]: [
        {
          tag: 'required',
        },
      ],
      file: [
        {
          tag: 'unreadable',
          param: 'line 2',
        },
      ],
      [`mapping.columns.${time}`]: [
        {
          tag: 'required',
        },
      ],
      [`mapping.values.${kind}`]: [
        {
          tag: 'oneof',
        },
      ],
      [`rows.4.${time}`]: [
        {
          tag: 'date',
        },
      ],
      'rows.2': [
        {
          tag: 'short_row',
        },
      ],
      [`rows.4.${kind}`]: [
        {
          tag: 'number',
        },
      ],
    });

    expect(errors).toEqual({
      form: {
        [name]: [
          {
            tag: 'required',
          },
        ],
      },
      file: [
        {
          tag: 'unreadable',
          param: 'line 2',
        },
      ],
      columns: {
        [time]: [
          {
            tag: 'required',
          },
        ],
      },
      values: {
        [kind]: [
          {
            tag: 'oneof',
          },
        ],
      },
      rows: [
        {
          line: 2,
          fields: {
            '': [
              {
                tag: 'short_row',
              },
            ],
          },
        },
        {
          line: 4,
          fields: {
            [time]: [
              {
                tag: 'date',
              },
            ],
            [kind]: [
              {
                tag: 'number',
              },
            ],
          },
        },
      ],
    });
  });
});
