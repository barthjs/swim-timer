/** ShortcutMap names the keys of each shortcut, as the keys a tooltip or the help shows. */
export type ShortcutMap = Record<string, readonly string[]>;

/**
 * EDITOR_SHORTCUTS are the keyboard shortcuts every editor offers. Single keys do not fire while an input has
 * the focus. The digits and the arrow keys are left to the items of an editor, so the actions use letters
 * and symbols.
 */
export const EDITOR_SHORTCUTS = {
  undo: ['meta', 'z'],
  redo: ['meta', 'y'],
  redoAlternative: ['meta', 'shift', 'z'],
  save: ['meta', 's'],
  selectAll: ['meta', 'a'],
  zoomIn: ['+'],
  zoomOut: ['-'],
  resetZoom: ['shift', 'f'],
  zoomToSelection: ['z'],
  grid: ['g'],
  guides: ['h'],
  rulers: ['r'],
  loupe: ['o'],
  preview: ['p'],
  snapping: ['m'],
  image: ['i'],
  viewMode: ['v'],
  sidebar: ['s'],
  fullscreen: ['f'],
  help: ['?'],
} as const satisfies ShortcutMap;

export type EditorShortcut = keyof typeof EDITOR_SHORTCUTS;

/**
 * shortcutKeys returns the keys of a shortcut as a new array,
 * the form a UTooltip and the keyLabels of useShortcutLabels take.
 */
export function shortcutKeys(keys: readonly string[]): string[] {
  return [...keys];
}

/**
 * shortcutDefinition joins the keys of a shortcut into the name defineShortcuts of Nuxt UI expects,
 * such as "meta_z" or "shift_f".
 */
export function shortcutDefinition(keys: readonly string[]): string {
  return keys.join('_');
}
