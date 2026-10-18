import { ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useDiscardConfirm, useDiscardGuard } from '@/composables/form/useDiscardConfirm';

const { confirm } = vi.hoisted(() => ({
  confirm: vi.fn<() => Promise<boolean>>(),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('@/composables/ui/useConfirm', () => ({
  useConfirm: () => ({
    confirm,
  }),
}));

describe('useDiscardGuard', () => {
  beforeEach(() => {
    confirm.mockReset();
  });

  describe('confirmDiscard', () => {
    it('allows dropping clean changes without asking', async () => {
      const { confirmDiscard } = useDiscardGuard(ref(false));

      const allowed = await confirmDiscard();

      expect(allowed).toBe(true);
      expect(confirm).not.toHaveBeenCalled();
    });

    it('skips the question when told to', async () => {
      const { confirmDiscard } = useDiscardGuard(ref(true), ref(true));

      const allowed = await confirmDiscard();

      expect(allowed).toBe(true);
      expect(confirm).not.toHaveBeenCalled();
    });

    it('allows dropping unsaved changes once discarding is confirmed', async () => {
      confirm.mockResolvedValue(true);
      const { confirmDiscard } = useDiscardGuard(ref(true));

      const allowed = await confirmDiscard();

      expect(allowed).toBe(true);
    });

    it('keeps unsaved changes when discarding is declined', async () => {
      confirm.mockResolvedValue(false);
      const { confirmDiscard } = useDiscardGuard(ref(true));

      const allowed = await confirmDiscard();

      expect(allowed).toBe(false);
      expect(confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'common.discardTitle',
          confirmLabel: 'common.discardConfirm',
          color: 'error',
        }),
      );
    });
  });
});

describe('useDiscardConfirm', () => {
  beforeEach(() => {
    confirm.mockReset();
  });

  describe('attemptClose', () => {
    it('closes a clean form without asking', async () => {
      const open = ref(true);
      const { attemptClose } = useDiscardConfirm(open, ref(false));

      await attemptClose();

      expect(open.value).toBe(false);
      expect(confirm).not.toHaveBeenCalled();
    });

    it('skips the question when told to', async () => {
      const open = ref(true);
      const { attemptClose } = useDiscardConfirm(open, ref(true), ref(true));

      await attemptClose();

      expect(open.value).toBe(false);
      expect(confirm).not.toHaveBeenCalled();
    });

    it('keeps a dirty form open when discarding is declined', async () => {
      confirm.mockResolvedValue(false);
      const open = ref(true);
      const { attemptClose } = useDiscardConfirm(open, ref(true));

      await attemptClose();

      expect(open.value).toBe(true);
      expect(confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'common.discardTitle',
          color: 'error',
        }),
      );
    });
  });

  describe('openProxy', () => {
    it('opens the modal', () => {
      const open = ref(false);
      const { openProxy } = useDiscardConfirm(open, ref(false));

      openProxy.value = true;

      expect(open.value).toBe(true);
    });

    it('closes a dirty form once discarding is confirmed', async () => {
      confirm.mockResolvedValue(true);
      const open = ref(true);
      const { openProxy } = useDiscardConfirm(open, ref(true));

      openProxy.value = false;

      await vi.waitFor(() => expect(open.value).toBe(false));
    });
  });
});
