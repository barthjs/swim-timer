/** Names the keys of each shortcut, as the keys a tooltip or the help shows. */
export type ShortcutMap = Record<string, readonly string[]>;

/** The keyboard shortcuts every editor offers, leaving the digits and arrow keys to its items. */
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
 * Returns the keys of a shortcut as a new array,
 * the form a UTooltip and the keyLabels of useShortcutLabels take.
 */
export function shortcutKeys(keys: readonly string[]): string[] {
  return [...keys];
}
