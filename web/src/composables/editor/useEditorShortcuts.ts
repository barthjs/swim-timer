import { defineShortcuts } from '@nuxt/ui/composables';

import type { EditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import type { History } from '@/composables/useHistory';
import { EDITOR_SHORTCUTS, shortcutDefinition } from '@/lib/editor/shortcuts';

// The zoom factor of one zoom step.
const ZOOM_STEP = 1.25;

/** EditorCanvasActions are what the shortcuts do with the canvas, which the canvas component of an editor offers. */
export type EditorCanvasActions = {
  zoomBy: (factor: number) => void;
  resetZoom: () => void;
  zoomToSelection: () => void;
};

export type EditorShortcutOptions = {
  workspace: EditorWorkspace;
  history: History;
  canvas: () => EditorCanvasActions | null | undefined;
  selectAll: () => void;
  pickImage: () => void;
  // Moves the selected items by a distance in content units.
  nudge: (dx: number, dy: number) => void;
  // The distances the arrow keys move the selected items, with and without Shift.
  nudgeStep: number;
  nudgeStepLarge: number;
  // The shortcuts of the editor itself, by the name defineShortcuts expects.
  extra: Record<string, () => void>;
};

/**
 * useEditorShortcuts registers the keyboard shortcuts every editor offers, together with those of the editor itself.
 * Actions that change items wait for the end of a gesture and do nothing in the view mode, undo and redo only wait
 * for the end of a gesture.
 */
export function useEditorShortcuts(options: EditorShortcutOptions): void {
  const { workspace, history } = options;
  const { view, whenEditing, whenIdle, toggle } = workspace;

  /** key returns the name of an editor shortcut for defineShortcuts. */
  function key(shortcut: keyof typeof EDITOR_SHORTCUTS): string {
    return shortcutDefinition(EDITOR_SHORTCUTS[shortcut]);
  }

  defineShortcuts({
    [key('undo')]: whenIdle(() => history.undo()),
    [key('redo')]: whenIdle(() => history.redo()),
    [key('redoAlternative')]: whenIdle(() => history.redo()),
    [key('selectAll')]: whenEditing(options.selectAll),
    [key('zoomIn')]: () => options.canvas()?.zoomBy(ZOOM_STEP),
    [key('zoomOut')]: () => options.canvas()?.zoomBy(1 / ZOOM_STEP),
    [key('resetZoom')]: () => options.canvas()?.resetZoom(),
    [key('zoomToSelection')]: () => options.canvas()?.zoomToSelection(),
    [key('grid')]: () => toggle(view.grid),
    [key('guides')]: () => toggle(view.guides),
    [key('rulers')]: () => toggle(view.rulers),
    [key('loupe')]: () => toggle(view.loupe),
    [key('preview')]: () => toggle(view.preview),
    [key('snapping')]: () => toggle(view.snapping),
    [key('image')]: () => options.pickImage(),
    [key('viewMode')]: () => workspace.toggleViewMode(),
    [key('sidebar')]: () => toggle(workspace.sidebarOpen),
    [key('fullscreen')]: () => void workspace.fullscreen.toggle(),
    [key('help')]: () => workspace.openHelp(),
    arrowup: whenEditing(() => options.nudge(0, -options.nudgeStep)),
    arrowdown: whenEditing(() => options.nudge(0, options.nudgeStep)),
    arrowleft: whenEditing(() => options.nudge(-options.nudgeStep, 0)),
    arrowright: whenEditing(() => options.nudge(options.nudgeStep, 0)),
    shift_arrowup: whenEditing(() => options.nudge(0, -options.nudgeStepLarge)),
    shift_arrowdown: whenEditing(() => options.nudge(0, options.nudgeStepLarge)),
    shift_arrowleft: whenEditing(() => options.nudge(-options.nudgeStepLarge, 0)),
    shift_arrowright: whenEditing(() => options.nudge(options.nudgeStepLarge, 0)),
    ...options.extra,
  });
}
