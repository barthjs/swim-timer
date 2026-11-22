import { onBeforeUnmount, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';

import { activeSection } from '@/lib/ui/scrollSpy';

/** Marks an element the scroll spy follows, its value names the section. */
export const SCROLL_SPY_ATTRIBUTE = 'data-scroll-section';

/** How far in pixels below the place a jump puts a section its top may still count as reached. */
const LINE_TOLERANCE = 8;

/** How close to the end of the page in pixels counts as the end. */
const END_TOLERANCE = 2;

/** Ends a jump in a browser that reports no end of the scroll. */
const JUMP_TIMEOUT_MS = 1000;

/**
 * Reports the section a reader is at among the marked elements of the container while the page scrolls,
 * and scrolls to a section on request. A jump keeps the chosen section, also one at the end that cannot reach the top.
 */
export function useScrollSpy<T extends string>(
  container: Ref<HTMLElement | null>,
  sections: MaybeRefOrGetter<readonly T[]>,
  onActive: (section: T) => void,
) {
  let scroller: HTMLElement | null = null;
  let jumping = false;
  let jumpTimer: ReturnType<typeof setTimeout> | undefined;

  /** Reports the section at the line of the view, unless a jump is running. */
  function update(): void {
    if (jumping || !scroller || !container.value) return;
    const viewTop =
      scroller === document.scrollingElement ? 0 : scroller.getBoundingClientRect().top;
    const tops = toValue(sections).map((section) => ({
      section,
      top: (element(section)?.getBoundingClientRect().top ?? Number.POSITIVE_INFINITY) - viewTop,
    }));
    const atEnd =
      scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - END_TOLERANCE;
    const active = activeSection(tops, activeLine(), atEnd);
    if (active) onActive(active);
  }

  /** Lets the scroll spy follow the page again. */
  function endJump(): void {
    jumping = false;
    clearTimeout(jumpTimer);
  }

  /** Follows the scrolling ancestor of the container, or stops without a container. */
  function listen(root: HTMLElement | null): void {
    if (scroller) {
      scrollTarget(scroller).removeEventListener('scroll', update);
      scrollTarget(scroller).removeEventListener('scrollend', endJump);
    }
    scroller = root ? scrollParent(root) : null;
    if (!scroller) return;
    scrollTarget(scroller).addEventListener('scroll', update, {
      passive: true,
    });
    scrollTarget(scroller).addEventListener('scrollend', endJump);
  }

  watch(container, listen, {
    flush: 'post',
    immediate: true,
  });
  onBeforeUnmount(() => {
    listen(null);
    clearTimeout(jumpTimer);
  });

  /**
   * Returns the distance from the top of the view at which a section becomes the current one,
   * the place a jump puts it, which its scroll margin keeps below a sticky header.
   */
  function activeLine(): number {
    const first = container.value?.querySelector(`[${SCROLL_SPY_ATTRIBUTE}]`);
    const margin = first ? Number.parseFloat(getComputedStyle(first).scrollMarginTop) || 0 : 0;
    return margin + LINE_TOLERANCE;
  }

  /** Returns the marked element of a section. */
  function element(section: T): Element | null | undefined {
    return container.value?.querySelector(`[${SCROLL_SPY_ATTRIBUTE}="${section}"]`);
  }

  /** Scrolls the section to the top of the view, smoothly unless the page just opened. */
  function scrollTo(section: T, smooth = true): void {
    const target = element(section);
    if (!target) return;
    jumping = true;
    clearTimeout(jumpTimer);
    jumpTimer = setTimeout(endJump, JUMP_TIMEOUT_MS);
    target.scrollIntoView({
      behavior: smooth ? 'smooth' : 'auto',
      block: 'start',
    });
  }

  return {
    scrollTo,
  };
}

/** Returns where the scroll events of a scroller arrive. */
function scrollTarget(scroller: HTMLElement): HTMLElement | Window {
  return scroller === document.scrollingElement ? window : scroller;
}

/** Returns the nearest ancestor that can scroll, or the document. */
function scrollParent(element: HTMLElement): HTMLElement {
  // The height is not checked, since the content may only grow after the container is mounted
  for (let parent = element.parentElement; parent; parent = parent.parentElement) {
    const { overflowY } = getComputedStyle(parent);
    if (overflowY === 'auto' || overflowY === 'scroll') return parent;
  }
  const root = document.scrollingElement;
  return root instanceof HTMLElement ? root : document.documentElement;
}
