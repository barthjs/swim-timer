import { nextTick, reactive } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { CornerKey } from '@/api/types/preset';
import { createPreset, toEditorPreset, type EditorPreset } from '@/lib/preset/lanes';
import type { KeyValueStore } from '@/lib/browser/keyValueStore';

import { usePresetEditor } from '@/composables/preset/usePresetEditor';
import { imageLoad } from '@/test/image';

const store = vi.hoisted(() => ({
  get: vi.fn<KeyValueStore<Blob>['get']>(async () => null),
  set: vi.fn<KeyValueStore<Blob>['set']>(async () => {}),
  delete: vi.fn<KeyValueStore<Blob>['delete']>(async () => {}),
}));

vi.mock('@/lib/preset/backgroundImages', () => ({
  backgroundImageStore: store,
}));

describe('usePresetEditor', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  /** Creates the editor over a preset with three lanes in the default reference resolution. */
  function setup() {
    const preset = reactive<EditorPreset>(toEditorPreset(createPreset('test', 3)));
    const editor = usePresetEditor(
      preset,
      () => ['1', '2', '3'],
      () => 'test',
    );
    editor.history.reset();
    return {
      preset,
      editor,
    };
  }

  describe('selection', () => {
    it('steps through the lanes in the order of their numbers', () => {
      const { editor } = setup();
      editor.selection.selectOnly('3');

      editor.selection.selectNext(1);

      expect(editor.selection.active.value).toBe('1');
    });
  });

  describe('history', () => {
    it('reverts typed changes as one step', async () => {
      const { preset, editor } = setup();
      preset.description = 'a';
      await nextTick();
      vi.advanceTimersByTime(100);
      preset.description = 'ab';
      await nextTick();
      vi.advanceTimersByTime(400);

      editor.history.undo();

      expect(preset.description).toBeNull();
      expect(editor.history.canUndo.value).toBe(false);
    });

    it('leaves the commit of a drag to its end', async () => {
      const { preset, editor } = setup();
      editor.dragging.value = true;

      preset.lanes['1']![CornerKey.TopLeft].x = 10;
      await nextTick();
      vi.advanceTimersByTime(1000);

      expect(editor.history.canUndo.value).toBe(false);
    });
  });

  describe('backgroundImage', () => {
    it('keeps the image of a preset under its name', async () => {
      const { editor } = setup();
      const image = new Blob(['image']);

      await editor.backgroundImage.set(image, 'test');

      expect(store.set).toHaveBeenCalledWith('preset:test', image);
    });

    it('warns about an image with another aspect ratio than the reference resolution', () => {
      const { editor } = setup();

      editor.backgroundImage.onLoad(imageLoad(1000, 1000));

      expect(editor.backgroundImage.mismatch.value).toBe(true);
    });
  });

  describe('guides', () => {
    it('keeps the guides of a preset under its name', async () => {
      const { editor } = setup();
      const lines = [
        {
          axis: 'x' as const,
          value: 100,
        },
      ];

      editor.guides.lines.value = lines;
      await nextTick();

      expect(JSON.parse(localStorage.getItem('editor.guides.test') ?? '[]')).toEqual(lines);
    });
  });
});
