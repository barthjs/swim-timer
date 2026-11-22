import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { chosenRecordTableId } from '@/lib/record/recordTable';
import { fakeDistinct } from '@/test/fake';

describe('chosenRecordTableId', () => {
  const [id, otherId, unknownId] = fakeDistinct(() => faker.string.alphanumeric(6), 3);
  const tables = [
    {
      id,
    },
    {
      id: otherId,
    },
  ];

  it('returns the id of a listed table', () => {
    const chosen = chosenRecordTableId(otherId, tables);

    expect(chosen).toBe(otherId);
  });

  it('returns empty for every table', () => {
    const chosen = chosenRecordTableId('', tables);

    expect(chosen).toBe('');
  });

  it('falls back to every table for an unknown table', () => {
    const chosen = chosenRecordTableId(unknownId, tables);

    expect(chosen).toBe('');
  });

  it('falls back to every table while no table is listed', () => {
    const chosen = chosenRecordTableId(id, []);

    expect(chosen).toBe('');
  });
});
