import type { ShortcutMap } from '@/lib/editor/shortcuts';

/**
 * PRESET_EDITOR_SHORTCUTS are the keyboard shortcuts of the preset editor beside those every editor offers.
 * Tab steps through the lanes while no control has the focus, see useEditorWorkspace.
 */
export const PRESET_EDITOR_SHORTCUTS = {
  nextLane: ['tab'],
  previousLane: ['shift', 'tab'],
  lock: ['l'],
  align: ['a'],
} as const satisfies ShortcutMap;

export type PresetEditorShortcut = keyof typeof PRESET_EDITOR_SHORTCUTS;
