import { nextTick } from 'vue';
import { afterEach, describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { fakeDistinct } from '@/test/fake';
import { withSetup } from '@/test/withSetup';

describe('useTableColumnState', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('starts with an unchanged layout', () => {
    const { result: state } = withSetup(() => useTableColumnState(faker.lorem.slug()));

    expect(state.value).toEqual({
      order: [],
      hidden: [],
      shown: [],
      sizing: {},
    });
  });

  it('keeps the layout per table in the browser', async () => {
    const [table, otherTable] = fakeDistinct(() => faker.lorem.slug(), 2);
    const [first, second, third] = fakeDistinct(() => faker.database.column(), 3);
    const { result: state } = withSetup(() => useTableColumnState(table));
    state.value = {
      order: [first, second],
      hidden: [third],
      shown: [],
      sizing: {
        [first]: faker.number.int({
          min: 80,
          max: 400,
        }),
      },
    };
    await nextTick();

    const { result: restored } = withSetup(() => useTableColumnState(table));
    const { result: other } = withSetup(() => useTableColumnState(otherTable));

    expect(restored.value).toEqual(state.value);
    expect(other.value.hidden).toEqual([]);
  });
});
