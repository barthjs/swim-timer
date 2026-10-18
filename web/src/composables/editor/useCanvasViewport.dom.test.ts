import { nextTick, ref } from 'vue';
import { afterEach, describe, expect, it } from 'vitest';

import { pointer } from '@/test/pointer';
import { withSetup } from '@/test/withSetup';
import { DEFAULT_MAX_ZOOM, useCanvasViewport } from '@/composables/editor/useCanvasViewport';

type Rect = { left: number; top: number; width: number; height: number };

/** areaElement creates an element that reports the given position and size. */
function areaElement(rect: Rect): { element: HTMLElement; resize: (next: Rect) => void } {
  const element = document.createElement('div');
  let current = rect;
  element.getBoundingClientRect = () =>
    ({
      ...current,
      x: current.left,
      y: current.top,
      right: current.left + current.width,
      bottom: current.top + current.height,
      toJSON: () => ({}),
    }) as DOMRect;
  return {
    element,
    resize: (next) => {
      current = next;
      window.dispatchEvent(new Event('resize'));
    },
  };
}

type WheelInit = {
  deltaX?: number;
  deltaY: number;
  deltaMode?: number;
  clientX?: number;
  clientY?: number;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
};

/**
 * wheel creates a wheel event with a position and modifier keys, which the WheelEvent of happy-dom leaves out.
 */
function wheel(init: WheelInit) {
  const event = new WheelEvent('wheel', {
    cancelable: true,
    deltaY: init.deltaY,
  });
  Object.defineProperty(event, 'deltaX', {
    value: init.deltaX ?? 0,
  });
  Object.defineProperty(event, 'deltaMode', {
    value: init.deltaMode ?? 0,
  });
  Object.defineProperty(event, 'clientX', {
    value: init.clientX ?? 0,
  });
  Object.defineProperty(event, 'clientY', {
    value: init.clientY ?? 0,
  });
  Object.defineProperty(event, 'ctrlKey', {
    value: init.ctrlKey ?? false,
  });
  Object.defineProperty(event, 'metaKey', {
    value: init.metaKey ?? false,
  });
  Object.defineProperty(event, 'shiftKey', {
    value: init.shiftKey ?? false,
  });
  return event;
}

/** key creates a keyboard event for a key code. */
function key(type: 'keydown' | 'keyup', code: string): KeyboardEvent {
  return new KeyboardEvent(type, {
    code,
    bubbles: true,
    cancelable: true,
  });
}

describe('useCanvasViewport', () => {
  let unmount = () => {};

  afterEach(() => unmount());

  /** mount creates a viewport over an area of the given size and unmounts it after the test. */
  function mount(rect: Rect, contentWidth = 1920, contentHeight = 1080, padding = 0) {
    const area = areaElement(rect);
    const width = ref(contentWidth);
    const height = ref(contentHeight);
    const mounted = withSetup(() =>
      useCanvasViewport({
        area: area.element,
        contentWidth: width,
        contentHeight: height,
        padding,
      }),
    );
    unmount = mounted.unmount;
    return {
      viewport: mounted.result,
      area,
      width,
      height,
    };
  }

  describe('fitScale', () => {
    it('fits wide content into the width of the area and centers it', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 1000,
      });

      const scale = viewport.fitScale.value;

      expect(scale).toBe(0.5);
      expect(viewport.pan).toEqual({
        x: 0,
        y: 230,
      });
      expect(viewport.contentStyle.value).toEqual({
        width: '1920px',
        height: '1080px',
        transform: 'translate(0px, 230px) scale(0.5)',
        transformOrigin: '0 0',
      });
    });

    it('fits tall content into the height of the area and keeps the padding', () => {
      const { viewport } = mount(
        {
          left: 0,
          top: 0,
          width: 1000,
          height: 1000,
        },
        1080,
        1920,
        20,
      );

      const scale = viewport.fitScale.value;

      expect(scale).toBe(0.5);
      expect(viewport.pan).toEqual({
        x: 230,
        y: 20,
      });
    });

    it('refits after a resize of the area while neither zoomed nor panned', async () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      area.resize({
        left: 0,
        top: 0,
        width: 480,
        height: 540,
      });
      await nextTick();

      expect(viewport.fitScale.value).toBe(0.25);
      expect(viewport.pan.y).toBe(135);
    });

    it('refits after a change of the content size while neither zoomed nor panned', async () => {
      const { viewport, width, height } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      width.value = 1080;
      height.value = 1920;
      await nextTick();

      expect(viewport.fitScale.value).toBe(540 / 1920);
    });
  });

  describe('transform', () => {
    it('places the content by its pan and scale', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.panBy(10, 20);

      expect(viewport.transform.value).toEqual({
        x: 10,
        y: 20,
        scale: 0.5,
      });
    });
  });

  describe('center', () => {
    it('centers the content again', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      viewport.panBy(10, -5);

      viewport.center();

      expect(viewport.pan).toEqual({
        x: 0,
        y: 0,
      });
    });
  });

  describe('zoomAt', () => {
    it('keeps the point under the cursor in place when zooming', () => {
      const { viewport } = mount({
        left: 100,
        top: 50,
        width: 960,
        height: 540,
      });
      const before = viewport.clientToContent(400, 200);

      viewport.zoomAt(300, 150, 2);

      expect(viewport.zoom.value).toBe(2);
      expect(viewport.clientToContent(400, 200)).toEqual(before);
    });
  });

  describe('zoomBy', () => {
    it('limits the zoom', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.zoomBy(100);

      expect(viewport.zoom.value).toBe(DEFAULT_MAX_ZOOM);
    });

    it('keeps the zoom and the pan after a resize', async () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      viewport.zoomBy(100);
      const pan = {
        ...viewport.pan,
      };

      area.resize({
        left: 0,
        top: 0,
        width: 1920,
        height: 1080,
      });
      await nextTick();

      expect(viewport.pan).toEqual(pan);
    });
  });

  describe('zoomToScale', () => {
    it('zooms until one unit spans the given screen pixels', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.zoomToScale(2);

      expect(viewport.scale.value).toBeCloseTo(2);
    });
  });

  describe('resetZoom', () => {
    it('resets the zoom and fits the content again', async () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      viewport.zoomBy(100);
      area.resize({
        left: 0,
        top: 0,
        width: 1920,
        height: 1080,
      });
      await nextTick();

      viewport.resetZoom();

      expect(viewport.zoom.value).toBe(1);
      expect(viewport.scale.value).toBe(1);
      expect(viewport.pan).toEqual({
        x: 0,
        y: 0,
      });
    });
  });

  describe('zoomToBounds', () => {
    it('fills the area with the bounds', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.zoomToBounds({
        minX: 480,
        maxX: 960,
        minY: 270,
        maxY: 540,
      });

      expect(viewport.scale.value).toBe(2);
      expect(viewport.pan).toEqual({
        x: -960,
        y: -540,
      });
    });

    it('centers bounds without an extent at the current zoom', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.zoomToBounds({
        minX: 100,
        maxX: 100,
        minY: 100,
        maxY: 100,
      });

      expect(viewport.zoom.value).toBe(1);
      expect(viewport.pan).toEqual({
        x: 430,
        y: 220,
      });
    });

    it('limits the zoom to its range', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.zoomToBounds({
        minX: 0,
        maxX: 1,
        minY: 0,
        maxY: 1,
      });

      expect(viewport.zoom.value).toBe(DEFAULT_MAX_ZOOM);
    });
  });

  describe('panBy', () => {
    it('pans by a distance', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      viewport.panBy(10, -5);

      expect(viewport.pan).toEqual({
        x: 10,
        y: -5,
      });
    });
  });

  describe('startPan', () => {
    it('pans with the pointer from the current position', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      viewport.panBy(10, -5);

      viewport.startPan(
        pointer('pointerdown', {
          clientX: 100,
          clientY: 100,
        }),
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 130,
          clientY: 120,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(viewport.pan).toEqual({
        x: 40,
        y: 15,
      });
    });
  });

  describe('clientToArea', () => {
    it('converts a position in the window into one in the area', () => {
      const { viewport } = mount({
        left: 100,
        top: 50,
        width: 960,
        height: 540,
      });

      const point = viewport.clientToArea(110, 60);

      expect(point).toEqual({
        x: 10,
        y: 10,
      });
    });
  });

  describe('clientToContent', () => {
    it('converts a position in the window into one in the content', () => {
      const { viewport } = mount({
        left: 100,
        top: 50,
        width: 960,
        height: 540,
      });

      const point = viewport.clientToContent(580, 320);

      expect(point).toEqual({
        x: 960,
        y: 540,
      });
    });
  });

  describe('contentToArea', () => {
    it('converts a position in the content into one in the area', () => {
      const { viewport } = mount({
        left: 100,
        top: 50,
        width: 960,
        height: 540,
      });

      const point = viewport.contentToArea(960, 540);

      expect(point).toEqual({
        x: 480,
        y: 270,
      });
    });
  });

  describe('pointer', () => {
    it.each([
      {
        name: 'the right button',
        button: 2,
      },
      {
        name: 'the middle button',
        button: 1,
      },
    ])('pans with $name before the content takes the press', ({ button }) => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const child = document.createElement('div');
      area.element.appendChild(child);
      let childPressed = false;
      child.addEventListener('pointerdown', () => {
        childPressed = true;
      });

      child.dispatchEvent(
        pointer('pointerdown', {
          button,
          clientX: 100,
          clientY: 100,
        }),
      );
      const panningDuringPress = viewport.panning.value;
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 120,
          clientY: 90,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(childPressed).toBe(false);
      expect(panningDuringPress).toBe(true);
      expect(viewport.panning.value).toBe(false);
      expect(viewport.pan).toEqual({
        x: 20,
        y: -10,
      });
    });

    it('pans with the left button while Space is held', () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      window.dispatchEvent(key('keydown', 'Space'));

      area.element.dispatchEvent(
        pointer('pointerdown', {
          clientX: 100,
          clientY: 100,
        }),
      );
      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 110,
          clientY: 130,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(viewport.pan).toEqual({
        x: 10,
        y: 30,
      });
    });

    it('leaves the left button to the content without Space', () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      area.element.dispatchEvent(
        pointer('pointerdown', {
          clientX: 100,
          clientY: 100,
        }),
      );

      expect(viewport.panning.value).toBe(false);
    });

    it('keeps the context menu closed on the area', () => {
      const { area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const menu = new MouseEvent('contextmenu', {
        cancelable: true,
        bubbles: true,
      });

      area.element.dispatchEvent(menu);

      expect(menu.defaultPrevented).toBe(true);
    });
  });

  describe('keyboard', () => {
    it('holds Space for panning', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const spaceDown = key('keydown', 'Space');

      window.dispatchEvent(spaceDown);

      expect(viewport.spacePressed.value).toBe(true);
      expect(spaceDown.defaultPrevented).toBe(true);
    });

    it('releases Space when the key goes up', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      window.dispatchEvent(key('keydown', 'Space'));

      window.dispatchEvent(key('keyup', 'Space'));

      expect(viewport.spacePressed.value).toBe(false);
    });

    it('releases Space when the window loses the focus', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      window.dispatchEvent(key('keydown', 'Space'));

      window.dispatchEvent(new Event('blur'));

      expect(viewport.spacePressed.value).toBe(false);
    });

    it('leaves Space to text fields and buttons', () => {
      const { viewport } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const input = document.createElement('input');
      const button = document.createElement('button');
      document.body.append(input, button);
      const typed = key('keydown', 'Space');
      const pressed = key('keydown', 'Space');

      input.dispatchEvent(typed);
      button.dispatchEvent(pressed);

      expect(viewport.spacePressed.value).toBe(false);
      expect(typed.defaultPrevented).toBe(false);
      expect(pressed.defaultPrevented).toBe(false);
      input.remove();
      button.remove();
    });
  });

  describe('wheel', () => {
    it.each([
      {
        modifier: 'Ctrl',
        init: {
          ctrlKey: true,
        },
      },
      {
        modifier: 'Cmd',
        init: {
          metaKey: true,
        },
      },
    ])('zooms with the wheel around the pointer while $modifier is held', ({ init }) => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const before = viewport.clientToContent(200, 100);
      const zooming = wheel({
        deltaY: -100,
        clientX: 200,
        clientY: 100,
        ...init,
      });

      area.element.dispatchEvent(zooming);

      expect(viewport.zoom.value).toBeGreaterThan(1);
      expect(viewport.clientToContent(200, 100).x).toBeCloseTo(before.x);
      expect(viewport.clientToContent(200, 100).y).toBeCloseTo(before.y);
      expect(zooming.defaultPrevented).toBe(true);
    });

    it('pans with the wheel and two fingers on a touchpad', () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });
      const scroll = wheel({
        deltaX: 30,
        deltaY: 50,
      });

      area.element.dispatchEvent(scroll);

      expect(viewport.zoom.value).toBe(1);
      expect(viewport.pan).toEqual({
        x: -30,
        y: -50,
      });
      expect(scroll.defaultPrevented).toBe(true);
    });

    it.each([
      {
        name: 'a vertical',
        deltaX: 0,
        deltaY: 50,
        expected: -50,
      },
      {
        name: 'a horizontal',
        deltaX: 20,
        deltaY: 0,
        expected: -20,
      },
    ])('pans sideways with Shift and $name wheel', ({ deltaX, deltaY, expected }) => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      area.element.dispatchEvent(
        wheel({
          deltaX,
          deltaY,
          shiftKey: true,
        }),
      );

      expect(viewport.pan).toEqual({
        x: expected,
        y: 0,
      });
    });

    it('converts a wheel scrolling by lines into pixels', () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      area.element.dispatchEvent(
        wheel({
          deltaY: 3,
          deltaMode: 1,
        }),
      );

      expect(viewport.pan).toEqual({
        x: 0,
        y: -48,
      });
    });

    it('limits the zoom of one large wheel step', () => {
      const { viewport, area } = mount({
        left: 0,
        top: 0,
        width: 960,
        height: 540,
      });

      area.element.dispatchEvent(
        wheel({
          deltaY: -1000,
          ctrlKey: true,
        }),
      );

      expect(viewport.zoom.value).toBeCloseTo(Math.exp(0.2));
    });
  });
});
