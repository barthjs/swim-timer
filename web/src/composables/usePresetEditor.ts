import { ref } from 'vue';

import { useBackgroundImage } from '@/composables/editor/useBackgroundImage';
import { useEditorGuides } from '@/composables/editor/useEditorGuides';
import { useEditorSelection } from '@/composables/editor/useEditorSelection';
import { useEditorVisibility } from '@/composables/editor/useEditorVisibility';
import { useHistory } from '@/composables/useHistory';
import { backgroundImageStore } from '@/lib/backgroundImages';
import { frameOf, type EditorPreset } from '@/lib/lanes';

/**
 * usePresetEditor holds the editing state of the preset canvas, such as the selection, the history,
 * the hidden lanes, the background image, and the guides. It stays apart from the preset itself, which holds only
 * the data that is saved. The selection holds the keys of the lanes in the order of their numbers.
 */
export function usePresetEditor(
  preset: EditorPreset,
  laneKeys: () => readonly string[],
  savedName: () => string | null,
) {
  const selection = useEditorSelection(laneKeys);

  // The lanes hidden for a while, such as to reach one that another covers
  const visibility = useEditorVisibility<string>();

  // Whether lanes are dragged, which holds back the automatic history checkpoint until the drag ends
  const dragging = ref(false);

  const history = useHistory(preset, {
    paused: () => dragging.value,
  });

  // The image behind the lanes, such as a snapshot of the camera, to trace the lanes on
  const backgroundImage = useBackgroundImage({
    store: backgroundImageStore,
    keyPrefix: 'preset',
    storageKey: 'editor',
    content: () => frameOf(preset),
  });

  // The lines dragged out of the rulers, which the browser keeps per saved preset
  const guides = useEditorGuides({
    storageKey: 'editor',
    document: savedName,
  });

  return {
    selection,
    visibility,
    dragging,
    history,
    backgroundImage,
    guides,
  };
}
