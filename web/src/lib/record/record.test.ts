import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import type { RecordResponse } from '@/api/types/record';
import type { RecordTableResponse } from '@/api/types/recordTable';
import {
  RECORD_COLUMNS,
  emptyRecordFilters,
  filterRecords,
  formatRecordDate,
  formatRecordDistance,
  formatRecordTime,
  matchesRecordSearch,
  recordDistanceKey,
  recordDistances,
  recordFacet,
  recordRows,
} from '@/lib/record/record';
import { fakeDistinct } from '@/test/fake';

const [tableId, otherTableId] = fakeDistinct(() => faker.string.alphanumeric(6), 2);

/** Returns a record of a table with the given changes. */
function record(changes: Partial<RecordResponse> = {}): RecordResponse {
  return {
    tableId,
    course: 'LCM',
    gender: 'W',
    relayCount: 1,
    distance: 100,
    stroke: 'FR',
    ageFrom: 0,
    ageTo: 0,
    time: faker.number.int({
      min: 2000,
      max: 30000,
    }),
    approved: faker.datatype.boolean(),
    name: `${faker.person.lastName()}, ${faker.person.firstName()}`,
    club: faker.company.name(),
    place: faker.location.city(),
    date: faker.date.past().toISOString().slice(0, 10),
    birthYear: faker.number.int({
      min: 1950,
      max: 2015,
    }),
    ...changes,
  };
}

/** Returns a record table with the given id. */
function table(id: string): RecordTableResponse {
  return {
    id,
    name: faker.lorem.words(2),
    abbreviation: faker.string.alpha(2).toUpperCase(),
    enabled: true,
    recordCount: faker.number.int({
      min: 1,
      max: 100,
    }),
  };
}

describe('RECORD_COLUMNS', () => {
  it('names every field of a record except its table', () => {
    const { tableId: _, ...fields } = record();

    const columns = [...RECORD_COLUMNS].sort();

    expect(columns).toEqual(Object.keys(fields).sort());
  });
});

describe('emptyRecordFilters', () => {
  it('returns filters that match every record', () => {
    const filters = emptyRecordFilters();

    expect(filterRecords(recordRows([record()], [table(tableId)]), filters)).toHaveLength(1);
  });
});

describe('formatRecordTime', () => {
  it('formats hundredths like the times of a heat', () => {
    const text = formatRecordTime(6234);

    expect(text).toBe('01:02,34');
  });
});

describe('formatRecordDate', () => {
  it('formats a day with its year', () => {
    const text = formatRecordDate('2020-03-15', 'de');

    expect(text).toBe('15.03.2020');
  });

  it('returns a value in another format as it is', () => {
    const text = formatRecordDate('unknown', 'de');

    expect(text).toBe('unknown');
  });

  it('returns nothing for an unknown day', () => {
    const text = formatRecordDate('', 'de');

    expect(text).toBe('');
  });
});

describe('formatRecordDistance', () => {
  it('formats the distance of an individual event', () => {
    const distance = faker.helpers.arrayElement([50, 100, 200]);

    const label = formatRecordDistance({
      relayCount: 1,
      distance,
    });

    expect(label).toBe(`${distance} m`);
  });

  it('formats a relay by its legs', () => {
    const relayCount = faker.number.int({
      min: 2,
      max: 8,
    });
    const distance = faker.helpers.arrayElement([50, 100, 200]);

    const label = formatRecordDistance({
      relayCount,
      distance,
    });

    expect(label).toBe(`${relayCount}×${distance} m`);
  });
});

describe('recordDistanceKey', () => {
  it('tells a relay from an individual event over the length of its leg', () => {
    const distance = faker.helpers.arrayElement([50, 100, 200]);

    const keys = [
      recordDistanceKey({
        relayCount: 1,
        distance,
      }),
      recordDistanceKey({
        relayCount: 4,
        distance,
      }),
    ];

    expect(new Set(keys).size).toBe(2);
  });

  it('tells a relay from an individual event over its total length', () => {
    const keys = [
      recordDistanceKey({
        relayCount: 1,
        distance: 200,
      }),
      recordDistanceKey({
        relayCount: 4,
        distance: 50,
      }),
    ];

    expect(new Set(keys).size).toBe(2);
  });
});

describe('recordRows', () => {
  it('joins the records with their tables under a key per table and position', () => {
    const rows = recordRows(
      [
        record(),
        record({
          tableId: otherTableId,
        }),
        record(),
      ],
      [table(tableId), table(otherTableId)],
    );

    expect(rows.map((row) => row.key)).toEqual([
      `${tableId}-0`,
      `${otherTableId}-0`,
      `${tableId}-1`,
    ]);
    expect(rows[1]?.table?.id).toBe(otherTableId);
  });

  it('leaves the table of an unknown table out', () => {
    const rows = recordRows([record()], []);

    expect(rows[0]?.table).toBeUndefined();
  });
});

describe('matchesRecordSearch', () => {
  it('finds the holder, the club, and the place regardless of case', () => {
    const item = record();

    expect(matchesRecordSearch(item, item.name.toUpperCase())).toBe(true);
    expect(matchesRecordSearch(item, item.club.toLowerCase())).toBe(true);
    expect(matchesRecordSearch(item, item.place.toUpperCase())).toBe(true);
    expect(matchesRecordSearch(item, faker.string.uuid())).toBe(false);
  });

  it('matches every record without a search', () => {
    expect(matchesRecordSearch(record(), '  ')).toBe(true);
  });
});

describe('filterRecords', () => {
  it('keeps the records that match every set filter', () => {
    const rows = recordRows(
      [
        record(),
        record({
          course: 'SCM',
        }),
        record({
          stroke: 'BA',
        }),
        record({
          distance: 200,
        }),
        record({
          gender: 'M',
        }),
      ],
      [table(tableId)],
    );

    const filtered = filterRecords(rows, {
      ...emptyRecordFilters(),
      courses: ['LCM'],
      genders: ['W'],
      strokes: ['FR'],
      distances: [
        recordDistanceKey({
          relayCount: 1,
          distance: 100,
        }),
      ],
    });

    expect(filtered.map((row) => row.key)).toEqual([`${tableId}-0`]);
  });

  it('keeps a relay apart from an individual event over the length of its leg', () => {
    const rows = recordRows(
      [
        record({
          distance: 50,
        }),
        record({
          relayCount: 4,
          distance: 50,
        }),
      ],
      [table(tableId)],
    );

    const filtered = filterRecords(rows, {
      ...emptyRecordFilters(),
      distances: [
        recordDistanceKey({
          relayCount: 4,
          distance: 50,
        }),
      ],
    });

    expect(filtered.map((row) => row.key)).toEqual([`${tableId}-1`]);
  });
});

describe('recordFacet', () => {
  it('returns the distinct values sorted', () => {
    const rows = recordRows(
      [
        record({
          distance: 200,
        }),
        record({
          distance: 50,
        }),
        record({
          distance: 200,
        }),
      ],
      [table(tableId)],
    );

    const distances = recordFacet(rows, (item) => item.distance);

    expect(distances).toEqual([50, 200]);
  });
});

describe('recordDistances', () => {
  it('returns the distinct distances with the individual events first, each sorted by length', () => {
    const rows = recordRows(
      [
        record({
          relayCount: 4,
          distance: 100,
        }),
        record({
          distance: 200,
        }),
        record({
          relayCount: 4,
          distance: 50,
        }),
        record({
          distance: 50,
        }),
        record({
          distance: 200,
        }),
      ],
      [table(tableId)],
    );

    const distances = recordDistances(rows);

    expect(distances).toEqual([
      {
        relayCount: 1,
        distance: 50,
      },
      {
        relayCount: 1,
        distance: 200,
      },
      {
        relayCount: 4,
        distance: 50,
      },
      {
        relayCount: 4,
        distance: 100,
      },
    ]);
  });
});
