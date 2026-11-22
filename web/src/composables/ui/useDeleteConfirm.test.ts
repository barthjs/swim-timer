import { beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import type { ConfirmOptions } from '@/composables/ui/useConfirm';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';

const { confirm } = vi.hoisted(() => ({
  confirm: vi.fn<(options: ConfirmOptions) => Promise<boolean>>(),
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

describe('useDeleteConfirm', () => {
  beforeEach(() => {
    confirm.mockReset();
  });

  describe('confirmDelete', () => {
    it('asks with the label, the color, and the icon of a deletion', async () => {
      const title = faker.lorem.sentence();
      const description = faker.lorem.sentence();
      confirm.mockResolvedValue(true);

      const confirmed = await useDeleteConfirm().confirmDelete({
        title,
        description,
      });

      expect(confirmed).toBe(true);
      expect(confirm).toHaveBeenCalledWith({
        title,
        description,
        confirmLabel: 'common.delete',
        color: 'error',
        icon: 'i-lucide-trash-2',
      });
    });
  });
});
