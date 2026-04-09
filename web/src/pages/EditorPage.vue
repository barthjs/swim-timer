<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  AlignJustify,
  ChevronLeft,
  ChevronRight,
  Crosshair,
  Grid,
  Home,
  Image as ImageIcon,
  Lock,
  Plus,
  Redo2,
  RotateCcw,
  Save,
  Trash2,
  Undo2,
  Unlock,
} from 'lucide-vue-next';
import { useRouter } from 'vue-router';

import { presetsApi } from '@/api/presetsApi.ts';

import AppVersion from '@/components/AppVersion.vue';
import BaseButton from '@/components/BaseButton.vue';
import ConfirmModal from '@/components/ConfirmModal.vue';
import FormInput from '@/components/FormInput.vue';
import PoolCanvas from '@/components/PoolCanvas.vue';
import SelectDropdown from '@/components/SelectDropdown.vue';

import { useFormValidation } from '@/composables/useFormValidation.ts';
import { useHistory } from '@/composables/useHistory.ts';
import { useUnsavedChanges } from '@/composables/useUnsavedChanges.ts';

import { useToastStore } from '@/stores/toastStore.ts';

import type { LanePreset, Point, Preset } from '@/types';
import { CornerKey, TextAnchor } from '@/types';
import { isValidationError } from '@/types/api.ts';

const confirmDialog = ref<{ isOpen: boolean; message: string; action: (() => void) | null }>({
  isOpen: false,
  message: '',
  action: null,
});

const presetNames = ref<string[]>([]);
const currentPresetName = ref<string | null>(null);
const currentPreset = ref<Preset | null>(null);
const originalPresetString = ref<string>('');
const newPresetName = ref<string>('');
const lanesCount = ref<string>('8');

const isSidebarOpen = ref<boolean>(true);
const selectedLanes = ref<Set<number>>(new Set());

const showGrid = ref<boolean>(false);
const showCrosshair = ref<boolean>(true);
const mouseX = ref<number>(0);
const mouseY = ref<number>(0);

const bgImage = ref<string | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);

const dragState = ref<{
  mode: 'point' | 'lane';
  laneNum?: number;
  corner?: CornerKey;
  startX: number;
  startY: number;
  initialState: string;
} | null>(null);

const marquee = ref<{
  startX: number;
  startY: number;
  currentX: number;
  currentY: number;
} | null>(null);

const { t } = useI18n();
const router = useRouter();

const { errors, setErrors, clearErrors } = useFormValidation();
const { saveHistory, undo, redo, clearHistory } = useHistory(currentPreset);

const toastStore = useToastStore();

const showCoords = computed(() => showCrosshair.value && (!!dragState.value || !!marquee.value));

const hasChanges = computed((): boolean => {
  if (!currentPreset.value) return false;
  return JSON.stringify(currentPreset.value) !== originalPresetString.value;
});

useUnsavedChanges(hasChanges);

const cameraPositionOptions = computed(() => [t('editor.leftLTR'), t('editor.rightRTL')]);

// Updates text anchor position when camera orientation changes,
// affecting how text is displayed relative to the video
const handleCameraPositionUpdate = (label: string | number) => {
  if (currentPreset.value) {
    currentPreset.value.textAnchor =
      label === t('editor.leftLTR') ? TextAnchor.Left : TextAnchor.Right;
    saveHistory();
  }
};

const laneOrderOptions = computed(() => [t('editor.topLane1'), t('editor.topHighest')]);

// Toggles lane order direction (normal 1-8 or reversed highest-lowest) for display layout
const handleLaneOrderUpdate = (label: string | number) => {
  if (currentPreset.value) {
    currentPreset.value.reverseLanes = label === t('editor.topHighest');
    saveHistory();
  }
};

const activeLaneNum = computed((): number | null => {
  if (selectedLanes.value.size === 1) return Array.from(selectedLanes.value)[0]!;
  return null;
});

const activeLane = computed((): LanePreset | null => {
  if (currentPreset.value && activeLaneNum.value !== null) {
    return currentPreset.value.lanes[activeLaneNum.value] || null;
  }
  return null;
});

const triggerUpload = (): void => fileInput.value?.click();

// Loads a reference image for editing alignment, creating a temporary blob URL.
// Resets input value to allow re-selecting the same file
const handleImageUpload = (e: Event): void => {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0];
  if (file) {
    bgImage.value = URL.createObjectURL(file);
  }
  target.value = '';
};

const toggleLock = (): void => {
  if (!activeLane.value) return;
  activeLane.value.locked = !activeLane.value.locked;
  saveHistory();
};

const executeConfirm = (): void => {
  if (confirmDialog.value.action) confirmDialog.value.action();
  confirmDialog.value.isOpen = false;
};

// Fetches available preset names from the backend.
// Automatically selects the first preset on the initial load
const loadPresetNames = async (): Promise<void> => {
  try {
    presetNames.value = await presetsApi.getPresetNames();
    if (presetNames.value.length > 0 && !currentPresetName.value) {
      currentPresetName.value = presetNames.value[0] ?? null;
    }
  } catch (e) {
    toastStore.error(t('editor.failedLoadPresets'));
    console.error(e);
  }
};

// Loads a specific preset from the backend and resets
// the editor state (selection, errors, history).
const loadPresetData = async (name: string): Promise<void> => {
  try {
    const data = await presetsApi.getPreset(name);
    currentPreset.value = data;
    originalPresetString.value = JSON.stringify(data);
    selectedLanes.value.clear();
    clearErrors();
    clearHistory();
    saveHistory();
  } catch (e) {
    toastStore.error(t('editor.failedLoadData', { name }));
    console.error(e);
  }
};

watch(currentPresetName, async (newName) => {
  if (newName) {
    await loadPresetData(newName);
  } else {
    if (!currentPreset.value || currentPreset.value.name === '') {
      currentPreset.value = null;
      originalPresetString.value = '';
      selectedLanes.value.clear();
      clearHistory();
    }
  }
});

// Creates a new preset with the selected number of lanes positioned at default
// viewport coordinates and validates name uniqueness
const addPreset = (): void => {
  const name = newPresetName.value.trim();
  if (!name) return;
  if (presetNames.value.includes(name)) return toastStore.error(t('editor.presetExists'));

  const newPreset: Preset = {
    name: name,
    comment: null,
    virtualWidth: 2000,
    virtualHeight: 200,
    reverseLanes: false,
    textAnchor: TextAnchor.Left,
    lanes: {},
  };

  const laneHeight = 50;
  const laneGap = 10;
  const startX1 = window.innerWidth / 4;
  const startX2 = (window.innerWidth * 3) / 4;
  const maxLaneNum = parseInt(lanesCount.value, 10);

  for (let i = 1; i <= maxLaneNum; i++) {
    const startY = 100 + i * (laneHeight + laneGap);
    newPreset.lanes[i] = {
      [CornerKey.TopLeft]: { x: startX1, y: startY },
      [CornerKey.TopRight]: { x: startX2, y: startY },
      [CornerKey.BottomRight]: { x: startX2, y: startY + laneHeight },
      [CornerKey.BottomLeft]: { x: startX1, y: startY + laneHeight },
      locked: false,
    };
  }

  currentPresetName.value = null;
  currentPreset.value = newPreset;
  originalPresetString.value = JSON.stringify(newPreset);
  selectedLanes.value.clear();
  newPresetName.value = '';
  clearErrors();
  clearHistory();
  saveHistory();
};

// Saves the current preset to the backend after confirmation,
// then resets history and updates the preset list
const savePreset = (): void => {
  if (!currentPreset.value || !currentPreset.value.name) return;
  confirmDialog.value = {
    isOpen: true,
    message: t('editor.saveChanges'),
    action: async () => {
      const nameToSave = currentPreset.value!.name;
      try {
        await presetsApi.savePreset(nameToSave, currentPreset.value!);
        originalPresetString.value = JSON.stringify(currentPreset.value);
        currentPresetName.value = nameToSave;
        clearErrors();
        clearHistory();
        await loadPresetNames();
        toastStore.success(t('editor.presetSaved'));
      } catch (e) {
        if (isValidationError(e)) {
          setErrors(e.errors);
        }
        toastStore.error(t('editor.failedSavePreset'));
        console.error(e);
      }
    },
  };
};

// Reverts preset to last saved state after confirmation,
// discarding all unsaved edits
const resetPreset = (): void => {
  confirmDialog.value = {
    isOpen: true,
    message: t('editor.discardChanges'),
    action: () => {
      if (originalPresetString.value) currentPreset.value = JSON.parse(originalPresetString.value);
      clearErrors();
      clearHistory();
      saveHistory();
    },
  };
};

// Permanently deletes the current preset from the backend after confirmation,
// then reloads the preset list
const deletePreset = (): void => {
  if (!currentPresetName.value) return;
  confirmDialog.value = {
    isOpen: true,
    message: t('editor.deleteConfirm'),
    action: async () => {
      try {
        await presetsApi.deletePreset(currentPresetName.value!);
        currentPresetName.value = null;
        currentPreset.value = null;
        originalPresetString.value = '';
        selectedLanes.value.clear();
        await loadPresetNames();
        toastStore.success(t('editor.presetDeleted'));
      } catch (e) {
        toastStore.error(t('editor.failedDeletePreset'));
        console.error(e);
      }
    },
  };
};

// Applies projective transformation to distribute lanes evenly using homography math.
// This corrects perspective distortion so lanes remain proportional even when the camera is at an angle.
// Requires lanes 1 and the last lane to be manually positioned as reference points.
const alignLanes = (): void => {
  if (!currentPreset.value) return;
  const lanes = currentPreset.value.lanes;

  // Determine the last lane number dynamically from the preset
  const maxLane = Math.max(...Object.keys(lanes).map(Number));

  if (!lanes[1] || !lanes[maxLane]) return;

  // Extract the 4 corners of the entire pool area from manually positioned lanes 1 and the last lane
  const x0 = lanes[1][CornerKey.TopLeft].x,
    y0 = lanes[1][CornerKey.TopLeft].y; // P0: Top Left
  const x1 = lanes[1][CornerKey.TopRight].x,
    y1 = lanes[1][CornerKey.TopRight].y; // P1: Top Right
  const x2 = lanes[maxLane][CornerKey.BottomRight].x,
    y2 = lanes[maxLane][CornerKey.BottomRight].y; // P2: Bottom Right
  const x3 = lanes[maxLane][CornerKey.BottomLeft].x,
    y3 = lanes[maxLane][CornerKey.BottomLeft].y; // P3: Bottom Left

  // Compute homography coefficients (perspective transform parameters).
  // These handle cases where the pool appears trapezoid-shaped due to camera perspective.
  // g and h are perspective factors; when both are 0, the transformation is affine (no perspective).
  const sumX = x0 - x1 + x2 - x3;
  const sumY = y0 - y1 + y2 - y3;
  const dx1 = x1 - x2;
  const dx2 = x3 - x2;
  const dy1 = y1 - y2;
  const dy2 = y3 - y2;

  const D = dx1 * dy2 - dx2 * dy1;

  let g = 0,
    h = 0;
  // Guard against division by zero. If D is near zero, the trapezoid is degenerate.
  if (Math.abs(D) > 0.0001) {
    g = (sumX * dy2 - sumY * dx2) / D;
    h = (dx1 * sumY - dy1 * sumX) / D;
  }

  // Compute affine transformation coefficients (a, b, c, d, e, f).
  // These map normalized coordinates (u=0 to 1, v=0 to 1) to screen pixel coordinates.
  // The formula is: x = (a*u + b*v + c) / (g*u + h*v + 1), same for y with d, e, f.
  const a = x1 - x0 + g * x1;
  const b = x3 - x0 + h * x3;
  const c = x0;
  const d = y1 - y0 + g * y1;
  const e = y3 - y0 + h * y3;
  const f = y0;

  // Map theoretical relative coordinates (u, v) in range [0, 1] to actual pixel coordinates on screen.
  // u represents the horizontal position (0=left, 1=right); v represents the vertical lane position.
  const mapPoint = (u: number, v: number): Point => {
    const denominator = g * u + h * v + 1;
    return {
      x: Math.round((a * u + b * v + c) / denominator),
      y: Math.round((d * u + e * v + f) / denominator),
    };
  };

  // Calculate proportional lane dimensions based on the actual number of lanes.
  // Each gap is 2% of pool height, lanes share remaining space equally.
  const gapRatio = 0.02; // Ratio of space between lanes
  const totalGapRatio = gapRatio * (maxLane - 1); // Gaps between all lanes
  const laneRatio = (1 - totalGapRatio) / maxLane; // Each lane's proportional height

  for (let i = 1; i <= maxLane; i++) {
    const lane = lanes[i];
    if (!lane || lane.locked) continue;

    const idx = i - 1;

    // Calculate the vertical proportion (v) for top and bottom of the current lane.
    // This distributes lanes proportionally with gaps between them.
    const topV = idx * (laneRatio + gapRatio);
    const bottomV = topV + laneRatio;

    // Apply projective transformation. The left side is u=0, the right side is u=1.
    // This maps each lane's four corners from the theoretical grid to their perspective-corrected positions.
    lane[CornerKey.TopLeft] = mapPoint(0, topV);
    lane[CornerKey.TopRight] = mapPoint(1, topV);
    lane[CornerKey.BottomLeft] = mapPoint(0, bottomV);
    lane[CornerKey.BottomRight] = mapPoint(1, bottomV);
  }

  saveHistory();
};

// Moves all selected lanes by (dx, dy) pixels.
// Skips locked lanes and only saves history if any lane actually moved
const nudge = (dx: number, dy: number): void => {
  if (!currentPreset.value || selectedLanes.value.size === 0) return;

  let moved = false;
  selectedLanes.value.forEach((num) => {
    const lane = currentPreset.value!.lanes[num];
    if (!lane || lane.locked) return;
    lane[CornerKey.TopLeft].x += dx;
    lane[CornerKey.TopLeft].y += dy;
    lane[CornerKey.TopRight].x += dx;
    lane[CornerKey.TopRight].y += dy;
    lane[CornerKey.BottomLeft].x += dx;
    lane[CornerKey.BottomLeft].y += dy;
    lane[CornerKey.BottomRight].x += dx;
    lane[CornerKey.BottomRight].y += dy;
    moved = true;
  });
  if (moved) saveHistory();
};

// Global keyboard handler for editor shortcuts. Skips form inputs to allow normal typing.
const onKeydown = (e: KeyboardEvent): void => {
  // Prevent catching input typing
  if (
    (e.target as HTMLElement).tagName === 'INPUT' ||
    (e.target as HTMLElement).tagName === 'SELECT'
  )
    return;

  if (e.ctrlKey && e.key === 'z') {
    e.preventDefault();
    undo();
    return;
  }
  if (e.ctrlKey && e.key === 'y') {
    e.preventDefault();
    redo();
    return;
  }
  if (e.ctrlKey && e.key === 's') {
    e.preventDefault();
    savePreset();
    return;
  }

  if (e.key === 'Escape') {
    selectedLanes.value.clear();
    return;
  }

  if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
    e.preventDefault();
    const amount = e.shiftKey ? 10 : 1;
    if (e.key === 'ArrowUp') nudge(0, -amount);
    if (e.key === 'ArrowDown') nudge(0, amount);
    if (e.key === 'ArrowLeft') nudge(-amount, 0);
    if (e.key === 'ArrowRight') nudge(amount, 0);
  }
};

// Starts marquee selection when clicking on the canvas background.
// Clears existing selection unless Shift is held for multi-select
const onBackgroundMouseDown = (e: MouseEvent): void => {
  // Ignore clicks on interactive UI elements
  const target = e.target as HTMLElement;
  if (target.closest('.controls') || target.closest('.handle') || target.closest('polygon')) {
    return;
  }

  marquee.value = {
    startX: e.clientX,
    startY: e.clientY,
    currentX: e.clientX,
    currentY: e.clientY,
  };
  if (!e.shiftKey) selectedLanes.value.clear();
  window.addEventListener('mousemove', onMarqueeMove);
  window.addEventListener('mouseup', onMarqueeUp);
};

// Updates marquee selection as the user drags.
const onMarqueeMove = (e: MouseEvent): void => {
  if (!marquee.value || !currentPreset.value) return;

  marquee.value.currentX = e.clientX;
  marquee.value.currentY = e.clientY;

  const minX = Math.min(marquee.value.startX, marquee.value.currentX);
  const maxX = Math.max(marquee.value.startX, marquee.value.currentX);
  const minY = Math.min(marquee.value.startY, marquee.value.currentY);
  const maxY = Math.max(marquee.value.startY, marquee.value.currentY);

  for (const [laneNumStr, lane] of Object.entries(currentPreset.value.lanes)) {
    if (lane.locked) continue;
    const laneNum = Number(laneNumStr);

    const lMinX = Math.min(lane[CornerKey.TopLeft].x, lane[CornerKey.BottomLeft].x);
    const lMaxX = Math.max(lane[CornerKey.TopRight].x, lane[CornerKey.BottomRight].x);
    const lMinY = Math.min(lane[CornerKey.TopLeft].y, lane[CornerKey.TopRight].y);
    const lMaxY = Math.max(lane[CornerKey.BottomLeft].y, lane[CornerKey.BottomRight].y);

    if (lMinX < maxX && lMaxX > minX && lMinY < maxY && lMaxY > minY) {
      selectedLanes.value.add(laneNum);
    } else if (!e.shiftKey) {
      selectedLanes.value.delete(laneNum);
    }
  }
};

const onMarqueeUp = (): void => {
  marquee.value = null;
  window.removeEventListener('mousemove', onMarqueeMove);
  window.removeEventListener('mouseup', onMarqueeUp);
};

// Initiates point drag mode for a specific lane corner. Saves initial state for real-time updates
const onCornerMouseDown = (e: MouseEvent, laneNum: number, corner: CornerKey): void => {
  if (!currentPreset.value) return;
  e.preventDefault();
  e.stopPropagation();

  dragState.value = {
    mode: 'point',
    laneNum,
    corner,
    startX: e.clientX,
    startY: e.clientY,
    initialState: JSON.stringify(currentPreset.value.lanes),
  };

  window.addEventListener('mousemove', onMouseMove);
  window.addEventListener('mouseup', onMouseUp);
};

// Handles lane selection with Shift for multi-select or Ctrl/Cmd style toggle.
// Initiates lane drag mode if lanes are selected
const onLaneMouseDown = (e: MouseEvent, laneNum: number): void => {
  if (!currentPreset.value) return;
  e.preventDefault();

  if (e.shiftKey) {
    if (selectedLanes.value.has(laneNum)) {
      selectedLanes.value.delete(laneNum);
    } else {
      selectedLanes.value.add(laneNum);
    }
  } else {
    if (!selectedLanes.value.has(laneNum)) {
      selectedLanes.value.clear();
      selectedLanes.value.add(laneNum);
    }
  }

  if (selectedLanes.value.size > 0) {
    dragState.value = {
      mode: 'lane',
      startX: e.clientX,
      startY: e.clientY,
      initialState: JSON.stringify(currentPreset.value.lanes),
    };
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }
};

// Updates coordinate display and applies real-time drag transformations.
// Point mode moves a single corner; lane mode translates entire lanes
const onMouseMove = (e: MouseEvent): void => {
  mouseX.value = e.clientX;
  mouseY.value = e.clientY;

  if (dragState.value && currentPreset.value) {
    const { mode, laneNum, corner, startX, startY, initialState } = dragState.value;
    const initialLanes = JSON.parse(initialState) as Record<number, LanePreset>;

    if (mode === 'point' && laneNum && corner) {
      const lane = currentPreset.value.lanes[laneNum];
      if (lane && lane[corner]) {
        lane[corner].x = e.clientX;
        lane[corner].y = e.clientY;
      }
    } else if (mode === 'lane') {
      const dx = e.clientX - startX;
      const dy = e.clientY - startY;

      selectedLanes.value.forEach((num) => {
        const lane = currentPreset.value!.lanes[num];
        const initLane = initialLanes[num];

        if (lane && initLane && !lane.locked) {
          lane[CornerKey.TopLeft].x = initLane[CornerKey.TopLeft].x + dx;
          lane[CornerKey.TopLeft].y = initLane[CornerKey.TopLeft].y + dy;
          lane[CornerKey.TopRight].x = initLane[CornerKey.TopRight].x + dx;
          lane[CornerKey.TopRight].y = initLane[CornerKey.TopRight].y + dy;
          lane[CornerKey.BottomLeft].x = initLane[CornerKey.BottomLeft].x + dx;
          lane[CornerKey.BottomLeft].y = initLane[CornerKey.BottomLeft].y + dy;
          lane[CornerKey.BottomRight].x = initLane[CornerKey.BottomRight].x + dx;
          lane[CornerKey.BottomRight].y = initLane[CornerKey.BottomRight].y + dy;
        }
      });
    }
  }
};

// Finalizes drag operation and saves state to history. Cleans up event listeners
const onMouseUp = (): void => {
  if (dragState.value) saveHistory();
  dragState.value = null;
  window.removeEventListener('mousemove', onMouseMove);
  window.removeEventListener('mouseup', onMouseUp);
};

onMounted(() => {
  loadPresetNames();
  window.addEventListener('keydown', onKeydown);
  window.addEventListener('mousemove', (e) => {
    if (!dragState.value && !marquee.value) {
      mouseX.value = e.clientX;
      mouseY.value = e.clientY;
    }
  });
});

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown);
  window.removeEventListener('mousemove', onMouseMove);
  window.removeEventListener('mouseup', onMouseUp);
  window.removeEventListener('mousemove', onMarqueeMove);
  window.removeEventListener('mouseup', onMarqueeUp);
});
</script>

<template>
  <div
    class="text-fg min-h-screen bg-transparent select-none"
    :class="{ 'show-grid': showGrid }"
    @mousedown="onBackgroundMouseDown"
  >
    <input
      type="file"
      ref="fileInput"
      accept="image/*"
      @change="handleImageUpload"
      class="hidden"
    />
    <img
      v-if="bgImage"
      :src="bgImage"
      class="pointer-events-none absolute top-0 left-0 h-dvh w-dvw object-contain opacity-60"
      alt="Reference Image"
    />

    <template v-if="showCrosshair && (dragState || marquee)">
      <div
        class="bg-primary pointer-events-none absolute inset-x-0 z-50 h-px mix-blend-difference"
        :style="{ top: mouseY + 'px' }"
      ></div>
      <div
        class="bg-primary pointer-events-none absolute inset-y-0 z-50 w-px mix-blend-difference"
        :style="{ left: mouseX + 'px' }"
      ></div>
      <div
        class="bg-surface absolute z-70 rounded-md p-1.5 text-xs shadow-lg"
        v-if="showCoords"
        :style="{ left: mouseX + 12 + 'px', top: mouseY + 12 + 'px' }"
      >
        <div class="leading-none whitespace-nowrap">X: {{ mouseX }}</div>
        <div class="leading-none whitespace-nowrap">Y: {{ mouseY }}</div>
      </div>
    </template>

    <div
      v-if="marquee"
      class="marquee-box pointer-events-none absolute z-90"
      :style="{
        left: Math.min(marquee.startX, marquee.currentX) + 'px',
        top: Math.min(marquee.startY, marquee.currentY) + 'px',
        width: Math.abs(marquee.currentX - marquee.startX) + 'px',
        height: Math.abs(marquee.currentY - marquee.startY) + 'px',
      }"
    ></div>

    <PoolCanvas
      :currentPreset="currentPreset"
      :selectedLanes="selectedLanes"
      :activeLaneNum="activeLaneNum"
      :activeLane="activeLane"
      @lane-down="onLaneMouseDown"
      @corner-down="onCornerMouseDown"
    />

    <div
      class="absolute top-4 z-101 flex flex-col gap-1 transition-[left] duration-300 ease-in-out"
      :class="isSidebarOpen ? 'left-88' : 'left-0'"
    >
      <BaseButton
        size="icon"
        @click="isSidebarOpen = !isSidebarOpen"
        @mousedown.stop
        class="bg-surface/90 rounded-l-none rounded-r-md"
        :class="isSidebarOpen ? 'border-subtle border-l' : ''"
        :title="isSidebarOpen ? t('editor.closeSidebar') : t('editor.openSidebar')"
      >
        <ChevronLeft v-if="isSidebarOpen" :size="18" />
        <ChevronRight v-else :size="18" />
      </BaseButton>

      <BaseButton
        :icon="Home"
        size="icon"
        class="bg-surface/90 rounded-l-none rounded-r-md"
        :title="t('common.backToHome')"
        @click="router.push({ name: 'home' })"
      />
    </div>

    <ConfirmModal
      :isOpen="confirmDialog.isOpen"
      :message="confirmDialog.message"
      @cancel="confirmDialog.isOpen = false"
      @confirm="executeConfirm"
    />

    <div
      class="bg-surface/90 scrollbar-none absolute top-0 bottom-0 left-0 z-100 flex w-88 flex-col space-y-4 overflow-y-auto p-4 transition-transform duration-300 ease-in-out"
      :class="{ '-translate-x-full': !isSidebarOpen }"
      @mousedown.stop
    >
      <div class="flex flex-col space-y-2">
        <SelectDropdown
          v-model="currentPresetName"
          :options="presetNames"
          :placeholder="t('editor.choosePreset')"
          :emptyText="t('editor.noPresets')"
        />

        <div class="flex gap-2">
          <FormInput
            v-model="newPresetName"
            :placeholder="t('editor.newPreset')"
            @keyup.enter="addPreset"
            class="w-full"
          />
          <BaseButton
            :icon="Plus"
            size="icon"
            :title="t('editor.addPreset')"
            @click="addPreset"
            class="mt-auto h-10 w-10 shrink-0"
          />
        </div>

        <div v-if="newPresetName.trim()" class="flex flex-col space-y-1">
          <label class="text-fg/80 text-sm">{{ t('editor.lanesCount') }}</label>
          <SelectDropdown
            v-model="lanesCount"
            :options="['2', '4', '6', '8', '10']"
            :placeholder="t('editor.selectLanesCount')"
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2">
        <BaseButton
          type="toggle"
          :active="showGrid"
          :icon="Grid"
          :title="t('editor.grid')"
          @click="showGrid = !showGrid"
        >
          {{ t('editor.grid') }}
        </BaseButton>

        <BaseButton
          type="toggle"
          :active="showCrosshair"
          :icon="Crosshair"
          :title="t('editor.guides')"
          @click="showCrosshair = !showCrosshair"
        >
          {{ t('editor.guides') }}
        </BaseButton>

        <BaseButton
          v-if="!bgImage"
          type="toggle"
          :icon="ImageIcon"
          :title="t('editor.bgImage')"
          @click="triggerUpload"
          class="col-span-2"
        >
          {{ t('editor.bgImage') }}
        </BaseButton>

        <BaseButton
          v-if="bgImage"
          type="toggle"
          :icon="Trash2"
          :title="t('editor.clearImage')"
          @click="bgImage = null"
          class="col-span-2"
        >
          {{ t('editor.clearImage') }}
        </BaseButton>

        <BaseButton
          v-if="currentPreset"
          variant="primary"
          :icon="AlignJustify"
          @click="alignLanes"
          class="col-span-2"
        >
          {{ t('editor.alignLanes') }}
        </BaseButton>
      </div>

      <div v-if="currentPreset" class="flex flex-col space-y-2">
        <div class="flex items-center justify-between">
          <h4 class="text-lg">
            {{ currentPresetName || t('editor.draft', { name: currentPreset.name }) }}
          </h4>
          <span v-if="hasChanges" class="bg-warning rounded px-1 py-0.5 text-xs">
            {{ t('editor.changed') }}
          </span>
        </div>

        <div class="flex space-x-2">
          <BaseButton
            type="toggle"
            :icon="Undo2"
            :iconSize="16"
            :title="t('editor.undo', { key: 'Ctrl+Z' })"
            @click="undo"
            class="w-full"
          />

          <BaseButton
            type="toggle"
            :icon="Redo2"
            :iconSize="16"
            :title="t('editor.redo', { key: 'Ctrl+Y' })"
            @click="redo"
            class="w-full"
          />
        </div>

        <div class="flex flex-col space-y-2">
          <label class="text-sm font-medium">{{ t('editor.virtualDimensions') }}</label>
          <div class="flex gap-2">
            <FormInput
              type="number"
              v-model="currentPreset.virtualWidth"
              :error="errors.VirtualWidth"
              :min="1"
              :max="10000"
              @update:model-value="saveHistory"
              class="w-full"
            />
            <FormInput
              type="number"
              v-model="currentPreset.virtualHeight"
              :error="errors.VirtualHeight"
              :min="1"
              :max="10000"
              @update:model-value="saveHistory"
              class="w-full"
            />
          </div>
        </div>

        <div class="flex flex-col gap-2">
          <label class="text-sm font-medium">{{ t('editor.laneOrder') }}</label>
          <SelectDropdown
            :model-value="
              currentPreset.reverseLanes ? t('editor.topHighest') : t('editor.topLane1')
            "
            :options="laneOrderOptions"
            :error="errors.ReverseLanes"
            @update:model-value="handleLaneOrderUpdate"
          />
        </div>

        <div class="flex flex-col gap-2">
          <label class="text-sm font-medium">{{ t('editor.cameraPosition') }}</label>
          <SelectDropdown
            :model-value="
              currentPreset.textAnchor === TextAnchor.Left
                ? t('editor.leftLTR')
                : t('editor.rightRTL')
            "
            :options="cameraPositionOptions"
            :error="errors.TextAnchor"
            @update:model-value="handleCameraPositionUpdate"
          />
        </div>

        <FormInput
          :label="t('editor.comment')"
          :model-value="currentPreset.comment ?? ''"
          :error="errors.Comment"
          @update:model-value="
            currentPreset.comment = $event as string;
            saveHistory();
          "
        />
      </div>

      <div v-if="activeLaneNum && activeLane" class="flex flex-col space-y-2">
        <div class="flex items-center justify-between">
          <span>{{ t('editor.lane', { number: activeLaneNum }) }}</span>
          <BaseButton
            size="icon"
            :icon="activeLane.locked ? Lock : Unlock"
            :iconSize="16"
            :class="{ 'bg-subtle!': activeLane.locked }"
            @click="toggleLock"
          />
        </div>

        <div class="flex flex-col space-y-2" v-if="!activeLane.locked">
          <div class="flex flex-col gap-1">
            <span class="font-mono text-xs">{{ CornerKey.TopLeft.toUpperCase() }}</span>
            <div class="flex gap-2">
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.TopLeft].x"
                :error="errors[`Lanes[${activeLaneNum}].TopLeft.X`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.TopLeft].y"
                :error="errors[`Lanes[${activeLaneNum}].TopLeft.Y`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
            </div>
          </div>

          <div class="flex flex-col gap-1">
            <span class="font-mono text-xs">{{ CornerKey.TopRight.toUpperCase() }}</span>
            <div class="flex gap-2">
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.TopRight].x"
                :error="errors[`Lanes[${activeLaneNum}].TopRight.X`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.TopRight].y"
                :error="errors[`Lanes[${activeLaneNum}].TopRight.Y`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
            </div>
          </div>

          <div class="flex flex-col gap-1">
            <span class="font-mono text-xs">{{ CornerKey.BottomLeft.toUpperCase() }}</span>
            <div class="flex gap-2">
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.BottomLeft].x"
                :error="errors[`Lanes[${activeLaneNum}].BottomLeft.X`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.BottomLeft].y"
                :error="errors[`Lanes[${activeLaneNum}].BottomLeft.Y`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
            </div>
          </div>

          <div class="flex flex-col gap-1">
            <span class="font-mono text-xs">{{ CornerKey.BottomRight.toUpperCase() }}</span>
            <div class="flex gap-2">
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.BottomRight].x"
                :error="errors[`Lanes[${activeLaneNum}].BottomRight.X`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
              <FormInput
                type="number"
                v-model="activeLane[CornerKey.BottomRight].y"
                :error="errors[`Lanes[${activeLaneNum}].BottomRight.Y`]"
                :min="0"
                :max="10000"
                @update:model-value="saveHistory"
                class="w-full"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="mt-auto flex flex-col space-y-2">
        <BaseButton
          v-if="hasChanges || (!currentPresetName && currentPreset)"
          variant="success"
          :icon="Save"
          @click="savePreset"
        >
          {{ t('common.save') }}
        </BaseButton>

        <BaseButton v-if="hasChanges" variant="warning" :icon="RotateCcw" @click="resetPreset">
          {{ t('editor.reset') }}
        </BaseButton>

        <BaseButton v-if="currentPresetName" variant="danger" :icon="Trash2" @click="deletePreset">
          {{ t('common.delete') }}
        </BaseButton>
      </div>

      <footer class="border-subtle border-t pt-4">
        <AppVersion />
      </footer>
    </div>
  </div>
</template>

<style scoped>
.editor.show-grid {
  background-size: 50px 50px;
  background-image:
    linear-gradient(to right, rgba(128, 128, 128, 0.4) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(128, 128, 128, 0.4) 1px, transparent 1px);
}

.marquee-box {
  border: 1px dashed var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
}
</style>
