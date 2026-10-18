import { nextTick } from 'vue';
import { afterEach, describe, expect, it } from 'vitest';

import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { withSetup } from '@/test/withSetup';

describe('useTableColumnState', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('starts with an unchanged layout', () => {
    const { result: state } = withSetup(() => useTableColumnState('users'));

    expect(state.value).toEqual({
      order: [],
      hidden: [],
      sizing: {},
    });
  });

  it('keeps the layout per table in the browser', async () => {
    const { result: state } = withSetup(() => useTableColumnState('users'));
    state.value = {
      order: ['email', 'name'],
      hidden: ['role'],
      sizing: {
        name: 240,
      },
    };
    await nextTick();

    const { result: restored } = withSetup(() => useTableColumnState('users'));
    const { result: other } = withSetup(() => useTableColumnState('stickers'));

    expect(restored.value).toEqual(state.value);
    expect(other.value.hidden).toEqual([]);
  });
});
