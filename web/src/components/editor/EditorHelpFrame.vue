<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useShortcutLabels, type ShortcutKey } from '@/composables/useShortcutLabels';
import { EDITOR_SHORTCUTS, shortcutKeys, type EditorShortcut } from '@/lib/editor/shortcuts';

/** HelpShortcut is a row of the shortcut tab: the keys or mouse actions and what they do. */
export type HelpShortcut = {
  keys: ShortcutKey[];
  label: string;
};

// The help of an editor with the guide of the editor in one tab and its keyboard shortcuts
// and mouse actions in another. The tab lists the shortcuts every editor offers, then those of the editor,
// then the mouse actions of the canvas. Each shortcut lists its keys, written for the platform and the language.
const props = defineProps<{
  // The shortcuts and mouse actions of the editor itself.
  shortcuts: HelpShortcut[];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();
const { keyLabel } = useShortcutLabels();

// The mouse actions and held keys of the canvas, which every editor shares.
const CANVAS_ACTIONS: Array<{ keys: ShortcutKey[]; label: string }> = [
  {
    keys: ['meta'],
    label: 'noSnapping',
  },
  {
    keys: ['shift'],
    label: 'constrain',
  },
  {
    keys: ['escape'],
    label: 'clearSelection',
  },
  {
    keys: ['space'],
    label: 'pan',
  },
  {
    keys: ['wheel'],
    label: 'panWheel',
  },
  {
    keys: ['shift', 'wheel'],
    label: 'panSideways',
  },
  {
    keys: ['meta', 'wheel'],
    label: 'zoom',
  },
];

const rows = computed<HelpShortcut[]>(() => [
  ...(Object.keys(EDITOR_SHORTCUTS) as EditorShortcut[]).map((shortcut) => ({
    keys: shortcutKeys(EDITOR_SHORTCUTS[shortcut]),
    label: t(`editor.shortcuts.${shortcut}`),
  })),
  ...props.shortcuts,
  ...CANVAS_ACTIONS.map((action) => ({
    keys: action.keys,
    label: t(`editor.shortcuts.${action.label}`),
  })),
]);

const tabs = computed(() => [
  {
    label: t('editor.guideTab'),
    icon: 'i-lucide-book-open',
    slot: 'guide' as const,
  },
  {
    label: t('editor.shortcutsTab'),
    icon: 'i-lucide-keyboard',
    slot: 'shortcuts' as const,
  },
]);
</script>

<template>
  <UModal v-model:open="open" :title="t('editor.help')" :ui="{ content: 'max-w-3xl' }">
    <template #body>
      <UTabs :items="tabs" variant="link" class="w-full" :ui="{ content: 'pt-4' }">
        <template #guide>
          <div class="text-default flex flex-col gap-4 text-sm">
            <slot name="guide" />
          </div>
        </template>

        <template #shortcuts>
          <dl class="text-default grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-2 text-sm">
            <template v-for="shortcut in rows" :key="shortcut.label">
              <dt class="flex gap-1">
                <UKbd v-for="key in shortcut.keys" :key="key">{{ keyLabel(key) }}</UKbd>
              </dt>
              <dd>{{ shortcut.label }}</dd>
            </template>
          </dl>
        </template>
      </UTabs>
    </template>
  </UModal>
</template>
