import { nextTick, reactive, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useDraftRecovery } from '@/composables/editor/useDraftRecovery';
import { withSetup } from '@/test/withSetup';

const KEY = 'draft-test.draft.first';

describe('useDraftRecovery', () => {
  let unmount = () => {};

  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });
  afterEach(() => {
    unmount();
    vi.useRealTimers();
  });

  /** mount creates the recovery of a saved document with a count and a confirmation the test answers. */
  function mount(answer = true) {
    const state = reactive({
      count: 0,
    });
    const dirty = ref(false);
    const confirm = vi.fn<() => Promise<boolean>>(async () => answer);
    const mounted = withSetup(() =>
      useDraftRecovery({
        storageKey: 'draft-test',
        document: () => 'first',
        state,
        dirty,
        confirm,
      }),
    );
    unmount = mounted.unmount;
    return {
      recovery: mounted.result,
      state,
      dirty,
      confirm,
    };
  }

  describe('offer', () => {
    it('restores a kept draft after asking', async () => {
      const { recovery, state } = mount();
      localStorage.setItem(
        KEY,
        JSON.stringify({
          count: 3,
        }),
      );

      const restored = await recovery.offer();

      expect(restored).toBe(true);
      expect(state.count).toBe(3);
    });

    it('deletes a declined draft', async () => {
      const { recovery, state } = mount(false);
      localStorage.setItem(
        KEY,
        JSON.stringify({
          count: 3,
        }),
      );

      await recovery.offer();

      expect(state.count).toBe(0);
      expect(localStorage.getItem(KEY)).toBeNull();
    });

    it('asks nothing without a draft that differs', async () => {
      const { recovery, confirm } = mount();
      localStorage.setItem(
        KEY,
        JSON.stringify({
          count: 0,
        }),
      );

      await recovery.offer();

      expect(confirm).not.toHaveBeenCalled();
    });
  });

  describe('discard', () => {
    it('deletes the draft of a document', () => {
      const { recovery } = mount();
      localStorage.setItem(KEY, '{}');

      recovery.discard('first');

      expect(localStorage.getItem(KEY)).toBeNull();
    });
  });

  describe('saving', () => {
    it('keeps unsaved changes once they rest', async () => {
      const { state, dirty } = mount();
      state.count = 2;
      dirty.value = true;
      await nextTick();

      vi.runAllTimers();

      expect(JSON.parse(localStorage.getItem(KEY)!)).toEqual({
        count: 2,
      });
    });

    it('deletes the draft once the changes are saved or discarded', async () => {
      const { dirty } = mount();
      dirty.value = true;
      await nextTick();
      localStorage.setItem(KEY, '{}');

      dirty.value = false;
      await nextTick();

      expect(localStorage.getItem(KEY)).toBeNull();
    });
  });
});
