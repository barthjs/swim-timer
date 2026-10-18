import { nextTick, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { useEditorGuides, type GuideView } from '@/composables/editor/useEditorGuides';
import { pointer } from '@/test/pointer';
import { withSetup } from '@/test/withSetup';

// A canvas whose area and content share their coordinates, with a 100 x 100 content.
const VIEW: GuideView = {
  content: {
    width: 100,
    height: 100,
  },
  toArea: (clientX, clientY) => ({
    x: clientX,
    y: clientY,
  }),
  toContent: (clientX, clientY) => ({
    x: clientX,
    y: clientY,
  }),
};

describe('useEditorGuides', () => {
  let unmount = () => {};

  beforeEach(() => localStorage.clear());
  afterEach(() => {
    window.dispatchEvent(pointer('pointerup'));
    unmount();
  });

  /** mount creates the guides of a document whose name the test can change. */
  function mount(name: string | null = 'first') {
    const document = ref<string | null>(name);
    const mounted = withSetup(() =>
      useEditorGuides({
        storageKey: 'guides-test',
        document: () => document.value,
      }),
    );
    unmount = mounted.unmount;
    return {
      guides: mounted.result,
      document,
    };
  }

  /** place presses on a ruler at a point and drags the new guide to another point. */
  function place(
    guides: ReturnType<typeof mount>['guides'],
    ruler: 'horizontal' | 'vertical',
    to: { x: number; y: number },
  ): void {
    guides.startFromRuler(
      pointer('pointerdown', {
        clientX: to.x,
        clientY: to.y,
      }),
      ruler,
      VIEW,
    );
    window.dispatchEvent(
      pointer('pointerup', {
        clientX: to.x,
        clientY: to.y,
      }),
    );
  }

  describe('persist', () => {
    it('keeps the guides of a document under its name', async () => {
      const { guides, document } = mount(null);
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });

      guides.persist('saved');
      document.value = 'saved';
      await nextTick();

      expect(guides.lines.value).toHaveLength(1);
    });

    it('shows the guides kept for another document when it opens', async () => {
      const { guides, document } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });
      await nextTick();

      document.value = 'second';
      await nextTick();

      expect(guides.lines.value).toEqual([]);
    });
  });

  describe('remove', () => {
    it('deletes the guides of a deleted document', async () => {
      const { guides } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });
      await nextTick();

      guides.remove('first');

      expect(localStorage.getItem('guides-test.guides.first')).toBeNull();
    });
  });

  describe('targets', () => {
    it('returns the guides as lines to snap to', () => {
      const { guides } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });

      const targets = guides.targets();

      expect(targets).toEqual({
        x: [40],
        y: [],
      });
    });
  });

  describe('startFromRuler', () => {
    it('places a vertical guide from the ruler at the top', () => {
      const { guides } = mount();

      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });

      expect(guides.lines.value).toEqual([
        {
          axis: 'x',
          value: 40,
        },
      ]);
    });

    it('places a horizontal guide from the ruler on the left', () => {
      const { guides } = mount();

      place(guides, 'vertical', {
        x: 50,
        y: 30,
      });

      expect(guides.lines.value).toEqual([
        {
          axis: 'y',
          value: 30,
        },
      ]);
    });

    it('removes a guide released on its ruler', () => {
      const { guides } = mount();

      place(guides, 'horizontal', {
        x: 40,
        y: 5,
      });

      expect(guides.lines.value).toEqual([]);
    });

    it('removes a guide released outside the content', () => {
      const { guides } = mount();

      place(guides, 'vertical', {
        x: 50,
        y: 140,
      });

      expect(guides.lines.value).toEqual([]);
    });
  });

  describe('startMove', () => {
    it('lets a guide follow the pointer', () => {
      const { guides } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });
      guides.startMove(pointer('pointerdown'), 0, VIEW);

      window.dispatchEvent(
        pointer('pointermove', {
          clientX: 70,
          clientY: 50,
        }),
      );

      expect(guides.lines.value[0]!.value).toBe(70);
    });
  });

  describe('removeAt', () => {
    it('removes one guide', () => {
      const { guides } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });

      guides.removeAt(0);

      expect(guides.lines.value).toEqual([]);
    });
  });

  describe('clear', () => {
    it('removes every guide', () => {
      const { guides } = mount();
      place(guides, 'horizontal', {
        x: 40,
        y: 50,
      });
      place(guides, 'vertical', {
        x: 50,
        y: 30,
      });

      guides.clear();

      expect(guides.lines.value).toEqual([]);
    });
  });
});
