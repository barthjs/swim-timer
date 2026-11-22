import { effectScope, nextTick, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import {
  RECORD_DISPLAY_MS,
  useRecordAnnouncements,
} from '@/composables/record/useRecordAnnouncements';
import type { RecordBanner } from '@/lib/record/recordBanner';
import { fakeDistinct } from '@/test/fake';

const [heatKey, nextHeatKey] = fakeDistinct(() => faker.string.alphanumeric(5), 2);
const [first, second, third] = fakeDistinct(() => faker.string.alphanumeric(8), 3);

/** Returns a record of a lane of the given heat under the given key. */
function banner(key: string, heat = heatKey): RecordBanner {
  return {
    key,
    heat,
    lane: {
      number: 1,
      firstName: faker.person.firstName(),
      lastName: faker.person.lastName(),
      birthYear: String(
        faker.number.int({
          min: 1950,
          max: 2015,
        }),
      ),
      club: faker.company.name(),
      lap: 4,
      meter: 200,
      time: faker.number.int({
        min: 20000,
        max: 300000,
      }),
      rank: 1,
      finished: true,
    },
    record: {
      abbreviation: faker.string.alpha(2).toUpperCase(),
      tableName: faker.lorem.words(2),
      ageFrom: 0,
      ageTo: 0,
      equalled: false,
      previousTime: faker.number.int({
        min: 20000,
        max: 300000,
      }),
      previousName: `${faker.person.lastName()}, ${faker.person.firstName()}`,
    },
  };
}

describe('useRecordAnnouncements', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the records one after another for their duration', async () => {
    const heat = ref(heatKey);
    const list = ref([banner(first), banner(second)]);
    const scope = effectScope();
    const { current } = scope.run(() => useRecordAnnouncements(list, heat))!;

    const shownFirst = current.value?.key;
    vi.advanceTimersByTime(RECORD_DISPLAY_MS);
    const shownSecond = current.value?.key;
    vi.advanceTimersByTime(RECORD_DISPLAY_MS);
    await nextTick();

    expect(shownFirst).toBe(first);
    expect(shownSecond).toBe(second);
    expect(current.value).toBeNull();
    scope.stop();
  });

  it('keeps a record on screen while the list changes, such as with the next heat', async () => {
    const heat = ref(heatKey);
    const list = ref([banner(first)]);
    const scope = effectScope();
    const { current } = scope.run(() => useRecordAnnouncements(list, heat))!;

    list.value = [];
    await nextTick();
    vi.advanceTimersByTime(RECORD_DISPLAY_MS - 1);

    expect(current.value?.key).toBe(first);
    scope.stop();
  });

  it('keeps the waiting records of a heat while the next heat is shown', async () => {
    const heat = ref(heatKey);
    const list = ref([banner(first), banner(second)]);
    const scope = effectScope();
    const { current } = scope.run(() => useRecordAnnouncements(list, heat))!;

    heat.value = nextHeatKey;
    list.value = [];
    await nextTick();
    vi.advanceTimersByTime(RECORD_DISPLAY_MS);

    expect(current.value?.key).toBe(second);
    scope.stop();
  });

  it('drops a waiting record its heat no longer lists', async () => {
    const heat = ref(heatKey);
    const list = ref([banner(first), banner(second)]);
    const scope = effectScope();
    const { current } = scope.run(() => useRecordAnnouncements(list, heat))!;

    list.value = [banner(first)];
    await nextTick();
    vi.advanceTimersByTime(RECORD_DISPLAY_MS);

    expect(current.value).toBeNull();
    scope.stop();
  });

  it('shows a record only once when its heat is made official again', async () => {
    const heat = ref(heatKey);
    const list = ref([banner(first)]);
    const scope = effectScope();
    const { current } = scope.run(() => useRecordAnnouncements(list, heat))!;
    vi.advanceTimersByTime(RECORD_DISPLAY_MS);

    list.value = [banner(first), banner(third)];
    await nextTick();

    expect(current.value?.key).toBe(third);
    scope.stop();
  });
});
