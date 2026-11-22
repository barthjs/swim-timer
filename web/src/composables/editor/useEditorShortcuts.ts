import { defineShortcuts } from '@nuxt/ui/composables';

import type { EditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import type { History } from '@/composables/editor/useHistory';
import { EDITOR_SHORTCUTS } from '@/lib/editor/shortcuts';
import { shortcutDefinition } from '@/lib/ui/shortcuts';

const ZOOM_STEP = 1.25;

/** What the shortcuts do with the canvas, which the canvas component of an editor offers. */
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
  // Moves the selected items by a distance in content units
  nudge: (dx: number, dy: number) => void;
  // The distances the arrow keys move the selected items, with and without Shift
  nudgeStep: number;
  nudgeStepLarge: number;
  // The shortcuts of the editor itself, by the name defineShortcuts expects
  extra: Record<string, () => void>;
};

/**
 * Registers the keyboard shortcuts every editor offers together with those of the editor itself,
 * where actions that change items wait for the end of a gesture and do nothing in the view mode.
 */
export function useEditorShortcuts(options: EditorShortcutOptions): void {
  const { workspace, history } = options;
  const { view, whenEditing, whenIdle, toggle } = workspace;

  defineShortcuts({
    [shortcutKey('undo')]: whenIdle(() => history.undo()),
    [shortcutKey('redo')]: whenIdle(() => history.redo()),
    [shortcutKey('redoAlternative')]: whenIdle(() => history.redo()),
    [shortcutKey('selectAll')]: whenEditing(options.selectAll),
    [shortcutKey('zoomIn')]: () => options.canvas()?.zoomBy(ZOOM_STEP),
    [shortcutKey('zoomOut')]: () => options.canvas()?.zoomBy(1 / ZOOM_STEP),
    [shortcutKey('resetZoom')]: () => options.canvas()?.resetZoom(),
    [shortcutKey('zoomToSelection')]: () => options.canvas()?.zoomToSelection(),
    [shortcutKey('grid')]: () => toggle(view.grid),
    [shortcutKey('guides')]: () => toggle(view.guides),
    [shortcutKey('rulers')]: () => toggle(view.rulers),
    [shortcutKey('loupe')]: () => toggle(view.loupe),
    [shortcutKey('preview')]: () => toggle(view.preview),
    [shortcutKey('snapping')]: () => toggle(view.snapping),
    [shortcutKey('image')]: () => options.pickImage(),
    [shortcutKey('viewMode')]: () => workspace.toggleViewMode(),
    [shortcutKey('sidebar')]: () => toggle(workspace.sidebarOpen),
    [shortcutKey('fullscreen')]: () => void workspace.fullscreen.toggle(),
    [shortcutKey('help')]: () => workspace.openHelp(),
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

/** Returns the name of an editor shortcut for defineShortcuts. */
function shortcutKey(shortcut: keyof typeof EDITOR_SHORTCUTS): string {
  return shortcutDefinition(EDITOR_SHORTCUTS[shortcut]);
}
