import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { HeatStatus, type Lane, type LaneRecord, type LiveHeat } from '@/api/types/live';
import { recordBanners, recordHeat } from '@/lib/record/recordBanner';
import { fakeDistinct } from '@/test/fake';

const [meetRecord, clubRecord] = fakeDistinct(() => faker.string.alpha(2).toUpperCase(), 2);
const event = faker.number.int({
  min: 1,
  max: 60,
});
const heatNumber = faker.number.int({
  min: 1,
  max: 10,
});

/** Returns a record of a table with the given abbreviation, broken or equalled. */
function laneRecord(abbreviation: string, equalled = false): LaneRecord {
  return {
    abbreviation,
    tableName: faker.lorem.words(2),
    ageFrom: 0,
    ageTo: 0,
    equalled,
    previousTime: faker.number.int({
      min: 20000,
      max: 300000,
    }),
    previousName: `${faker.person.lastName()}, ${faker.person.firstName()}`,
  };
}

/** Returns a finished lane with a rank and the given records. */
function lane(number: number, rank: number, records?: LaneRecord[]): Lane {
  return {
    number,
    firstName: faker.person.firstName(),
    lastName: faker.person.lastName(),
    birthYear: String(
      faker.number.int({
        min: 1950,
        max: 2015,
      }),
    ),
    club: faker.company.name(),
    lap: 2,
    meter: 100,
    time: faker.number.int({
      min: 20000,
      max: 300000,
    }),
    rank,
    finished: true,
    records,
  };
}

/** Returns a heat with the given status and lanes. */
function heat(status: LiveHeat['status'], lanes: Lane[]): LiveHeat {
  return {
    event,
    phase: 0,
    phaseName: '',
    heat: heatNumber,
    heatCount: heatNumber,
    lapCount: 2,
    name: faker.lorem.words(3),
    status,
    firstLane: 1,
    lanes: Object.fromEntries(lanes.map((item) => [String(item.number), item])),
  };
}

describe('recordBanners', () => {
  it('returns the lanes with a record by their rank', () => {
    const banners = recordBanners(
      heat(HeatStatus.Official, [
        lane(5, 2, [laneRecord(clubRecord)]),
        lane(3, 1, [laneRecord(meetRecord), laneRecord(clubRecord)]),
        lane(4, 3),
      ]),
    );

    expect(banners.map((banner) => banner.key)).toEqual([
      `${event}-${heatNumber}-3`,
      `${event}-${heatNumber}-5`,
    ]);
  });

  it('shows the first broken record of a lane before an equalled one', () => {
    const banners = recordBanners(
      heat(HeatStatus.Official, [
        lane(3, 1, [laneRecord(meetRecord, true), laneRecord(clubRecord)]),
      ]),
    );

    expect(banners.map((banner) => banner.record.abbreviation)).toEqual([clubRecord]);
  });

  it('returns the records of a running heat', () => {
    const banners = recordBanners(heat(HeatStatus.Start, [lane(3, 1, [laneRecord(meetRecord)])]));

    expect(banners).toHaveLength(1);
  });

  it('returns nothing without a heat', () => {
    expect(recordBanners(null)).toEqual([]);
  });
});

describe('recordHeat', () => {
  it('identifies the heat by its event and number', () => {
    const id = recordHeat({
      event,
      heat: heatNumber,
    });

    expect(id).toBe(`${event}-${heatNumber}`);
  });

  it('returns nothing without a heat', () => {
    expect(recordHeat(null)).toBe('');
  });
});
