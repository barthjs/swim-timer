<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import EditorAlignMenu from '@/components/editor/EditorAlignMenu.vue';
import EditorImageControl from '@/components/editor/EditorImageControl.vue';
import type { BackgroundImage } from '@/composables/editor/useBackgroundImage';
import type { EditorGuides } from '@/composables/editor/useEditorGuides';
import type { EditorViewOptions, EditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import type { History } from '@/composables/useHistory';
import { useShortcutLabels } from '@/composables/useShortcutLabels';
import type { AlignEdge, DistributeAxis } from '@/lib/editor/align';
import { EDITOR_SHORTCUTS, shortcutKeys, type EditorShortcut } from '@/lib/editor/shortcuts';

// The toolbar above the canvas of an editor. The first group holds undo, redo, the zoom, and the zoom to the selection,
// the second one the view options, the tools of the editor in the default slot, the alignment, the image,
// and on the right the view mode, the fullscreen mode, the sidebar, and the help. A group that does not fit beside
// the first one takes its own row, and its buttons wrap where even that row is too narrow. Every tooltip names its shortcut. The view mode hides the view options, so their buttons
// are disabled meanwhile.
const props = defineProps<{
  workspace: EditorWorkspace;
  history: History;
  // The zoom relative to the actual size of the content.
  zoom: number;
  // Whether items are selected that the zoom to the selection can show.
  hasSelection: boolean;
  // The number of selected items that can move, which the alignment works on.
  alignCount: number;
  image: BackgroundImage;
  // What the editor calls its image, such as a background image.
  imageLabel: string;
  guides: EditorGuides;
}>();

const emit = defineEmits<{
  zoomIn: [];
  zoomOut: [];
  resetZoom: [];
  zoomToSelection: [];
  // Zooms to a share of the actual size, such as 1 for 100 percent.
  zoomTo: [factor: number];
  align: [edge: AlignEdge];
  distribute: [axis: DistributeAxis];
  pickImage: [];
  clearImage: [];
}>();

const { t, locale } = useI18n();
const { keyLabels } = useShortcutLabels();

// The zoom levels the menu at the zoom offers, as shares of the actual size.
const ZOOM_LEVELS = [0.25, 0.5, 1, 2, 4];

/** percent formats a share as a percentage in the language of the page. */
function percent(value: number): string {
  return new Intl.NumberFormat(locale.value, {
    style: 'percent',
    maximumFractionDigits: 0,
  }).format(value);
}

const zoomItems = computed<DropdownMenuItem[][]>(() => [
  ZOOM_LEVELS.map((level) => ({
    label: percent(level),
    onSelect: () => emit('zoomTo', level),
  })),
  [
    {
      label: t('editor.resetZoom'),
      icon: 'i-lucide-maximize',
      kbds: kbds('resetZoom'),
      onSelect: () => emit('resetZoom'),
    },
    {
      label: t('editor.zoomToSelection'),
      icon: 'i-lucide-scan-search',
      kbds: kbds('zoomToSelection'),
      disabled: !props.hasSelection,
      onSelect: () => emit('zoomToSelection'),
    },
  ],
]);

const viewMode = computed(() => props.workspace.viewMode.value);
const sidebar = computed(() => props.workspace.sidebarOpen.value);
const fullscreen = computed(() => props.workspace.fullscreen.isFullscreen.value);

/** kbds returns the key labels of an editor shortcut for a tooltip. */
function kbds(shortcut: EditorShortcut): string[] {
  return keyLabels(shortcutKeys(EDITOR_SHORTCUTS[shortcut]));
}

// The view options in the order of the toolbar, with their icons.
const TOGGLES: Array<{ key: keyof EditorViewOptions; icon: string }> = [
  {
    key: 'grid',
    icon: 'i-lucide-grid-3x3',
  },
  {
    key: 'guides',
    icon: 'i-lucide-crosshair',
  },
  {
    key: 'rulers',
    icon: 'i-lucide-ruler',
  },
  {
    key: 'loupe',
    icon: 'i-lucide-search',
  },
  {
    key: 'preview',
    icon: 'i-lucide-eye',
  },
  {
    key: 'snapping',
    icon: 'i-lucide-magnet',
  },
];

const toggles = computed(() =>
  TOGGLES.map((toggle) => ({
    key: toggle.key,
    icon: toggle.icon,
    label: t(`editor.view.${toggle.key}`),
    active: props.workspace.view[toggle.key].value,
    kbds: kbds(toggle.key),
  })),
);

/** onToggle switches a view option. */
function onToggle(key: keyof EditorViewOptions): void {
  props.workspace.toggle(props.workspace.view[key]);
}
</script>

<template>
  <div class="bg-elevated border-default flex flex-wrap items-center gap-1 rounded-md border p-1">
    <div class="flex flex-wrap items-center gap-1">
      <UTooltip :text="t('editor.undo')" :kbds="kbds('undo')">
        <UButton
          icon="i-lucide-undo-2"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.undo')"
          :disabled="!history.canUndo.value"
          @click="history.undo()"
        />
      </UTooltip>
      <UTooltip :text="t('editor.redo')" :kbds="kbds('redo')">
        <UButton
          icon="i-lucide-redo-2"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.redo')"
          :disabled="!history.canRedo.value"
          @click="history.redo()"
        />
      </UTooltip>

      <div class="bg-accented mx-1 h-5 w-px" />

      <UTooltip :text="t('editor.zoomOut')" :kbds="kbds('zoomOut')">
        <UButton
          icon="i-lucide-zoom-out"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.zoomOut')"
          @click="emit('zoomOut')"
        />
      </UTooltip>
      <UDropdownMenu :items="zoomItems">
        <UButton
          :label="percent(zoom)"
          size="sm"
          color="neutral"
          variant="ghost"
          class="w-16 justify-center tabular-nums"
          :aria-label="t('editor.zoomLevel')"
        />
      </UDropdownMenu>
      <UTooltip :text="t('editor.zoomIn')" :kbds="kbds('zoomIn')">
        <UButton
          icon="i-lucide-zoom-in"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.zoomIn')"
          @click="emit('zoomIn')"
        />
      </UTooltip>
      <UTooltip :text="t('editor.resetZoom')" :kbds="kbds('resetZoom')">
        <UButton
          icon="i-lucide-maximize"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.resetZoom')"
          @click="emit('resetZoom')"
        />
      </UTooltip>
      <UTooltip :text="t('editor.zoomToSelection')" :kbds="kbds('zoomToSelection')">
        <UButton
          icon="i-lucide-scan-search"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.zoomToSelection')"
          :disabled="!hasSelection"
          @click="emit('zoomToSelection')"
        />
      </UTooltip>
    </div>

    <div class="bg-accented mx-1 hidden h-5 w-px sm:block" />

    <div class="flex grow flex-wrap items-center gap-1">
      <UTooltip
        v-for="toggle in toggles"
        :key="toggle.key"
        :text="toggle.label"
        :kbds="toggle.kbds"
      >
        <UButton
          :icon="toggle.icon"
          size="sm"
          :color="toggle.active ? 'primary' : 'neutral'"
          :variant="toggle.active ? 'soft' : 'ghost'"
          :aria-label="toggle.label"
          :aria-pressed="toggle.active"
          :disabled="viewMode"
          @click="onToggle(toggle.key)"
        />
      </UTooltip>

      <UTooltip :text="t('editor.guideLines.clear')">
        <UButton
          icon="i-lucide-eraser"
          size="sm"
          color="neutral"
          variant="ghost"
          :aria-label="t('editor.guideLines.clear')"
          :disabled="viewMode || guides.lines.value.length === 0"
          @click="guides.clear()"
        />
      </UTooltip>

      <div class="bg-accented mx-1 h-5 w-px" />
      <slot />
      <EditorAlignMenu
        :count="alignCount"
        @align="(edge) => emit('align', edge)"
        @distribute="(axis) => emit('distribute', axis)"
      />
      <EditorImageControl
        :label="imageLabel"
        :opacity="image.opacity.value"
        @update:opacity="image.setOpacity"
        :has-image="!!image.image.value"
        :kbds="kbds('image')"
        @pick="emit('pickImage')"
        @clear="emit('clearImage')"
      />

      <div class="ml-auto flex flex-wrap items-center gap-1">
        <UTooltip
          :text="viewMode ? t('editor.exitViewMode') : t('editor.viewMode')"
          :kbds="kbds('viewMode')"
        >
          <UButton
            icon="i-lucide-scan-eye"
            size="sm"
            :color="viewMode ? 'primary' : 'neutral'"
            :variant="viewMode ? 'soft' : 'ghost'"
            :aria-label="viewMode ? t('editor.exitViewMode') : t('editor.viewMode')"
            :aria-pressed="viewMode"
            @click="workspace.toggleViewMode()"
          />
        </UTooltip>
        <UTooltip
          v-if="workspace.fullscreen.isSupported.value"
          :text="fullscreen ? t('editor.exitFullscreen') : t('editor.fullscreen')"
          :kbds="kbds('fullscreen')"
        >
          <UButton
            :icon="fullscreen ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'"
            size="sm"
            :color="fullscreen ? 'primary' : 'neutral'"
            :variant="fullscreen ? 'soft' : 'ghost'"
            :aria-label="fullscreen ? t('editor.exitFullscreen') : t('editor.fullscreen')"
            :aria-pressed="fullscreen"
            @click="void workspace.fullscreen.toggle()"
          />
        </UTooltip>
        <UTooltip
          :text="sidebar ? t('editor.hideSidebar') : t('editor.showSidebar')"
          :kbds="kbds('sidebar')"
        >
          <UButton
            :icon="sidebar ? 'i-lucide-panel-right-close' : 'i-lucide-panel-right-open'"
            size="sm"
            color="neutral"
            variant="ghost"
            :aria-label="sidebar ? t('editor.hideSidebar') : t('editor.showSidebar')"
            :aria-pressed="sidebar"
            @click="workspace.toggle(workspace.sidebarOpen)"
          />
        </UTooltip>
        <UTooltip :text="t('editor.help')" :kbds="kbds('help')">
          <UButton
            icon="i-lucide-circle-help"
            size="sm"
            color="neutral"
            variant="ghost"
            :aria-label="t('editor.help')"
            @click="workspace.openHelp()"
          />
        </UTooltip>
      </div>
    </div>
  </div>
</template>
