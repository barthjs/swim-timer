import { afterEach, describe, expect, it } from 'vitest';

import type { Bounds, Point } from '@/lib/editor/geometry';
import { pointer } from '@/test/pointer';
import { withSetup } from '@/test/withSetup';
import { useEditorSelection } from '@/composables/editor/useEditorSelection';
import { marqueeBounds, useMarqueeSelection } from '@/composables/editor/useMarqueeSelection';

// Three items stacked from top to bottom, each 100 wide and 50 high with a gap of 50.
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
  [
    'b',
    {
      minX: 0,
      maxX: 100,
      minY: 100,
      maxY: 150,
    },
  ],
  [
    'c',
    {
      minX: 0,
      maxX: 100,
      minY: 200,
      maxY: 250,
    },
  ],
];

describe('marqueeBounds', () => {
  it('spans the area independent of the drawing direction', () => {
    const area = {
      start: {
        x: 50,
        y: 80,
      },
      current: {
        x: 10,
        y: 20,
      },
    };

    const bounds = marqueeBounds(area);

    expect(bounds).toEqual({
      minX: 10,
      maxX: 50,
      minY: 20,
      maxY: 80,
    });
  });
});

describe('useMarqueeSelection', () => {
  let unmount = () => {};

  afterEach(() => unmount());

  /** mount creates the marquee selection over the three items, with client pixels as content units. */
  function mount() {
    const mounted = withSetup(() => {
      const selection = useEditorSelection(() => ITEMS.map(([key]) => key));
      const marquee = useMarqueeSelection({
        toContent: (clientX, clientY) => ({
          x: clientX,
          y: clientY,
        }),
        items: () => ITEMS,
        selection,
      });
      return {
        selection,
        marquee,
      };
    });
    unmount = mounted.unmount;
    return mounted.result;
  }

  /** drag draws a marquee from one point to another and keeps the pointer pressed. */
  function drag(
    marquee: ReturnType<typeof mount>['marquee'],
    from: Point,
    to: Point,
    init: PointerEventInit = {},
  ): void {
    marquee.startMarquee(
      pointer('pointerdown', {
        clientX: from.x,
        clientY: from.y,
        ...init,
      }),
    );
    window.dispatchEvent(
      pointer('pointermove', {
        clientX: to.x,
        clientY: to.y,
        ...init,
      }),
    );
  }

  describe('startMarquee', () => {
    it('shows the marquee while the pointer is pressed', () => {
      const { marquee } = mount();

      marquee.startMarquee(
        pointer('pointerdown', {
          clientX: 400,
          clientY: 140,
        }),
      );

      expect(marquee.marquee.value).not.toBeNull();
    });

    it('selects every item the rectangle touches', () => {
      const { selection, marquee } = mount();

      drag(
        marquee,
        {
          x: 50,
          y: 40,
        },
        {
          x: 150,
          y: 120,
        },
      );

      expect([...selection.selected.value].sort()).toEqual(['a', 'b']);
    });

    it('changes the selection while the rectangle shrinks', () => {
      const { selection, marquee } = mount();
      drag(
        marquee,
        {
          x: 50,
          y: 40,
        },
        {
          x: 150,
          y: 120,
        },
      );

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 150,
          clientY: 60,
        }),
      );

      expect([...selection.selected.value]).toEqual(['a']);
    });

    it('removes the marquee on release', () => {
      const { marquee } = mount();
      marquee.startMarquee(pointer('pointerdown'));

      window.dispatchEvent(pointer('pointerup'));

      expect(marquee.marquee.value).toBeNull();
    });

    it.each([
      {
        modifier: 'Shift',
        init: {
          shiftKey: true,
        },
      },
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
    ])('keeps and extends the selection with $modifier', ({ init }) => {
      const { selection, marquee } = mount();
      selection.set(['c']);

      drag(
        marquee,
        {
          x: 50,
          y: 40,
        },
        {
          x: 150,
          y: 60,
        },
        init,
      );

      expect([...selection.selected.value].sort()).toEqual(['a', 'c']);
    });

    it('clears the selection on a plain click', () => {
      const { selection, marquee } = mount();
      selection.set(['b']);

      marquee.startMarquee(
        pointer('pointerdown', {
          clientX: 300,
          clientY: 300,
        }),
      );
      window.dispatchEvent(pointer('pointerup'));

      expect(selection.selected.value.size).toBe(0);
    });

    it('ignores other buttons', () => {
      const { selection, marquee } = mount();
      selection.set(['b']);

      marquee.startMarquee(
        pointer('pointerdown', {
          button: 2,
        }),
      );

      expect([...selection.selected.value]).toEqual(['b']);
    });
  });
});
