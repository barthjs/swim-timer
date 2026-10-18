import { afterEach, describe, expect, it } from 'vitest';

import { useConfirm } from '@/composables/ui/useConfirm';

describe('useConfirm', () => {
  afterEach(() => {
    useConfirm().settle(false);
  });

  describe('confirm', () => {
    it('opens the dialog with the given options', () => {
      const { state, confirm } = useConfirm();

      void confirm({
        title: 'Delete?',
        color: 'error',
      });

      expect(state.value).toEqual({
        title: 'Delete?',
        color: 'error',
        open: true,
      });
    });

    it('shares one dialog between all callers', () => {
      const first = useConfirm();
      const second = useConfirm();

      void first.confirm({
        title: 'Shared',
      });

      expect(second.state.value.title).toBe('Shared');
    });

    it('cancels a pending confirmation when another one opens', async () => {
      const { confirm, settle } = useConfirm();
      const first = confirm({
        title: 'First',
      });

      const second = confirm({
        title: 'Second',
      });

      await expect(first).resolves.toBe(false);
      settle(true);
      await expect(second).resolves.toBe(true);
    });
  });

  describe('settle', () => {
    it('resolves the confirmation with the answer and closes the dialog', async () => {
      const { state, confirm, settle } = useConfirm();
      const answer = confirm({
        title: 'Delete?',
      });

      settle(true);

      await expect(answer).resolves.toBe(true);
      expect(state.value.open).toBe(false);
    });

    it('ignores a settle without a pending confirmation', () => {
      const { state, settle } = useConfirm();

      const call = () => settle(true);

      expect(call).not.toThrow();
      expect(state.value.open).toBe(false);
    });
  });
});
