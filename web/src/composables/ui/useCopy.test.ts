import type { useToast } from '@nuxt/ui/composables';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useCopy } from '@/composables/ui/useCopy';

const { copyText, toast } = vi.hoisted(() => ({
  copyText: vi.fn<(text: string) => Promise<boolean>>(),
  toast: {
    add: vi.fn<ReturnType<typeof useToast>['add']>(),
  },
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => toast,
}));

vi.mock('@/lib/browser/clipboard', () => ({
  copyText,
}));

describe('useCopy', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('copy', () => {
    it('copies the text and confirms it with the title', async () => {
      copyText.mockResolvedValue(true);
      const text = faker.lorem.sentence();
      const title = faker.lorem.words();

      await useCopy().copy(text, title);

      expect(copyText).toHaveBeenCalledWith(text);
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title,
          color: 'success',
        }),
      );
    });

    it('tells that the copy failed', async () => {
      copyText.mockResolvedValue(false);

      await useCopy().copy(faker.lorem.sentence(), faker.lorem.words());

      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'common.copyFailed',
          color: 'error',
        }),
      );
    });
  });
});
