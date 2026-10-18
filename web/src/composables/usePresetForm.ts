import {
  computed,
  inject,
  provide,
  reactive,
  ref,
  toRaw,
  toValue,
  type InjectionKey,
  type MaybeRefOrGetter,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { presetRepository } from '@/api/repositories/preset';
import type { PresetResponse } from '@/api/types';
import { useApiError } from '@/composables/useApiError';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useDraftRecovery } from '@/composables/editor/useDraftRecovery';
import { useDiscardGuard } from '@/composables/form/useDiscardConfirm';
import { useDirtyForm } from '@/composables/form/useDirtyForm';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { usePresetEditor } from '@/composables/usePresetEditor';
import { downloadJSON } from '@/lib/download';
import {
  alignOffsets,
  alignReference,
  distributeOffsets,
  type AlignEdge,
  type DistributeAxis,
} from '@/lib/editor/align';
import { mapPointBetween, unionBounds, type Bounds, type Point } from '@/lib/editor/geometry';
import {
  CORNERS,
  alignLanes,
  clampCoordinate,
  createPreset,
  frameOf,
  isDimension,
  isLaneGap,
  laneBounds,
  parsePreset,
  scaleLanes,
  toEditorPreset,
  toPresetRequest,
  translateLane,
  type EditorPreset,
  type PixelLane,
} from '@/lib/lanes';

export type PresetForm = ReturnType<typeof createPresetForm>;

const presetFormKey: InjectionKey<PresetForm> = Symbol('presetForm');

/** providePresetForm creates the preset form for a page and makes it available to its components. */
export function providePresetForm(
  form: MaybeRefOrGetter<FormErrorTarget | null | undefined>,
): PresetForm {
  const presetForm = createPresetForm(form);
  provide(presetFormKey, presetForm);
  return presetForm;
}

/** usePresetForm returns the preset form provided by the editor page. */
export function usePresetForm(): PresetForm {
  const presetForm = inject(presetFormKey);
  if (!presetForm) {
    throw new Error('usePresetForm must be used below providePresetForm');
  }
  return presetForm;
}

/**
 * createPresetForm holds the edited preset, the editing state of the canvas, and the operations on lanes and stored
 * presets. The preset is edited in pixels of its reference resolution and converted from and into the stored
 * fractions only when it enters or leaves the editor.
 */
function createPresetForm(form: MaybeRefOrGetter<FormErrorTarget | null | undefined>) {
  const { t } = useI18n();
  const { messageFor } = useApiError();
  const { confirm } = useConfirm();
  const toast = useToast();

  const presetNames = ref<string[]>([]);
  // The name the edited preset is saved under, or null for a new, unsaved draft
  const savedName = ref<string | null>(null);
  const loaded = ref(false);
  // Counts every opened preset, also consecutive drafts without a saved name,
  // so views can reset their editing aids whenever another preset opens.
  const openCount = ref(0);
  // Numbers the opened presets and the requests for stored ones, so a stored preset that arrives
  // after another one was selected or opened does not replace it.
  let latestOpen = 0;
  const preset = reactive<EditorPreset>(toEditorPreset(createPreset('', 0)));

  const editor = usePresetEditor(
    preset,
    () => Object.keys(preset.lanes).sort((a, b) => Number(a) - Number(b)),
    () => savedName.value,
  );
  const { selection, visibility, history, backgroundImage, guides } = editor;
  const dirtyForm = useDirtyForm(preset);

  const isDirty = computed(() => loaded.value && dirtyForm.isDirty.value);
  const isDraft = computed(() => loaded.value && savedName.value === null);
  // An unsaved draft is dropped on leaving it, so it counts as a change to discard
  const { confirmDiscard } = useDiscardGuard(computed(() => isDirty.value || isDraft.value));

  // The unsaved changes of a saved preset survive a reload in the browser, opening it again offers them
  const recovery = useDraftRecovery({
    storageKey: 'editor',
    document: () => savedName.value,
    state: preset,
    dirty: isDirty,
    confirm: () =>
      confirm({
        title: t('editor.draft.title'),
        description: t('editor.draft.description'),
        confirmLabel: t('editor.draft.restore'),
        icon: 'i-lucide-history',
      }),
  });

  const activeLane = computed(() =>
    selection.active.value ? (preset.lanes[selection.active.value] ?? null) : null,
  );

  // The selected lanes that can move, since a locked lane keeps its place
  const movableLanes = computed(() =>
    [...selection.selected.value]
      .map((key) => preset.lanes[key])
      .filter((lane): lane is PixelLane => lane !== undefined && !lane.locked),
  );

  // The bounds of the selected lanes, which zooming to the selection fills the canvas with
  const selectionBounds = computed<Bounds | null>(() =>
    unionBounds(
      [...selection.selected.value]
        .map((key) => preset.lanes[key])
        .filter((lane) => lane !== undefined)
        .map(laneBounds),
    ),
  );

  /**
   * clearForm removes the field errors of a failed save, which no longer apply to another state.
   */
  function clearForm(): void {
    toValue(form)?.clear();
  }

  /**
   * applyPreset opens a preset as the saved state: the undo history and the dirty check start from it.
   */
  function applyPreset(next: PresetResponse, name: string | null): void {
    latestOpen++;
    Object.assign(preset, toEditorPreset(structuredClone(next)));
    savedName.value = name;
    loaded.value = true;
    openCount.value++;
    dirtyForm.capture();
    selection.clear();
    visibility.showAll();
    clearForm();
    history.reset();
  }

  /** closePreset closes the open preset without saving it. */
  function closePreset(): void {
    latestOpen++;
    savedName.value = null;
    loaded.value = false;
    selection.clear();
  }

  /** loadPresetNames reads the names of the stored presets. */
  async function loadPresetNames(): Promise<void> {
    try {
      presetNames.value = await presetRepository.list();
    } catch (err) {
      toast.add({
        title: messageFor(err),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
    }
  }

  /**
   * selectPreset opens a stored preset after asking to discard unsaved changes.
   * Only the latest selection is opened, the answer for an earlier one is dropped.
   */
  async function selectPreset(name: string): Promise<void> {
    if (name === savedName.value) return;
    if (!(await confirmDiscard())) return;

    const request = ++latestOpen;
    try {
      const stored = await presetRepository.get(name);
      if (request !== latestOpen) return;
      applyPreset(stored, name);
    } catch (err) {
      if (request !== latestOpen) return;
      toast.add({
        title: t('pages.editor.loadError', {
          name,
        }),
        description: messageFor(err),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }
    await backgroundImage.show(name);
    await recovery.offer();
  }

  /** addPreset starts a new draft with rectangular lanes. It is stored on the first save. */
  async function addPreset(name: string, laneCount: number): Promise<boolean> {
    if (presetNames.value.includes(name)) {
      toast.add({
        title: t('pages.editor.presetExists'),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return false;
    }

    if (!(await confirmDiscard())) return false;

    applyPreset(createPreset(name, laneCount), null);
    return true;
  }

  /**
   * importPreset opens a preset file as a draft, named after the file if it holds no name.
   * A stored preset with the same name is replaced on save, so the import asks first.
   */
  async function importPreset(file: File | null | undefined): Promise<void> {
    if (!file) return;

    const imported = parsePreset(await file.text());
    if (!imported) {
      toast.add({
        title: t('pages.editor.importError', {
          name: file.name,
        }),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }

    const name = imported.name.trim() || file.name.replace(/\.json$/i, '');

    if (presetNames.value.includes(name)) {
      const replace = await confirm({
        title: t('pages.editor.importReplaceTitle'),
        description: t('pages.editor.importReplaceDescription', {
          name,
        }),
        confirmLabel: t('pages.editor.importReplace'),
        color: 'error',
        icon: 'i-lucide-file-up',
      });
      if (!replace) return;
    }

    if (!(await confirmDiscard())) return;

    applyPreset(
      {
        ...imported,
        name,
      },
      null,
    );
  }

  /**
   * downloadPreset saves the preset as shown in the editor, including unsaved changes,
   * as a file that importPreset reads back.
   */
  function downloadPreset(): void {
    if (!loaded.value || !preset.name) return;
    downloadJSON(`${preset.name}.json`, toPresetRequest(toRaw(preset)));
  }

  const { mutate: persistPreset, pending: saving } = useMutation(
    () => presetRepository.save(preset.name, toPresetRequest(preset)),
    {
      success: () => t('pages.editor.saveSuccess'),
      form,
      values: preset,
      onSuccess: async (saved) => {
        const previous = savedName.value;
        applyPreset(saved, saved.name);
        // A draft or a renamed preset takes the image along, and a renamed preset leaves nothing under its old name
        await backgroundImage.persist(saved.name, previous);
        guides.persist(saved.name, previous);
        if (previous !== null && previous !== saved.name) recovery.discard(previous);
        if (previous !== null && previous !== saved.name) await presetRepository.delete(previous);
        await loadPresetNames();
      },
    },
  );

  /**
   * savePreset stores the open preset after a confirmation. A renamed preset replaces the stored one,
   * unless another preset already has the new name.
   */
  async function savePreset(): Promise<void> {
    if (!loaded.value || !preset.name) return;

    const renamed = savedName.value !== null && savedName.value !== preset.name;
    if (renamed && presetNames.value.includes(preset.name)) {
      toast.add({
        title: t('pages.editor.presetExists'),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }

    const confirmed = await confirm({
      title: t('pages.editor.saveTitle'),
      description: renamed
        ? t('pages.editor.renameDescription', {
            from: savedName.value,
            name: preset.name,
          })
        : t('pages.editor.saveDescription', {
            name: preset.name,
          }),
      confirmLabel: t('common.save'),
      icon: 'i-lucide-save',
    });
    if (confirmed) void persistPreset();
  }

  /** resetPreset drops all changes since the preset was opened or saved. */
  async function resetPreset(): Promise<void> {
    const confirmed = await confirm({
      title: t('common.discardTitle'),
      description: t('common.discardDescription'),
      confirmLabel: t('common.discardConfirm'),
      color: 'error',
      icon: 'i-lucide-rotate-ccw',
    });
    if (!confirmed) return;

    dirtyForm.reset();
    selection.clear();
    clearForm();
    history.reset();
  }

  const { mutate: removePreset, pending: deleting } = useMutation(
    (name: string) => presetRepository.delete(name),
    {
      success: () => t('pages.editor.deleteSuccess'),
      onSuccess: async (_, name) => {
        closePreset();
        await backgroundImage.remove(name);
        guides.remove(name);
        recovery.discard(name);
        await loadPresetNames();
      },
    },
  );

  /** deletePreset deletes the open preset after a confirmation. */
  async function deletePreset(): Promise<void> {
    const name = savedName.value;
    if (!name) return;

    const confirmed = await confirm({
      title: t('pages.editor.deleteTitle'),
      description: t('pages.editor.deleteDescription', {
        name,
      }),
      confirmLabel: t('common.delete'),
      color: 'error',
      icon: 'i-lucide-trash-2',
    });
    if (confirmed) void removePreset(name);
  }

  /** align distributes the lanes between lane 1 and the last lane, see alignLanes. */
  function align(): void {
    if (!loaded.value) return;

    if (!alignLanes(preset.lanes, frameOf(preset), preset.laneGap)) {
      toast.add({
        title: t('pages.editor.alignError'),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }
    history.commit();
  }

  /**
   * setLaneGap changes the gap between the lanes and distributes the lanes again as align does, in one undo step.
   * While lane 1 and the last lane do not span a valid outline, only the value changes.
   * Values the backend rejects are ignored.
   */
  function setLaneGap(laneGap: number): void {
    if (!loaded.value || !isLaneGap(laneGap) || laneGap === preset.laneGap) return;

    preset.laneGap = laneGap;
    alignLanes(preset.lanes, frameOf(preset), laneGap);
    history.commit();
  }

  /**
   * toggleLock locks every selected lane if one of them is unlocked, and unlocks them otherwise,
   * so a mixed selection ends up locked.
   */
  function toggleLock(): void {
    const lanes = [...selection.selected.value]
      .map((key) => preset.lanes[key])
      .filter((lane) => lane !== undefined);
    if (lanes.length === 0) return;

    const lock = lanes.some((lane) => !lane.locked);
    for (const lane of lanes) {
      lane.locked = lock;
    }
    history.commit();
  }

  /**
   * nudge moves every selected, unlocked lane. A series of nudges becomes one history entry once it rests.
   */
  function nudge(dx: number, dy: number): void {
    if (!loaded.value) return;
    moveLanes(
      movableLanes.value,
      movableLanes.value.map(() => ({
        x: dx,
        y: dy,
      })),
    );
  }

  /**
   * alignSelection lines up the selected, unlocked lanes at an edge or a center line of their bounds,
   * or a single lane with the image.
   */
  function alignSelection(edge: AlignEdge): void {
    const lanes = movableLanes.value;
    if (!loaded.value || lanes.length === 0) return;

    const bounds = lanes.map(laneBounds);
    moveLanes(lanes, alignOffsets(bounds, edge, alignReference(bounds, frameOf(preset))));
    history.commit();
  }

  /** distributeSelection spaces the selected, unlocked lanes evenly along an axis. */
  function distributeSelection(axis: DistributeAxis): void {
    const lanes = movableLanes.value;
    if (!loaded.value) return;

    moveLanes(lanes, distributeOffsets(lanes.map(laneBounds), axis));
    history.commit();
  }

  /** setLocked locks or unlocks a single lane, such as from the list of the lanes. */
  function setLocked(key: string, locked: boolean): void {
    const lane = preset.lanes[key];
    if (!lane || lane.locked === locked) return;
    lane.locked = locked;
    history.commit();
  }

  /**
   * transformSelection moves and scales the selected, unlocked lanes together into new bounds, so they keep
   * their places relative to each other. Every corner stays inside the image.
   */
  function transformSelection(target: Bounds): void {
    const lanes = movableLanes.value;
    const from = unionBounds(lanes.map(laneBounds));
    if (!loaded.value || !from) return;

    const frame = frameOf(preset);
    for (const lane of lanes) {
      for (const corner of CORNERS) {
        const moved = mapPointBetween(lane[corner], from, target);
        lane[corner].x = clampCoordinate(moved.x, frame.width);
        lane[corner].y = clampCoordinate(moved.y, frame.height);
      }
    }
    history.commit();
  }

  /** moveLanes moves each lane by its own shift. Every corner stays inside the image, see translateLane. */
  function moveLanes(lanes: PixelLane[], offsets: Point[]): void {
    const frame = frameOf(preset);
    lanes.forEach((lane, index) => {
      const offset = offsets[index]!;
      translateLane(lane, structuredClone(toRaw(lane)), offset.x, offset.y, frame);
    });
  }

  /**
   * setFrameSize changes the reference resolution. The lanes keep their place in the image,
   * so only the aspect ratio of the canvas and the pixel values change. Sizes the backend rejects are ignored.
   */
  function setFrameSize(width: number, height: number): void {
    if (!loaded.value || !isDimension(width) || !isDimension(height)) return;
    if (width === preset.frameWidth && height === preset.frameHeight) return;

    scaleLanes(preset.lanes, frameOf(preset), {
      width,
      height,
    });
    preset.frameWidth = width;
    preset.frameHeight = height;
    history.commit();
  }

  /** setBackgroundImage shows an image behind the lanes, or removes it with null, see useBackgroundImage. */
  async function setBackgroundImage(image: Blob | null): Promise<void> {
    await backgroundImage.set(image, savedName.value);
  }

  return {
    preset,
    editor,
    presetNames,
    savedName,
    loaded,
    openCount,
    dirtyForm,
    isDirty,
    isDraft,
    activeLane,
    movableLanes,
    selectionBounds,
    saving,
    deleting,
    confirmDiscard,
    loadPresetNames,
    selectPreset,
    addPreset,
    importPreset,
    downloadPreset,
    savePreset,
    resetPreset,
    deletePreset,
    align,
    setLaneGap,
    toggleLock,
    nudge,
    setLocked,
    alignSelection,
    distributeSelection,
    transformSelection,
    setFrameSize,
    setBackgroundImage,
  };
}
