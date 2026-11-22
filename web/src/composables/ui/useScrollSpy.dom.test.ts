import { ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { SCROLL_SPY_ATTRIBUTE, useScrollSpy } from '@/composables/ui/useScrollSpy';
import { withSetup } from '@/test/withSetup';

/** The height of the view of the tests in pixels. */
const VIEW_HEIGHT = 1000;

/** Returns an element in the document with a marked child per section at the given tops. */
function container(tops: Record<string, number>): HTMLElement {
  const root = document.createElement('div');
  for (const [section, top] of Object.entries(tops)) {
    const element = document.createElement('section');
    element.setAttribute(SCROLL_SPY_ATTRIBUTE, section);
    element.getBoundingClientRect = () =>
      DOMRect.fromRect({
        y: top,
        height: 400,
      });
    element.scrollIntoView = vi.fn<(options?: ScrollIntoViewOptions | boolean) => void>();
    root.append(element);
  }
  document.body.append(root);
  return root;
}

/** Returns the marked child of a section. */
function child(root: HTMLElement, section: string): Element {
  return root.querySelector(`[${SCROLL_SPY_ATTRIBUTE}="${section}"]`)!;
}

/** Sets how far the document is scrolled and reports the scroll. */
function scrollPage(scrollTop: number, scrollHeight = 5000): void {
  const page = document.scrollingElement as HTMLElement;
  Object.defineProperty(page, 'scrollTop', {
    configurable: true,
    value: scrollTop,
  });
  Object.defineProperty(page, 'scrollHeight', {
    configurable: true,
    value: scrollHeight,
  });
  window.dispatchEvent(new Event('scroll'));
}

describe('useScrollSpy', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    Object.defineProperty(document.scrollingElement as HTMLElement, 'clientHeight', {
      configurable: true,
      value: VIEW_HEIGHT,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    document.body.replaceChildren();
  });

  describe('onActive', () => {
    it('reports the last section whose top reached the place a jump puts it', () => {
      const root = container({
        a: -500,
        b: 0,
        c: 700,
      });
      const onActive = vi.fn<(section: string) => void>();
      withSetup(() => useScrollSpy(ref(root), ['a', 'b', 'c'], onActive));

      scrollPage(800);

      expect(onActive).toHaveBeenLastCalledWith('b');
    });

    it('reports the last section at the end of the page', () => {
      const root = container({
        a: -500,
        b: 0,
        c: 700,
      });
      const onActive = vi.fn<(section: string) => void>();
      withSetup(() => useScrollSpy(ref(root), ['a', 'b', 'c'], onActive));

      scrollPage(4000);

      expect(onActive).toHaveBeenLastCalledWith('c');
    });
  });

  describe('scrollTo', () => {
    it('scrolls the section into view', () => {
      const root = container({
        a: 0,
        b: 600,
      });
      const { result } = withSetup(() =>
        useScrollSpy(ref(root), ['a', 'b'], vi.fn<(section: string) => void>()),
      );

      result.scrollTo('b');

      expect(child(root, 'b').scrollIntoView).toHaveBeenCalledWith({
        behavior: 'smooth',
        block: 'start',
      });
    });

    it('ignores the scrolling of the jump', () => {
      const root = container({
        a: 0,
        b: 600,
      });
      const onActive = vi.fn<(section: string) => void>();
      const { result } = withSetup(() => useScrollSpy(ref(root), ['a', 'b'], onActive));

      result.scrollTo('b');
      scrollPage(300);

      expect(onActive).not.toHaveBeenCalled();
    });

    it('follows the page again once the jump ended', () => {
      const root = container({
        a: -500,
        b: 0,
      });
      const onActive = vi.fn<(section: string) => void>();
      const { result } = withSetup(() => useScrollSpy(ref(root), ['a', 'b'], onActive));
      result.scrollTo('b');
      window.dispatchEvent(new Event('scrollend'));

      scrollPage(800);

      expect(onActive).toHaveBeenLastCalledWith('b');
    });
  });
});
