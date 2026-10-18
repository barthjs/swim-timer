import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { EditorCanvasActions } from '@/composables/editor/useEditorShortcuts';
import { useEditorShortcuts } from '@/composables/editor/useEditorShortcuts';
import { useEditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import { useHistory } from '@/composables/useHistory';
import { withSetup } from '@/test/withSetup';

const registered = vi.hoisted(() => ({
  shortcuts: {} as Record<string, () => void>,
}));

vi.mock('@nuxt/ui/composables', () => ({
  defineShortcuts: (shortcuts: Record<string, () => void>) => {
    registered.shortcuts = shortcuts;
  },
}));

describe('useEditorShortcuts', () => {
  let unmount = () => {};

  beforeEach(() => {
    localStorage.clear();
  });
  afterEach(() => unmount());

  /** mount registers the shortcuts of an editor whose actions are spies. */
  function mount(busy = false) {
    const canvas: EditorCanvasActions = {
      zoomBy: vi.fn<(factor: number) => void>(),
      resetZoom: vi.fn<() => void>(),
      zoomToSelection: vi.fn<() => void>(),
    };
    const actions = {
      selectAll: vi.fn<() => void>(),
      pickImage: vi.fn<() => void>(),
      nudge: vi.fn<(dx: number, dy: number) => void>(),
      extra: vi.fn<() => void>(),
    };
    const mounted = withSetup(() => {
      const workspace = useEditorWorkspace({
        storageKey: 'shortcuts-test',
        busy: () => busy,
      });
      const history = useHistory({
        count: 0,
      });
      useEditorShortcuts({
        workspace,
        history,
        canvas: () => canvas,
        selectAll: actions.selectAll,
        pickImage: actions.pickImage,
        nudge: actions.nudge,
        nudgeStep: 1,
        nudgeStepLarge: 10,
        extra: {
          l: actions.extra,
        },
      });
      return workspace;
    });
    unmount = mounted.unmount;
    return {
      workspace: mounted.result,
      canvas,
      actions,
    };
  }

  it('zooms the canvas', () => {
    const { canvas } = mount();

    registered.shortcuts['+']!();

    expect(canvas.zoomBy).toHaveBeenCalledWith(1.25);
  });

  it('switches a view option', () => {
    const { workspace } = mount();

    registered.shortcuts['g']!();

    expect(workspace.view.grid.value).toBe(false);
  });

  it('moves the selected items with the arrow keys', () => {
    const { actions } = mount();

    registered.shortcuts['arrowleft']!();

    expect(actions.nudge).toHaveBeenCalledWith(-1, 0);
  });

  it('moves the selected items farther with Shift', () => {
    const { actions } = mount();

    registered.shortcuts['shift_arrowdown']!();

    expect(actions.nudge).toHaveBeenCalledWith(0, 10);
  });

  it('runs the shortcuts of the editor itself', () => {
    const { actions } = mount();

    registered.shortcuts['l']!();

    expect(actions.extra).toHaveBeenCalledOnce();
  });

  it('opens the help', () => {
    const { workspace } = mount();

    registered.shortcuts['?']!();

    expect(workspace.helpOpen.value).toBe(true);
  });

  it('keeps the items in place during a gesture', () => {
    const { actions } = mount(true);

    registered.shortcuts['arrowleft']!();

    expect(actions.nudge).not.toHaveBeenCalled();
  });

  it('keeps the items in place in the view mode', () => {
    const { workspace, actions } = mount();
    workspace.toggleViewMode();

    registered.shortcuts['arrowleft']!();

    expect(actions.nudge).not.toHaveBeenCalled();
  });
});
