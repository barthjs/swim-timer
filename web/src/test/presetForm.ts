import { vi } from 'vitest';

import { withSetup } from '@/test/withSetup';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import type { PresetForm } from '@/composables/usePresetForm';
import { providePresetForm } from '@/composables/usePresetForm';
import { createPreset, toEditorPreset } from '@/lib/lanes';

export { presetFormMocks, resetPresetFormMocks } from '@/test/presetFormMocks';

/**
 * mountPresetForm creates a preset form inside a mounted component, optionally with an opened preset
 * of the given number of lanes, and runs extra composables in the same component. The lanes of the preset
 * span x 480 to 1440 of the 1920 x 1080 reference resolution, lane n y 100 + 60n to 150 + 60n.
 */
export function mountPresetForm<T>(
  setup: (presetForm: PresetForm) => T,
  laneCount = 0,
): { presetForm: PresetForm; result: T; unmount: () => void } {
  const form = {
    setErrors: vi.fn<FormErrorTarget['setErrors']>(),
    clear: vi.fn<() => void>(),
  };
  let presetForm: PresetForm | undefined;

  const { result, unmount } = withSetup(() => {
    presetForm = providePresetForm(form);
    if (laneCount > 0) {
      Object.assign(presetForm.preset, toEditorPreset(createPreset('test', laneCount)));
      presetForm.loaded.value = true;
      presetForm.savedName.value = 'test';
      presetForm.dirtyForm.capture();
      presetForm.editor.history.reset();
    }
    return setup(presetForm);
  });

  return {
    presetForm: presetForm as PresetForm,
    result,
    unmount,
  };
}
