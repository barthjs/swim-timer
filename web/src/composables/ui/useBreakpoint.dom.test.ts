import { afterEach, describe, expect, it, vi } from 'vitest';

import { useBreakpoint } from '@/composables/ui/useBreakpoint';
import { withSetup } from '@/test/withSetup';

/** Replaces matchMedia with a query whose match state a test can change. */
function fakeMediaQuery(matches: boolean) {
  const listeners = new Set<() => void>();
  const query = {
    matches,
    media: '',
    addEventListener: (_type: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_type: string, listener: () => void) => listeners.delete(listener),
    change(next: boolean) {
      query.matches = next;
      listeners.forEach((listener) => listener());
    },
    listeners,
  };
  vi.stubGlobal('matchMedia', (media: string) => {
    query.media = media;
    return query;
  });
  return query;
}

describe('useBreakpoint', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('matches the media query once mounted', () => {
    fakeMediaQuery(true);

    const { result: reached } = withSetup(() => useBreakpoint('md'));

    expect(reached.value).toBe(true);
  });

  it('queries the minimum width of the breakpoint', () => {
    const query = fakeMediaQuery(false);

    withSetup(() => useBreakpoint('sm'));

    expect(query.media).toBe('(min-width: 640px)');
  });

  it('follows a change of the media query', () => {
    const query = fakeMediaQuery(true);
    const { result: reached } = withSetup(() => useBreakpoint('md'));

    query.change(false);

    expect(reached.value).toBe(false);
  });

  it('stops listening once unmounted', () => {
    const query = fakeMediaQuery(true);
    const { unmount } = withSetup(() => useBreakpoint('md'));

    unmount();

    expect(query.listeners.size).toBe(0);
  });
});
