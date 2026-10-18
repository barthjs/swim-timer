import { createApp, defineComponent, h, toRaw } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { CornerKey, type PresetResponse } from '@/api/types';
import { withSetup } from '@/test/withSetup';
import {
  DEFAULT_LANE_GAP,
  createPreset,
  laneBounds,
  pixelPoint,
  toEditorPreset,
} from '@/lib/lanes';
import { presetFormMocks, mountPresetForm, resetPresetFormMocks } from '@/test/presetForm';
import { providePresetForm, usePresetForm, type PresetForm } from '@/composables/usePresetForm';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));
vi.mock('@nuxt/ui/composables', async () => {
  const { presetFormMocks } = await import('@/test/presetFormMocks');
  return {
    useToast: () => presetFormMocks.toast,
  };
});
vi.mock('@/composables/ui/useConfirm', async () => {
  const { presetFormMocks } = await import('@/test/presetFormMocks');
  return {
    useConfirm: () => ({
      confirm: presetFormMocks.confirm,
    }),
  };
});
vi.mock('@/api/repositories/preset', async () => ({
  presetRepository: (await import('@/test/presetFormMocks')).presetFormMocks.presets,
}));
vi.mock('@/lib/download', async () => ({
  downloadJSON: (await import('@/test/presetFormMocks')).presetFormMocks.download,
}));
vi.mock('@/lib/backgroundImages', async () => ({
  backgroundImageStore: (await import('@/test/presetFormMocks')).presetFormMocks.images,
}));

/** toEditorPresetFor returns a new preset with two lanes as the editor holds it. */
function toEditorPresetFor(name: string) {
  return toEditorPreset(createPreset(name, 2));
}

/** flush lets pending promise callbacks run. */
async function flush(): Promise<void> {
  for (let i = 0; i < 5; i++) await Promise.resolve();
}

/** echoSave lets the mocked repository return every saved preset as stored. */
function echoSave(): void {
  presetFormMocks.presets.save.mockImplementation(
    async (_name: string, preset: PresetResponse): Promise<PresetResponse> =>
      JSON.parse(JSON.stringify(preset)),
  );
}

describe('providePresetForm', () => {
  let unmount = () => {};

  beforeEach(() => {
    localStorage.clear();
    resetPresetFormMocks();
  });
  afterEach(() => unmount());

  /** mount creates the editor with a preset of the given lane count, or without a preset. */
  function mount(laneCount = 0) {
    const mounted = mountPresetForm((presetForm) => presetForm, laneCount);
    unmount = mounted.unmount;
    return mounted.presetForm;
  }

  describe('loadPresetNames', () => {
    it('stores the preset names', async () => {
      const presetForm = mount();
      presetFormMocks.presets.list.mockResolvedValueOnce(['a', 'b']);

      await presetForm.loadPresetNames();

      expect(presetForm.presetNames.value).toEqual(['a', 'b']);
    });

    it('reports a failure', async () => {
      const presetForm = mount();
      presetFormMocks.presets.list.mockRejectedValueOnce(new Error('offline'));

      await presetForm.loadPresetNames();

      expect(presetFormMocks.toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          color: 'error',
        }),
      );
    });
  });

  describe('selectPreset', () => {
    it('opens a stored preset as the saved state in pixels of its reference resolution', async () => {
      const presetForm = mount();
      const stored: PresetResponse = createPreset('stored', 2);
      presetFormMocks.presets.get.mockResolvedValue(stored);

      await presetForm.selectPreset('stored');

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: 480,
        y: 160,
      });
      expect(presetForm.loaded.value).toBe(true);
      expect(presetForm.savedName.value).toBe('stored');
      expect(presetForm.isDirty.value).toBe(false);
      expect(presetForm.isDraft.value).toBe(false);
      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });

    it('offers the unsaved changes the browser kept for the preset', async () => {
      const presetForm = mount();
      presetFormMocks.presets.get.mockResolvedValue(createPreset('stored', 2));
      localStorage.setItem(
        'editor.draft.stored',
        JSON.stringify({
          ...toEditorPresetFor('stored'),
          comment: 'kept',
        }),
      );

      await presetForm.selectPreset('stored');

      expect(presetForm.preset.comment).toBe('kept');
      expect(presetForm.isDirty.value).toBe(true);
    });

    it('shows every hidden lane of the last preset again', async () => {
      const presetForm = mount(2);
      presetForm.editor.visibility.toggle('1');
      presetFormMocks.presets.get.mockResolvedValue(createPreset('stored', 2));

      await presetForm.selectPreset('stored');

      expect(presetForm.editor.visibility.hidden.value.size).toBe(0);
    });

    it('shows the stored background image of an opened preset', async () => {
      const presetForm = mount();
      const image = new Blob(['image']);
      presetFormMocks.presets.get.mockResolvedValue(createPreset('stored', 2));
      presetFormMocks.images.get.mockResolvedValue(image);

      await presetForm.selectPreset('stored');

      expect(presetFormMocks.images.get).toHaveBeenCalledWith('preset:stored');
      expect(presetForm.editor.backgroundImage.image.value).toBe(image);
    });

    it('keeps an opened preset when it is selected again', async () => {
      const presetForm = mount();
      presetFormMocks.presets.get.mockResolvedValue(createPreset('stored', 2));
      await presetForm.selectPreset('stored');

      await presetForm.selectPreset('stored');

      expect(presetFormMocks.presets.get).toHaveBeenCalledOnce();
    });

    it('opens only the latest selected preset when an earlier answer arrives late', async () => {
      const presetForm = mount();
      let resolveFirst: (preset: PresetResponse) => void = () => {};
      presetFormMocks.presets.get.mockImplementationOnce(
        () =>
          new Promise<PresetResponse>((resolve) => {
            resolveFirst = resolve;
          }),
      );
      presetFormMocks.presets.get.mockResolvedValueOnce(createPreset('second', 3));
      const first = presetForm.selectPreset('first');
      await flush();

      await presetForm.selectPreset('second');
      resolveFirst(createPreset('first', 2));
      await first;

      expect(presetForm.savedName.value).toBe('second');
      expect(Object.keys(presetForm.preset.lanes)).toHaveLength(3);
    });

    it('keeps a new draft when a selected preset arrives late', async () => {
      const presetForm = mount();
      let resolveStored: (preset: PresetResponse) => void = () => {};
      presetFormMocks.presets.get.mockImplementationOnce(
        () =>
          new Promise<PresetResponse>((resolve) => {
            resolveStored = resolve;
          }),
      );
      const selection = presetForm.selectPreset('stored');
      await flush();

      await presetForm.addPreset('draft', 1);
      resolveStored(createPreset('stored', 2));
      await selection;

      expect(presetForm.isDraft.value).toBe(true);
      expect(presetForm.preset.name).toBe('draft');
    });

    it('reports a preset that cannot be loaded', async () => {
      const presetForm = mount();
      presetFormMocks.presets.get.mockRejectedValue(new Error('missing'));

      await presetForm.selectPreset('missing');

      expect(presetForm.loaded.value).toBe(false);
      expect(presetFormMocks.toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.editor.loadError',
        }),
      );
    });

    it('asks before unsaved changes are dropped', async () => {
      const presetForm = mount(2);
      presetForm.preset.lanes['1']![CornerKey.TopLeft].x += 5;
      presetFormMocks.confirm.mockResolvedValueOnce(false);

      await presetForm.selectPreset('other');

      expect(presetFormMocks.confirm).toHaveBeenCalledOnce();
      expect(presetFormMocks.presets.get).not.toHaveBeenCalled();
      expect(presetForm.isDirty.value).toBe(true);
    });
  });

  describe('addPreset', () => {
    it('starts a new draft', async () => {
      const presetForm = mount();

      const added = await presetForm.addPreset('new', 4);

      expect(added).toBe(true);
      expect(presetForm.isDraft.value).toBe(true);
      expect(Object.keys(presetForm.preset.lanes)).toHaveLength(4);
    });

    it('counts every opened preset, also a draft after a draft', async () => {
      const presetForm = mount();
      const before = presetForm.openCount.value;
      await presetForm.addPreset('first', 2);

      await presetForm.addPreset('second', 2);

      expect(presetForm.openCount.value).toBe(before + 2);
    });

    it('rejects an existing name', async () => {
      const presetForm = mount();
      presetForm.presetNames.value = ['taken'];

      const added = await presetForm.addPreset('taken', 4);

      expect(added).toBe(false);
      expect(presetFormMocks.toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.editor.presetExists',
        }),
      );
    });
  });

  describe('importPreset', () => {
    it('imports a preset file as a draft', async () => {
      const presetForm = mount();
      const content = JSON.stringify({
        ...createPreset('', 2),
        name: '',
      });

      await presetForm.importPreset(new File([content], 'preset-a.json'));

      expect(presetForm.isDraft.value).toBe(true);
      expect(presetForm.preset.name).toBe('preset-a');
    });

    it('asks before an import replaces an existing preset', async () => {
      const presetForm = mount();
      presetForm.presetNames.value = ['preset-a'];
      presetFormMocks.confirm.mockResolvedValueOnce(false);

      await presetForm.importPreset(
        new File([JSON.stringify(createPreset('preset-a', 1))], 'preset-a.json'),
      );

      expect(presetForm.loaded.value).toBe(false);
    });

    it('rejects a file that is not a preset', async () => {
      const presetForm = mount();

      await presetForm.importPreset(new File(['not json'], 'file.json'));

      expect(presetForm.loaded.value).toBe(false);
      expect(presetFormMocks.toast.add).toHaveBeenCalledOnce();
    });

    it('ignores a missing file', async () => {
      const presetForm = mount();

      await presetForm.importPreset(null);

      expect(presetForm.loaded.value).toBe(false);
      expect(presetFormMocks.toast.add).not.toHaveBeenCalled();
    });
  });

  describe('downloadPreset', () => {
    it('downloads the preset with unsaved changes', () => {
      const presetForm = mount(1);
      presetForm.preset.comment = 'changed';

      presetForm.downloadPreset();

      expect(presetFormMocks.download).toHaveBeenCalledWith(
        'test.json',
        expect.objectContaining({
          comment: 'changed',
          lanes: createPreset('test', 1).lanes,
        }),
      );
    });

    it('ignores a missing preset', () => {
      const presetForm = mount();

      presetForm.downloadPreset();

      expect(presetFormMocks.download).not.toHaveBeenCalled();
    });
  });

  describe('savePreset', () => {
    it('saves after a confirmation and reopens the saved preset', async () => {
      const presetForm = mount(2);
      presetForm.preset.comment = 'new comment';
      echoSave();

      await presetForm.savePreset();
      await flush();

      expect(presetFormMocks.presets.save).toHaveBeenCalledWith(
        'test',
        expect.objectContaining({
          comment: 'new comment',
          lanes: createPreset('test', 2).lanes,
        }),
      );
      expect(presetForm.isDirty.value).toBe(false);
      expect(presetFormMocks.presets.list).toHaveBeenCalled();
    });

    it('stores the background image of a draft on its first save', async () => {
      const presetForm = mount();
      const image = new Blob(['image']);
      await presetForm.addPreset('draft', 2);
      echoSave();
      await presetForm.setBackgroundImage(image);

      await presetForm.savePreset();
      await flush();

      expect(presetFormMocks.images.set).toHaveBeenCalledWith('preset:draft', image);
    });

    it('saves a renamed preset and deletes it under its old name', async () => {
      const presetForm = mount(1);
      presetForm.preset.name = 'renamed';
      echoSave();

      await presetForm.savePreset();
      await flush();

      expect(presetFormMocks.presets.save).toHaveBeenCalledWith('renamed', expect.anything());
      expect(presetFormMocks.presets.delete).toHaveBeenCalledWith('test');
      expect(presetForm.savedName.value).toBe('renamed');
    });

    it('rejects a new name another preset already has', async () => {
      const presetForm = mount(1);
      presetForm.presetNames.value = ['test', 'other'];
      presetForm.preset.name = 'other';

      await presetForm.savePreset();

      expect(presetFormMocks.presets.save).not.toHaveBeenCalled();
      expect(presetFormMocks.toast.add).toHaveBeenCalledOnce();
    });

    it('keeps the preset unsaved without a confirmation', async () => {
      const presetForm = mount(1);
      presetFormMocks.confirm.mockResolvedValueOnce(false);

      await presetForm.savePreset();

      expect(presetFormMocks.presets.save).not.toHaveBeenCalled();
    });

    it('ignores a missing preset', async () => {
      const presetForm = mount();

      await presetForm.savePreset();

      expect(presetFormMocks.presets.save).not.toHaveBeenCalled();
    });
  });

  describe('resetPreset', () => {
    it('resets to the saved state after a confirmation', async () => {
      const presetForm = mount(1);
      presetForm.preset.comment = 'changed';
      presetForm.editor.selection.selectOnly('1');

      await presetForm.resetPreset();

      expect(presetForm.preset.comment).toBeNull();
      expect(presetForm.editor.selection.selected.value.size).toBe(0);
    });

    it('keeps the changes without a confirmation', async () => {
      const presetForm = mount(1);
      presetForm.preset.comment = 'changed';
      presetFormMocks.confirm.mockResolvedValueOnce(false);

      await presetForm.resetPreset();

      expect(presetForm.preset.comment).toBe('changed');
    });
  });

  describe('deletePreset', () => {
    it('deletes the stored preset after a confirmation', async () => {
      const presetForm = mount(1);

      await presetForm.deletePreset();
      await flush();

      expect(presetFormMocks.presets.delete).toHaveBeenCalledWith('test');
      expect(presetForm.loaded.value).toBe(false);
    });

    it('deletes the background image with the preset', async () => {
      const presetForm = mount(1);
      await presetForm.setBackgroundImage(new Blob(['image']));

      await presetForm.deletePreset();
      await flush();

      expect(presetFormMocks.images.delete).toHaveBeenCalledWith('preset:test');
      expect(presetForm.editor.backgroundImage.image.value).toBeNull();
    });
  });

  describe('align', () => {
    it('aligns the lanes in one history entry', () => {
      const presetForm = mount(3);
      presetForm.preset.lanes['3']![CornerKey.BottomRight] = pixelPoint(1440, 400);

      presetForm.align();

      expect(presetForm.editor.history.canUndo.value).toBe(true);
    });

    it('reports an invalid outline', () => {
      const presetForm = mount(3);
      presetForm.preset.lanes['3']![CornerKey.BottomRight] = pixelPoint(500, 160);

      presetForm.align();

      expect(presetFormMocks.toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.editor.alignError',
        }),
      );
      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });

    it('ignores a missing preset', () => {
      const presetForm = mount();

      presetForm.align();

      expect(presetFormMocks.toast.add).not.toHaveBeenCalled();
    });
  });

  describe('setLaneGap', () => {
    it('distributes the lanes again when the lane gap changes', () => {
      const presetForm = mount(3);

      presetForm.setLaneGap(0);

      expect(presetForm.preset.laneGap).toBe(0);
      expect(presetForm.preset.lanes['2']![CornerKey.TopLeft].y).toBe(
        presetForm.preset.lanes['1']![CornerKey.BottomLeft].y,
      );
      expect(presetForm.editor.history.canUndo.value).toBe(true);
    });

    it('records the lane gap in the history', () => {
      const presetForm = mount(3);
      presetForm.setLaneGap(0);

      presetForm.editor.history.undo();

      expect(presetForm.preset.laneGap).toBe(DEFAULT_LANE_GAP);
    });

    it('keeps the lanes when the lane gap changes without a valid outline', () => {
      const presetForm = mount(3);
      presetForm.preset.lanes['3']![CornerKey.BottomRight] = pixelPoint(500, 160);
      const before = structuredClone(toRaw(presetForm.preset.lanes['2']!));

      presetForm.setLaneGap(50);

      expect(presetForm.preset.laneGap).toBe(50);
      expect(presetForm.preset.lanes['2']).toEqual(before);
      expect(presetFormMocks.toast.add).not.toHaveBeenCalled();
    });

    it.each([
      {
        laneGap: 101,
      },
      {
        laneGap: 2.5,
      },
    ])('ignores the lane gap $laneGap the backend rejects', ({ laneGap }) => {
      const presetForm = mount(3);

      presetForm.setLaneGap(laneGap);

      expect(presetForm.preset.laneGap).toBe(DEFAULT_LANE_GAP);
    });
  });

  describe('toggleLock', () => {
    it('locks the selected lane', () => {
      const presetForm = mount(2);
      presetForm.editor.selection.selectOnly('1');

      presetForm.toggleLock();

      expect(presetForm.activeLane.value?.locked).toBe(true);
    });

    it('records the lock in the history', () => {
      const presetForm = mount(2);
      presetForm.editor.selection.selectOnly('1');
      presetForm.toggleLock();

      presetForm.editor.history.undo();

      expect(presetForm.activeLane.value?.locked).toBe(false);
    });

    it('ignores an empty selection', () => {
      const presetForm = mount(2);

      presetForm.toggleLock();

      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });
  });

  describe('nudge', () => {
    it('moves the selected lane', () => {
      const presetForm = mount(2);
      presetForm.editor.selection.selectOnly('1');
      const before = presetForm.preset.lanes['1']![CornerKey.TopLeft].x;

      presetForm.nudge(10, 0);

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft].x).toBe(before + 10);
    });

    it('keeps a locked lane in place', () => {
      const presetForm = mount(2);
      presetForm.editor.selection.selectOnly('1');
      presetForm.toggleLock();
      const before = presetForm.preset.lanes['1']![CornerKey.TopLeft].x;

      presetForm.nudge(10, 0);

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft].x).toBe(before);
    });

    it('ignores a missing preset', () => {
      const presetForm = mount();

      presetForm.nudge(1, 1);

      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });
  });

  describe('setLocked', () => {
    it('locks a single lane', () => {
      const presetForm = mount(2);

      presetForm.setLocked('2', true);

      expect(presetForm.preset.lanes['2']!.locked).toBe(true);
      expect(presetForm.preset.lanes['1']!.locked).toBe(false);
    });
  });

  describe('alignSelection', () => {
    it('lines up the selected lanes at an edge', () => {
      const presetForm = mount(2);
      presetForm.preset.lanes['2']![CornerKey.TopLeft].x = 400;
      presetForm.preset.lanes['2']![CornerKey.BottomLeft].x = 400;
      presetForm.editor.selection.set(['1', '2']);

      presetForm.alignSelection('left');

      expect(laneBounds(presetForm.preset.lanes['1']!).minX).toBe(400);
    });

    it('records the alignment as one history entry', () => {
      const presetForm = mount(2);
      presetForm.preset.lanes['2']![CornerKey.TopLeft].x = 400;
      presetForm.preset.lanes['2']![CornerKey.BottomLeft].x = 400;
      presetForm.editor.history.commit();
      presetForm.editor.selection.set(['1', '2']);
      presetForm.alignSelection('left');

      presetForm.editor.history.undo();

      expect(laneBounds(presetForm.preset.lanes['1']!).minX).toBe(480);
    });

    it('centers a single lane in the image', () => {
      const presetForm = mount(1);
      presetForm.editor.selection.selectOnly('1');

      presetForm.alignSelection('centerY');

      const bounds = laneBounds(presetForm.preset.lanes['1']!);
      expect((bounds.minY + bounds.maxY) / 2).toBe(540);
    });

    it('keeps a locked lane in place', () => {
      const presetForm = mount(2);
      presetForm.preset.lanes['2']!.locked = true;
      const before = laneBounds(presetForm.preset.lanes['2']!);
      presetForm.editor.selection.set(['1', '2']);

      presetForm.alignSelection('top');

      expect(laneBounds(presetForm.preset.lanes['2']!)).toEqual(before);
    });
  });

  describe('distributeSelection', () => {
    it('evens out the gaps between the selected lanes', () => {
      const presetForm = mount(3);
      presetForm.preset.lanes['2']![CornerKey.TopLeft].y += 20;
      presetForm.preset.lanes['2']![CornerKey.TopRight].y += 20;
      presetForm.preset.lanes['2']![CornerKey.BottomRight].y += 20;
      presetForm.preset.lanes['2']![CornerKey.BottomLeft].y += 20;
      presetForm.editor.selection.selectAll();

      presetForm.distributeSelection('y');

      expect(laneBounds(presetForm.preset.lanes['2']!).minY).toBe(220);
    });
  });

  describe('transformSelection', () => {
    it('moves and scales the selected lanes into new bounds', () => {
      const presetForm = mount(2);
      presetForm.editor.selection.selectAll();
      const from = laneBounds(presetForm.preset.lanes['1']!);

      presetForm.transformSelection({
        minX: 0,
        maxX: 480,
        minY: 160,
        maxY: 270,
      });

      expect(laneBounds(presetForm.preset.lanes['1']!)).toEqual({
        minX: 0,
        maxX: 480,
        minY: from.minY,
        maxY: from.maxY,
      });
    });

    it('keeps a locked lane in place', () => {
      const presetForm = mount(2);
      presetForm.preset.lanes['2']!.locked = true;
      const before = laneBounds(presetForm.preset.lanes['2']!);
      presetForm.editor.selection.selectAll();

      presetForm.transformSelection({
        minX: 0,
        maxX: 100,
        minY: 0,
        maxY: 100,
      });

      expect(laneBounds(presetForm.preset.lanes['2']!)).toEqual(before);
    });
  });

  describe('setFrameSize', () => {
    it('changes the reference resolution and keeps the lanes in place', () => {
      const presetForm = mount(1);

      presetForm.setFrameSize(1280, 720);

      expect(presetForm.preset.frameWidth).toBe(1280);
      expect(presetForm.preset.frameHeight).toBe(720);
      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: 320,
        y: 106.67,
      });
    });

    it('records the reference resolution in the history', () => {
      const presetForm = mount(1);
      presetForm.setFrameSize(1280, 720);

      presetForm.editor.history.undo();

      expect(presetForm.preset.frameWidth).toBe(1920);
      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: 480,
        y: 160,
      });
    });

    it.each([
      {
        width: 0,
      },
      {
        width: 1280.5,
      },
    ])('ignores the width $width the backend rejects', ({ width }) => {
      const presetForm = mount(1);

      presetForm.setFrameSize(width, 720);

      expect(presetForm.preset.frameWidth).toBe(1920);
      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });

    it('ignores a missing preset', () => {
      const presetForm = mount();

      presetForm.setFrameSize(1280, 720);

      expect(presetForm.editor.history.canUndo.value).toBe(false);
    });
  });

  describe('setBackgroundImage', () => {
    it('stores the background image of a saved preset at once', async () => {
      const presetForm = mount(1);
      const image = new Blob(['image']);

      await presetForm.setBackgroundImage(image);

      expect(presetFormMocks.images.set).toHaveBeenCalledWith('preset:test', image);
    });

    it('deletes the background image of a saved preset at once', async () => {
      const presetForm = mount(1);
      await presetForm.setBackgroundImage(new Blob(['image']));

      await presetForm.setBackgroundImage(null);

      expect(presetFormMocks.images.delete).toHaveBeenCalledWith('preset:test');
      expect(presetForm.editor.backgroundImage.image.value).toBeNull();
    });

    it('keeps the background image of a draft until its first save', async () => {
      const presetForm = mount();
      const image = new Blob(['image']);
      await presetForm.addPreset('draft', 2);

      await presetForm.setBackgroundImage(image);

      expect(presetFormMocks.images.set).not.toHaveBeenCalled();
    });
  });
});

describe('usePresetForm', () => {
  let unmount = () => {};

  afterEach(() => unmount());

  it('returns the preset form provided by the editor page', () => {
    let provided: PresetForm | undefined;
    let injected: PresetForm | undefined;
    const Child = defineComponent({
      setup() {
        injected = usePresetForm();
        return () => null;
      },
    });
    const app = createApp({
      setup() {
        provided = providePresetForm(null);
        return () => h(Child);
      },
    });

    app.mount(document.createElement('div'));
    unmount = () => app.unmount();

    expect(injected).toBe(provided);
  });

  it('fails outside of the editor page', () => {
    const setup = () => withSetup(() => usePresetForm());

    expect(setup).toThrow('usePresetForm must be used below providePresetForm');
  });
});
