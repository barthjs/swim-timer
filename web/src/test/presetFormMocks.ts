import type { useToast } from '@nuxt/ui/composables';
import { vi } from 'vitest';

import type { presetRepository } from '@/api/repositories/preset';
import type { downloadJSON } from '@/lib/browser/download';
import type { KeyValueStore } from '@/lib/browser/keyValueStore';
import type { ConfirmOptions } from '@/composables/ui/useConfirm';

/**
 * The mocks the preset form needs, which each test file registers with vi.mock and a factory importing this module,
 * importing only types of the app, so the factories cannot form an import cycle.
 */
export const presetFormMocks = {
  toast: {
    add: vi.fn<ReturnType<typeof useToast>['add']>(),
  },
  confirm: vi.fn<(options: ConfirmOptions) => Promise<boolean>>(async () => true),
  presets: {
    list: vi.fn<typeof presetRepository.list>(async () => []),
    get: vi.fn<typeof presetRepository.get>(),
    save: vi.fn<typeof presetRepository.save>(),
    delete: vi.fn<typeof presetRepository.delete>(async () => {}),
  },
  download: vi.fn<typeof downloadJSON>(),
  images: {
    get: vi.fn<KeyValueStore<Blob>['get']>(async () => null),
    set: vi.fn<KeyValueStore<Blob>['set']>(async () => {}),
    delete: vi.fn<KeyValueStore<Blob>['delete']>(async () => {}),
  },
};

/** Restores the default behavior of every preset form mock. */
export function resetPresetFormMocks(): void {
  presetFormMocks.toast.add.mockReset();
  presetFormMocks.confirm.mockReset().mockResolvedValue(true);
  presetFormMocks.presets.list.mockReset().mockResolvedValue([]);
  presetFormMocks.presets.get.mockReset();
  presetFormMocks.presets.save.mockReset();
  presetFormMocks.presets.delete.mockReset().mockResolvedValue(undefined);
  presetFormMocks.download.mockReset();
  presetFormMocks.images.get.mockReset().mockResolvedValue(null);
  presetFormMocks.images.set.mockReset().mockResolvedValue(undefined);
  presetFormMocks.images.delete.mockReset().mockResolvedValue(undefined);
}
