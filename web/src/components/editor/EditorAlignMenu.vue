<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import {
  ALIGN_EDGES,
  DISTRIBUTE_MIN,
  type AlignEdge,
  type DistributeAxis,
} from '@/lib/editor/align';

// A menu that lines up the selected items at an edge or a center line, or spaces them evenly.
// Several items align with each other, a single item with the content.
const props = defineProps<{
  // The number of selected items that can move.
  count: number;
}>();

const emit = defineEmits<{
  align: [edge: AlignEdge];
  distribute: [axis: DistributeAxis];
}>();

const { t } = useI18n();

const ALIGN_ICONS: Record<AlignEdge, string> = {
  left: 'i-lucide-align-start-vertical',
  centerX: 'i-lucide-align-center-vertical',
  right: 'i-lucide-align-end-vertical',
  top: 'i-lucide-align-start-horizontal',
  centerY: 'i-lucide-align-center-horizontal',
  bottom: 'i-lucide-align-end-horizontal',
};

const items = computed<DropdownMenuItem[][]>(() => [
  ALIGN_EDGES.slice(0, 3).map(alignItem),
  ALIGN_EDGES.slice(3).map(alignItem),
  [
    {
      label: t('editor.distribute.x'),
      icon: 'i-lucide-align-horizontal-distribute-center',
      disabled: props.count < DISTRIBUTE_MIN,
      onSelect: () => emit('distribute', 'x'),
    },
    {
      label: t('editor.distribute.y'),
      icon: 'i-lucide-align-vertical-distribute-center',
      disabled: props.count < DISTRIBUTE_MIN,
      onSelect: () => emit('distribute', 'y'),
    },
  ],
]);

/** alignItem builds the menu entry that aligns at an edge. */
function alignItem(edge: AlignEdge): DropdownMenuItem {
  return {
    label: t(`editor.align.${edge}`),
    icon: ALIGN_ICONS[edge],
    disabled: props.count === 0,
    onSelect: () => emit('align', edge),
  };
}
</script>

<template>
  <UDropdownMenu :items="items">
    <UTooltip :text="t('editor.align.title')">
      <UButton
        icon="i-lucide-align-center-vertical"
        size="sm"
        color="neutral"
        variant="ghost"
        :aria-label="t('editor.align.title')"
        :disabled="count === 0"
      />
    </UTooltip>
  </UDropdownMenu>
</template>
