import { afterEach, describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { useConfirm } from '@/composables/ui/useConfirm';

describe('useConfirm', () => {
  afterEach(() => {
    useConfirm().settle(false);
  });

  describe('confirm', () => {
    it('opens the dialog with the given options', () => {
      const title = faker.lorem.sentence();
      const { state, confirm } = useConfirm();

      void confirm({
        title,
        color: 'error',
      });

      expect(state.value).toEqual({
        title,
        color: 'error',
        open: true,
      });
    });

    it('shares one dialog between all callers', () => {
      const title = faker.lorem.sentence();
      const first = useConfirm();
      const second = useConfirm();

      void first.confirm({
        title,
      });

      expect(second.state.value.title).toBe(title);
    });

    it('cancels a pending confirmation when another one opens', async () => {
      const { confirm, settle } = useConfirm();
      const first = confirm({
        title: faker.lorem.sentence(),
      });

      const second = confirm({
        title: faker.lorem.sentence(),
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
        title: faker.lorem.sentence(),
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
