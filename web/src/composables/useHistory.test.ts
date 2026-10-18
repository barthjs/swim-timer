import { nextTick, reactive } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useHistory } from '@/composables/useHistory';

/** setup creates a history over a nested target and starts it at the current state. */
function setup(maxHistory?: number, paused = () => false) {
  const target = reactive({
    count: 0,
    nested: {
      label: 'a',
    },
  });
  const history = useHistory(target, {
    maxHistory,
    paused,
  });
  history.reset();
  return {
    target,
    history,
  };
}

describe('useHistory', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  describe('reset', () => {
    it('starts without undo or redo', () => {
      const { history } = setup();

      const steps = [history.canUndo.value, history.canRedo.value];

      expect(steps).toEqual([false, false]);
    });
  });

  describe('commit', () => {
    it('skips commits without a change', () => {
      const { history } = setup();

      history.commit();

      expect(history.canUndo.value).toBe(false);
    });

    it('drops the redo branch when a new change is committed', () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      history.undo();
      target.count = 2;

      history.commit();

      expect(history.canRedo.value).toBe(false);
    });

    it('keeps at most the configured number of entries', () => {
      const { target, history } = setup(3);

      for (let count = 1; count <= 5; count++) {
        target.count = count;
        history.commit();
      }

      while (history.canUndo.value) {
        history.undo();
      }

      expect(target.count).toBe(3);
    });

    it('does not share state between the target and the history entries', () => {
      const { target, history } = setup();
      target.nested.label = 'b';
      history.commit();
      history.undo();
      target.nested.label = 'c';

      history.redo();

      expect(target.nested.label).toBe('b');
    });
  });

  describe('undo', () => {
    it('restores the previous committed state in place', () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      target.nested.label = 'b';
      history.commit();

      history.undo();

      expect(target).toEqual({
        count: 1,
        nested: {
          label: 'a',
        },
      });
    });

    it('stops at the start of the history', () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      target.nested.label = 'b';
      history.commit();

      history.undo();
      history.undo();

      expect(target).toEqual({
        count: 0,
        nested: {
          label: 'a',
        },
      });
      expect(history.canUndo.value).toBe(false);
    });

    it('records a pending change before it undoes', async () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      target.count = 2;
      await nextTick();

      history.undo();
      history.redo();

      expect(target.count).toBe(2);
    });
  });

  describe('redo', () => {
    it('restores the undone changes in place', () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      target.nested.label = 'b';
      history.commit();
      history.undo();
      history.undo();

      history.redo();
      history.redo();

      expect(target).toEqual({
        count: 1,
        nested: {
          label: 'b',
        },
      });
      expect(history.canRedo.value).toBe(false);
    });
  });

  describe('automatic checkpoint', () => {
    it('records a change once it rests', async () => {
      const { target, history } = setup();
      target.count = 1;
      await nextTick();
      target.count = 2;
      await nextTick();

      vi.runAllTimers();

      expect(history.canUndo.value).toBe(true);
    });

    it('waits while the history is paused', async () => {
      const { target, history } = setup(undefined, () => true);
      target.count = 1;
      await nextTick();

      vi.runAllTimers();

      expect(history.canUndo.value).toBe(false);
    });

    it('adds no entry for a restored snapshot', async () => {
      const { target, history } = setup();
      target.count = 1;
      history.commit();
      history.undo();
      await nextTick();

      vi.runAllTimers();

      expect(history.canRedo.value).toBe(true);
    });
  });
});
