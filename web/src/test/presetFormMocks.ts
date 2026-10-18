import type { useToast } from '@nuxt/ui/composables';
import { vi } from 'vitest';

import type { presetRepository } from '@/api/repositories/preset';
import type { downloadJSON } from '@/lib/download';
import type { KeyValueStore } from '@/lib/keyValueStore';
import type { ConfirmOptions } from '@/composables/ui/useConfirm';

/**
 * The mocks the preset form needs, shared by the tests of the editor composables. Each test file registers them
 * with vi.mock and factories importing this module, since vi.mock only works in the test file.
 * It imports only types of the app, so the mock factories cannot form an import cycle.
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

/** resetPresetFormMocks restores the default behavior of every preset form mock. */
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
