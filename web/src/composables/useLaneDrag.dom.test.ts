import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { CornerKey } from '@/api/types';
import { noSnap, type AxisTargets } from '@/lib/editor/snap';
import { pointer } from '@/test/pointer';
import { mountPresetForm, resetPresetFormMocks } from '@/test/presetForm';
import { useLaneDrag } from '@/composables/useLaneDrag';

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

describe('useLaneDrag', () => {
  let unmount = () => {};

  beforeEach(() => resetPresetFormMocks());
  // A test may end with the pointer still pressed, so the gesture is released before the next one
  afterEach(() => {
    window.dispatchEvent(pointer('pointerup'));
    unmount();
  });

  /** mount creates the lane drag on an editor with three lanes and the given snapping and zoom. */
  function mount(
    snapping = false,
    scale = 1,
    guides: AxisTargets = {
      x: [],
      y: [],
    },
  ) {
    const mounted = mountPresetForm(
      (presetForm) =>
        useLaneDrag(
          {
            snapping,
            toFrame: (clientX, clientY) => ({
              x: clientX / scale,
              y: clientY / scale,
            }),
            scale,
            guides: () => guides,
          },
          presetForm,
        ),
      3,
    );
    unmount = mounted.unmount;
    return {
      presetForm: mounted.presetForm,
      drag: mounted.result,
    };
  }

  describe('startLaneDrag', () => {
    it('selects the lane and starts a lane drag', () => {
      const { presetForm, drag } = mount();

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 300,
          clientY: 170,
        }),
        '1',
      );

      expect(drag.mode.value).toBe('lane');
      expect([...presetForm.editor.selection.selected.value]).toEqual(['1']);
    });

    it('moves a lane by the distance of the pointer', () => {
      const { presetForm, drag } = mount();
      const before = {
        ...presetForm.preset.lanes['1']![CornerKey.TopLeft],
      };

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 300,
          clientY: 170,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 310,
          clientY: 175,
        }),
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 320,
          clientY: 180,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: before.x + 20,
        y: before.y + 10,
      });
      expect(drag.mode.value).toBeNull();
    });

    it('records one history entry for the whole drag', () => {
      const { presetForm, drag } = mount();
      const before = {
        ...presetForm.preset.lanes['1']![CornerKey.TopLeft],
      };
      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 300,
          clientY: 170,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 310,
          clientY: 175,
        }),
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 320,
          clientY: 180,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      presetForm.editor.history.undo();

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual(before);
    });

    it('moves the whole selection and skips locked lanes', () => {
      const { presetForm, drag } = mount();
      presetForm.editor.selection.selected.value = new Set(['1', '2']);
      presetForm.preset.lanes['2']!.locked = true;
      const locked = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };
      const moved = {
        ...presetForm.preset.lanes['1']![CornerKey.TopLeft],
      };

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 0,
          clientY: 0,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: 40,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: moved.x,
        y: moved.y + 40,
      });
      expect(presetForm.preset.lanes['2']![CornerKey.TopLeft]).toEqual(locked);
    });

    it('adds a lane to the selection with Shift', () => {
      const { presetForm, drag } = mount();
      drag.startLaneDrag(pointer('pointerdown'), '1');
      window.dispatchEvent(pointer('pointerup'));

      drag.startLaneDrag(
        pointer('pointerdown', {
          shiftKey: true,
        }),
        '2',
      );
      window.dispatchEvent(pointer('pointerup'));

      expect([...presetForm.editor.selection.selected.value].sort()).toEqual(['1', '2']);
    });

    it('adds a lane to the selection with Ctrl or Cmd', () => {
      const { presetForm, drag } = mount();
      presetForm.editor.selection.selected.value = new Set(['1']);

      drag.startLaneDrag(
        pointer('pointerdown', {
          ctrlKey: true,
        }),
        '2',
      );
      window.dispatchEvent(pointer('pointerup'));

      expect([...presetForm.editor.selection.selected.value].sort()).toEqual(['1', '2']);
    });

    it('removes a selected lane from the selection with Shift', () => {
      const { presetForm, drag } = mount();
      presetForm.editor.selection.selected.value = new Set(['1', '2']);

      drag.startLaneDrag(
        pointer('pointerdown', {
          shiftKey: true,
        }),
        '1',
      );
      window.dispatchEvent(pointer('pointerup'));

      expect([...presetForm.editor.selection.selected.value]).toEqual(['2']);
    });

    it('starts no drag when Shift removes the last selected lane', () => {
      const { presetForm, drag } = mount();
      presetForm.editor.selection.selected.value = new Set(['2']);

      drag.startLaneDrag(
        pointer('pointerdown', {
          shiftKey: true,
        }),
        '2',
      );

      expect(presetForm.editor.selection.selected.value.size).toBe(0);
      expect(drag.mode.value).toBeNull();
    });

    it('converts the pointer into pixels of the reference resolution', () => {
      const { presetForm, drag } = mount(false, 0.5);
      const before = {
        ...presetForm.preset.lanes['1']![CornerKey.TopLeft],
      };

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 100,
          clientY: 100,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 110,
          clientY: 105,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopLeft]).toEqual({
        x: before.x + 20,
        y: before.y + 10,
      });
    });

    it('snaps a moved lane onto its neighbor', () => {
      const { presetForm, drag } = mount(true);
      const lane1 = presetForm.preset.lanes['1']!;
      const gap =
        presetForm.preset.lanes['2']![CornerKey.TopLeft].y - lane1[CornerKey.BottomLeft].y;

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 0,
          clientY: 0,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: gap - 3,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(lane1[CornerKey.BottomLeft].y).toBe(
        presetForm.preset.lanes['2']![CornerKey.TopLeft].y,
      );
    });

    it('aligns a moved lane with the edges of the others without a corner in reach', () => {
      const { presetForm, drag } = mount(true);
      const lane1 = presetForm.preset.lanes['1']!;
      const left = lane1[CornerKey.TopLeft].x;

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 0,
          clientY: 0,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 3,
          clientY: -60,
        }),
      );

      expect(lane1[CornerKey.TopLeft].x).toBe(left);
      expect(drag.feedback.value.lines).toEqual([
        {
          axis: 'x',
          value: left,
        },
      ]);
    });

    it('keeps the move on one axis while Shift is held', () => {
      const { presetForm, drag } = mount();
      const lane = presetForm.preset.lanes['1']!;
      const before = {
        ...lane[CornerKey.TopLeft],
      };
      drag.startLaneDrag(pointer('pointerdown'), '1');

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 30,
          clientY: 4,
          shiftKey: true,
        }),
      );

      expect(lane[CornerKey.TopLeft]).toEqual({
        x: before.x + 30,
        y: before.y,
      });
    });

    it('shows the distances to the neighbors', () => {
      const { drag } = mount();
      drag.startLaneDrag(pointer('pointerdown'), '2');

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: 5,
        }),
      );

      expect(drag.feedback.value.gaps.filter((gap) => gap.axis === 'y')).toHaveLength(2);
    });

    it('keeps even gaps to the neighbors without an edge in reach', () => {
      // Zoomed in, so the corners of the neighbors are out of the snap reach
      const { presetForm, drag } = mount(true, 2);
      const lane = presetForm.preset.lanes['2']!;
      const top = lane[CornerKey.TopLeft].y;
      drag.startLaneDrag(pointer('pointerdown'), '2');

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: 6,
        }),
      );

      expect(lane[CornerKey.TopLeft].y).toBe(top);
    });

    it('aligns a moved lane with a guide', () => {
      const { presetForm, drag } = mount(true, 1, {
        x: [500],
        y: [],
      });
      const lane = presetForm.preset.lanes['1']!;
      drag.startLaneDrag(pointer('pointerdown'), '1');

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 17,
          clientY: -80,
        }),
      );

      expect(lane[CornerKey.TopLeft].x).toBe(500);
    });

    it('snaps to no hidden lane', () => {
      const { presetForm, drag } = mount(true);
      presetForm.editor.visibility.toggle('2');
      const lane1 = presetForm.preset.lanes['1']!;
      const gap =
        presetForm.preset.lanes['2']![CornerKey.TopLeft].y - lane1[CornerKey.BottomLeft].y;
      drag.startLaneDrag(pointer('pointerdown'), '1');

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: gap - 3,
        }),
      );

      expect(lane1[CornerKey.BottomLeft].y).not.toBe(
        presetForm.preset.lanes['2']![CornerKey.TopLeft].y,
      );
    });

    it('narrows a lane pushed against the image border', () => {
      const { presetForm, drag } = mount();
      const lane = presetForm.preset.lanes['1']!;
      const top = lane[CornerKey.TopLeft].y;
      const bottom = lane[CornerKey.BottomLeft].y;

      drag.startLaneDrag(
        pointer('pointerdown', {
          clientX: 0,
          clientY: 0,
        }),
        '1',
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 0,
          clientY: -top - 10,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(lane[CornerKey.TopLeft].y).toBe(0);
      expect(lane[CornerKey.BottomLeft].y).toBe(bottom - top - 10);
    });

    it('holds back the automatic history entry until the drag ends', () => {
      const { presetForm, drag } = mount();

      drag.startLaneDrag(pointer('pointerdown'), '1');
      const during = presetForm.editor.dragging.value;
      window.dispatchEvent(pointer('pointerup'));

      expect(during).toBe(true);
      expect(presetForm.editor.dragging.value).toBe(false);
    });
  });

  describe('startCornerDrag', () => {
    it('starts a corner drag', () => {
      const { drag } = mount();

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.TopRight);

      expect(drag.mode.value).toBe('corner');
    });

    it('lets a corner follow the pointer', () => {
      const { presetForm, drag } = mount();

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.TopRight);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 812.6,
          clientY: 140,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopRight]).toEqual({
        x: 813,
        y: 140,
      });
    });

    it('converts the pointer into pixels of the reference resolution', () => {
      const { presetForm, drag } = mount(false, 0.5);

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.TopRight);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 400,
          clientY: 70,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopRight]).toEqual({
        x: 800,
        y: 140,
      });
    });

    it('snaps a corner onto a neighboring lane', () => {
      const { presetForm, drag } = mount(true);
      const target = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.BottomLeft);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: target.x + 3,
          clientY: target.y - 4,
        }),
      );

      expect(presetForm.preset.lanes['1']![CornerKey.BottomLeft]).toEqual(target);
      expect(drag.feedback.value.point).toEqual(target);
    });

    it('aligns a corner with the edges of other lanes without a corner in reach', () => {
      const { presetForm, drag } = mount(true);
      const right = presetForm.preset.lanes['2']![CornerKey.TopRight].x;

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.TopRight);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: right + 3,
          clientY: 50,
        }),
      );

      expect(presetForm.preset.lanes['1']![CornerKey.TopRight]).toEqual({
        x: right,
        y: 50,
      });
      expect(drag.feedback.value.lines).toEqual([
        {
          axis: 'x',
          value: right,
        },
      ]);
    });

    it('snaps a corner within the reach on the screen when zoomed', () => {
      const { presetForm, drag } = mount(true, 4);
      const target = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.BottomLeft);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: (target.x - 1) * 4,
          clientY: (target.y - 1) * 4,
        }),
      );

      expect(presetForm.preset.lanes['1']![CornerKey.BottomLeft]).toEqual(target);
    });

    it('keeps the snap reach the same on the screen when zoomed', () => {
      const { presetForm, drag } = mount(true, 4);
      const target = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };

      // Within the snap reach in pixels of the reference resolution, but outside it on the screen
      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.BottomLeft);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: (target.x - 4) * 4,
          clientY: (target.y - 4) * 4,
        }),
      );

      expect(drag.feedback.value).toEqual(noSnap());
    });

    it('keeps a corner where the pointer is while Ctrl or Cmd is held', () => {
      const { presetForm, drag } = mount(true);
      const target = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };
      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.BottomLeft);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: target.x + 3,
          clientY: target.y - 4,
        }),
      );

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: target.x + 3,
          clientY: target.y - 4,
          ctrlKey: true,
        }),
      );

      expect(presetForm.preset.lanes['1']![CornerKey.BottomLeft]).toEqual({
        x: target.x + 3,
        y: target.y - 4,
      });
      expect(drag.feedback.value).toEqual(noSnap());
    });

    it('clears the snap feedback on release', () => {
      const { presetForm, drag } = mount(true);
      const target = {
        ...presetForm.preset.lanes['2']![CornerKey.TopLeft],
      };
      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.BottomLeft);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: target.x + 3,
          clientY: target.y - 4,
        }),
      );

      window.dispatchEvent(pointer('pointerup'));

      expect(drag.feedback.value).toEqual(noSnap());
    });

    it('keeps dragged corners within the image', () => {
      const { presetForm, drag } = mount();

      drag.startCornerDrag(pointer('pointerdown'), '1', CornerKey.TopRight);
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 2500,
          clientY: -30,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(presetForm.preset.lanes['1']![CornerKey.TopRight]).toEqual({
        x: 1920,
        y: 0,
      });
    });
  });
});
