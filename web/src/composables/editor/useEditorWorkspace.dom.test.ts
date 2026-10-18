import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import {
  useEditorWorkspace,
  type EditorWorkspaceOptions,
} from '@/composables/editor/useEditorWorkspace';
import { withSetup } from '@/test/withSetup';

const STORAGE_KEY = 'workspace-test';

// The components of the running test, unmounted after it
let unmounts: Array<() => void> = [];

/** mountWorkspace creates the workspace inside a mounted component, which is unmounted after the test. */
function mountWorkspace(options: Partial<EditorWorkspaceOptions> = {}) {
  const mounted = withSetup(() =>
    useEditorWorkspace({
      storageKey: STORAGE_KEY,
      ...options,
    }),
  );
  let unmounted = false;
  // unmount unmounts the component once, so a test can unmount it early and the cleanup still runs
  const unmount = () => {
    if (unmounted) return;
    unmounted = true;
    mounted.unmount();
  };
  unmounts.push(unmount);
  return {
    result: mounted.result,
    unmount,
  };
}

/** pressKey dispatches a keydown on the document and reports whether a listener prevented it. */
function pressKey(key: string, init: KeyboardEventInit = {}): boolean {
  const event = new KeyboardEvent('keydown', {
    key,
    cancelable: true,
    ...init,
  });
  document.dispatchEvent(event);
  return event.defaultPrevented;
}

describe('useEditorWorkspace', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    for (const unmount of unmounts) unmount();
    unmounts = [];
  });

  describe('sidebarOpen', () => {
    it('shows the sidebar by default', () => {
      const { result } = mountWorkspace();

      const open = result.sidebarOpen.value;

      expect(open).toBe(true);
    });
  });

  describe('view', () => {
    it('shows the aids by default and leaves the preview off', () => {
      const { result } = mountWorkspace();

      const view = Object.fromEntries(
        Object.entries(result.view).map(([name, option]) => [name, option.value]),
      );

      expect(view).toEqual({
        grid: true,
        guides: true,
        rulers: true,
        loupe: true,
        preview: false,
        snapping: true,
      });
    });

    it('keeps a view option under the prefix of the editor', async () => {
      const { result } = mountWorkspace();

      result.toggle(result.view.loupe);
      await nextTick();

      expect(localStorage.getItem(`${STORAGE_KEY}.loupe`)).toBe('false');
    });
  });

  describe('storedOption', () => {
    it('keeps a view option under the prefix of the editor', async () => {
      const { result } = mountWorkspace();
      const grid = result.storedOption('grid', true);

      result.toggle(grid);
      await nextTick();

      expect(localStorage.getItem(`${STORAGE_KEY}.grid`)).toBe('false');
    });
  });

  describe('toggle', () => {
    it('keeps a hidden sidebar in the browser', async () => {
      const first = mountWorkspace();
      first.result.toggle(first.result.sidebarOpen);
      await nextTick();
      first.unmount();

      const second = mountWorkspace();

      expect(second.result.sidebarOpen.value).toBe(false);
    });
  });

  describe('openHelp', () => {
    it('opens the help', () => {
      const { result } = mountWorkspace();

      result.openHelp();

      expect(result.helpOpen.value).toBe(true);
    });
  });

  describe('toggleViewMode', () => {
    it('runs the hook only when the view mode starts', () => {
      const onEnterViewMode = vi.fn<() => void>();
      const { result } = mountWorkspace({
        onEnterViewMode,
      });

      result.toggleViewMode();
      result.toggleViewMode();

      expect(result.viewMode.value).toBe(false);
      expect(onEnterViewMode).toHaveBeenCalledOnce();
    });
  });

  describe('whenEditing', () => {
    it('runs the action outside the view mode', () => {
      const action = vi.fn<() => void>();
      const { result } = mountWorkspace();
      const guarded = result.whenEditing(action);

      guarded();

      expect(action).toHaveBeenCalledOnce();
    });

    it('blocks the action in the view mode', () => {
      const action = vi.fn<() => void>();
      const { result } = mountWorkspace();
      const guarded = result.whenEditing(action);
      result.toggleViewMode();

      guarded();

      expect(action).not.toHaveBeenCalled();
    });

    it('blocks the action during a gesture', () => {
      const action = vi.fn<() => void>();
      const { result } = mountWorkspace({
        busy: () => true,
      });
      const guarded = result.whenEditing(action);

      guarded();

      expect(action).not.toHaveBeenCalled();
    });
  });

  describe('whenIdle', () => {
    it('runs the action also in the view mode', () => {
      const action = vi.fn<() => void>();
      const { result } = mountWorkspace();
      const guarded = result.whenIdle(action);
      result.toggleViewMode();

      guarded();

      expect(action).toHaveBeenCalledOnce();
    });

    it('blocks the action during a gesture', () => {
      const action = vi.fn<() => void>();
      const { result } = mountWorkspace({
        busy: () => true,
      });
      const guarded = result.whenIdle(action);

      guarded();

      expect(action).not.toHaveBeenCalled();
    });
  });

  describe('Escape', () => {
    it('runs the escape hook without preventing the default, so dialogs still close', () => {
      const onEscape = vi.fn<() => void>();
      mountWorkspace({
        onEscape,
      });

      const prevented = pressKey('Escape');

      expect(prevented).toBe(false);
      expect(onEscape).toHaveBeenCalledOnce();
    });
  });

  describe('Tab', () => {
    it('moves to the next item with Tab while no control has the focus', () => {
      const onTab = vi.fn<(step: 1 | -1) => void>();
      mountWorkspace({
        onTab,
      });

      const prevented = pressKey('Tab');

      expect(prevented).toBe(true);
      expect(onTab.mock.calls).toEqual([[1]]);
    });

    it('moves to the previous item with Shift and Tab', () => {
      const onTab = vi.fn<(step: 1 | -1) => void>();
      mountWorkspace({
        onTab,
      });

      pressKey('Tab', {
        shiftKey: true,
      });

      expect(onTab.mock.calls).toEqual([[-1]]);
    });

    it('leaves Tab to the controls of the page while one has the focus', () => {
      const onTab = vi.fn<(step: 1 | -1) => void>();
      mountWorkspace({
        onTab,
      });
      const input = document.createElement('input');
      document.body.append(input);
      input.focus();

      const prevented = pressKey('Tab');

      expect(prevented).toBe(false);
      expect(onTab).not.toHaveBeenCalled();
      input.remove();
    });

    it('leaves Tab alone in the view mode', () => {
      const onTab = vi.fn<(step: 1 | -1) => void>();
      const { result } = mountWorkspace({
        onTab,
      });
      result.toggleViewMode();

      const prevented = pressKey('Tab');

      expect(prevented).toBe(false);
      expect(onTab).not.toHaveBeenCalled();
    });
  });
});
