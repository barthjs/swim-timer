import { ref, type Ref } from 'vue';
import { useEventListener, useFullscreen, useLocalStorage } from '@vueuse/core';

export type EditorWorkspaceOptions = {
  // The prefix of the keys the browser keeps the view options under, such as "editor".
  storageKey: string;
  // Runs when the view mode starts, for example to clear the selection.
  onEnterViewMode?: () => void;
  // Runs on Escape, for example to clear the selection.
  onEscape?: () => void;
  // Runs on Tab while no control has the focus, with 1 for the next and -1 for the previous item.
  onTab?: (step: 1 | -1) => void;
  // Reports a running gesture such as a drag, during which a key must not change the items under the pointer.
  busy?: () => boolean;
};

/** EditorViewOptions are the display aids of an editor, which the toolbar switches and the browser keeps. */
export type EditorViewOptions = {
  grid: Ref<boolean>;
  guides: Ref<boolean>;
  rulers: Ref<boolean>;
  loupe: Ref<boolean>;
  preview: Ref<boolean>;
  snapping: Ref<boolean>;
};

export type EditorWorkspace = ReturnType<typeof useEditorWorkspace>;

/**
 * useEditorWorkspace holds the state of an editor page that is not part of the edited data:
 * the view options kept in the browser, the sidebar beside the canvas, the view mode, the fullscreen mode,
 * and the help. It also handles Escape and Tab, which defineShortcuts cannot handle as the editor needs.
 */
export function useEditorWorkspace(options: EditorWorkspaceOptions) {
  // The cards beside the canvas can be hidden to give the canvas the whole width
  const sidebarOpen = storedOption('sidebar', true);

  // The view mode shows the content without editing aids and blocks editing.
  // It leaves the stored view options untouched, so leaving it brings them back.
  const viewMode = ref(false);

  const helpOpen = ref(false);

  const view: EditorViewOptions = {
    grid: storedOption('grid', true),
    guides: storedOption('guides', true),
    rulers: storedOption('rulers', true),
    loupe: storedOption('loupe', true),
    preview: storedOption('preview', false),
    snapping: storedOption('snapping', true),
  };

  // The fullscreen mode puts the whole document into fullscreen and the page lays the workspace over itself,
  // so the header and the navigation are hidden. Fullscreen on the workspace element itself would hide
  // everything Nuxt UI renders into the body, such as tooltips, dropdowns, dialogs, and toasts.
  // The browser ends the mode on Escape, and leaving the page ends it as well.
  const fullscreen = useFullscreen(undefined, {
    autoExit: true,
  });

  /**
   * storedOption returns a view option the browser keeps, since view options are a matter of taste
   * and not part of the edited data.
   */
  function storedOption(name: string, initial: boolean): Ref<boolean> {
    return useLocalStorage(`${options.storageKey}.${name}`, initial);
  }

  /** toggle switches a view option on or off. */
  function toggle(option: Ref<boolean>): void {
    option.value = !option.value;
  }

  /** openHelp opens the help of the editor. */
  function openHelp(): void {
    helpOpen.value = true;
  }

  /** toggleViewMode enters or leaves the view mode. */
  function toggleViewMode(): void {
    viewMode.value = !viewMode.value;
    if (viewMode.value) options.onEnterViewMode?.();
  }

  /** whenEditing wraps an editing action, so it does nothing in the view mode and during a gesture. */
  function whenEditing(action: () => void): () => void {
    return () => {
      if (!viewMode.value && !options.busy?.()) action();
    };
  }

  /** whenIdle wraps an action that also works in the view mode, so it does nothing during a gesture. */
  function whenIdle(action: () => void): () => void {
    return () => {
      if (!options.busy?.()) action();
    };
  }

  // Escape is not registered through defineShortcuts, which prevents the default action of every shortcut it handles,
  // and an open dialog only closes on an Escape whose default was not prevented.
  useEventListener(document, 'keydown', (event: KeyboardEvent) => {
    if (event.key === 'Escape') options.onEscape?.();
  });

  // Tab moves through the items only while no control has the focus, so the keyboard still moves
  // between the controls of the page as usual.
  useEventListener(document, 'keydown', (event: KeyboardEvent) => {
    if (!options.onTab || event.key !== 'Tab' || viewMode.value) return;
    if (event.ctrlKey || event.metaKey || event.altKey) return;
    if (document.activeElement && document.activeElement !== document.body) return;
    event.preventDefault();
    options.onTab(event.shiftKey ? -1 : 1);
  });

  return {
    sidebarOpen,
    viewMode,
    helpOpen,
    view,
    fullscreen,
    storedOption,
    toggle,
    openHelp,
    toggleViewMode,
    whenEditing,
    whenIdle,
  };
}
