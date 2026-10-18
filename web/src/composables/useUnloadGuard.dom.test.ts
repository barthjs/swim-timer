import { describe, expect, it } from 'vitest';

import { useUnloadGuard } from '@/composables/useUnloadGuard';
import { withSetup } from '@/test/withSetup';

/**
 * dispatchUnload fires a cancelable beforeunload event and reports whether a listener prevented it.
 */
function dispatchUnload(): boolean {
  const event = new Event('beforeunload', {
    cancelable: true,
  });
  window.dispatchEvent(event);
  return event.defaultPrevented;
}

describe('useUnloadGuard', () => {
  it('warns while there are changes', () => {
    const { unmount } = withSetup(() => useUnloadGuard(() => true));

    const prevented = dispatchUnload();
    unmount();

    expect(prevented).toBe(true);
  });

  it('lets the page unload without changes', () => {
    const { unmount } = withSetup(() => useUnloadGuard(() => false));

    const prevented = dispatchUnload();
    unmount();

    expect(prevented).toBe(false);
  });

  it('stops warning once the component unmounts', () => {
    const { unmount } = withSetup(() => useUnloadGuard(() => true));
    unmount();

    const prevented = dispatchUnload();

    expect(prevented).toBe(false);
  });
});
