import { afterEach, describe, expect, it } from 'vitest';

import type { Bounds } from '@/lib/editor/geometry';
import { pointer } from '@/test/pointer';
import { withSetup } from '@/test/withSetup';
import { EDITOR_SURFACE_ATTRIBUTE, useEditorCanvas } from '@/composables/editor/useEditorCanvas';
import { useEditorGuides } from '@/composables/editor/useEditorGuides';
import { useEditorSelection } from '@/composables/editor/useEditorSelection';

// One item of 100 x 50 in content units at the origin.
const ITEMS: Array<[string, Bounds]> = [
  [
    'a',
    {
      minX: 0,
      maxX: 100,
      minY: 0,
      maxY: 50,
    },
  ],
];

describe('useEditorCanvas', () => {
  let unmount = () => {};

  afterEach(() => {
    window.dispatchEvent(pointer('pointerup'));
    unmount();
  });

  /** mount creates the canvas over a 200 x 100 content, with a gesture and a view mode the test sets. */
  function mount(state = { gesture: false, viewMode: false }) {
    const mounted = withSetup(() => {
      const selection = useEditorSelection(() => ['a']);
      const canvas = useEditorCanvas({
        guides: useEditorGuides({
          storageKey: 'canvas-test',
          document: () => null,
        }),
        contentWidth: 200,
        contentHeight: 100,
        padding: 0,
        actualScale: 2,
        selection,
        items: () => ITEMS,
        gesture: () => state.gesture,
        viewMode: () => state.viewMode,
      });
      return {
        selection,
        canvas,
      };
    });
    unmount = mounted.unmount;
    return mounted.result;
  }

  describe('dragging', () => {
    it('follows a gesture on an item', () => {
      const { canvas } = mount({
        gesture: true,
        viewMode: false,
      });

      const dragging = canvas.dragging.value;

      expect(dragging).toBe(true);
    });

    it('follows a marquee', () => {
      const { canvas } = mount();
      const area = document.createElement('div');
      canvas.setArea(area);
      const event = pointer('pointerdown');
      Object.defineProperty(event, 'target', {
        value: area,
      });

      canvas.onAreaPointerDown(event);

      expect(canvas.dragging.value).toBe(true);
    });
  });

  describe('actualZoom', () => {
    it('relates the scale to the actual size of the content', () => {
      const { canvas } = mount();

      canvas.zoomToActual(1);

      expect(canvas.actualZoom.value).toBeCloseTo(1);
    });
  });

  describe('onAreaPointerDown', () => {
    it('starts a marquee on the area', () => {
      const { canvas } = mount();
      const area = document.createElement('div');
      canvas.setArea(area);
      const event = pointer('pointerdown');
      Object.defineProperty(event, 'target', {
        value: area,
      });

      canvas.onAreaPointerDown(event);

      expect(canvas.marquee.value).not.toBeNull();
    });

    it('starts a marquee on the surface of the content', () => {
      const { canvas } = mount();
      const surface = document.createElement('div');
      surface.setAttribute(EDITOR_SURFACE_ATTRIBUTE, '');
      const event = pointer('pointerdown');
      Object.defineProperty(event, 'target', {
        value: surface,
      });

      canvas.onAreaPointerDown(event);

      expect(canvas.marquee.value).not.toBeNull();
    });

    it('leaves a press on an item to the item', () => {
      const { canvas } = mount();
      const event = pointer('pointerdown');
      Object.defineProperty(event, 'target', {
        value: document.createElement('div'),
      });

      canvas.onAreaPointerDown(event);

      expect(canvas.marquee.value).toBeNull();
    });

    it('starts no marquee in the view mode', () => {
      const { canvas } = mount({
        gesture: false,
        viewMode: true,
      });
      const area = document.createElement('div');
      canvas.setArea(area);
      const event = pointer('pointerdown');
      Object.defineProperty(event, 'target', {
        value: area,
      });

      canvas.onAreaPointerDown(event);

      expect(canvas.marquee.value).toBeNull();
    });
  });

  describe('setArea', () => {
    it('ignores what is no element', () => {
      const { canvas } = mount();

      canvas.setArea(null);

      expect(canvas.area.value).toBeNull();
    });
  });

  describe('zoomToBounds', () => {
    it('keeps the zoom without bounds', () => {
      const { canvas } = mount();

      canvas.zoomToBounds(null);

      expect(canvas.viewport.zoom.value).toBe(1);
    });
  });
});
