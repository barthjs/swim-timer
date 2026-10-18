<script setup lang="ts" generic="K extends string | number">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { isAdditiveSelect } from '@/composables/editor/useEditorSelection';

/** EditorListItem is an item of an editor as its list shows it. */
export type EditorListItem<K> = {
  key: K;
  label: string;
  locked: boolean;
  hidden: boolean;
};

// The items of an editor in a list like the layers of an image editor, with their lock and their visibility.
// A click selects an item, with Shift, Ctrl, or Cmd in addition to the selection. A sortable list moves an item
// that is dragged onto another one to its place.
const props = defineProps<{
  items: EditorListItem<K>[];
  selected: ReadonlySet<K>;
  sortable?: boolean;
}>();

const emit = defineEmits<{
  select: [key: K, additive: boolean];
  toggleLock: [key: K];
  toggleHidden: [key: K];
  move: [from: number, to: number];
}>();

const { t } = useI18n();

// The position of the item that is dragged, and of the item it is dragged over
const draggedIndex = ref<number | null>(null);
const overIndex = ref<number | null>(null);

/** onDragStart remembers the dragged item. */
function onDragStart(event: DragEvent, index: number): void {
  draggedIndex.value = index;
  event.dataTransfer?.setData('text/plain', String(index));
}

/** onDragOver marks the item a dragged one would take the place of. */
function onDragOver(event: DragEvent, index: number): void {
  if (draggedIndex.value === null) return;
  event.preventDefault();
  overIndex.value = index;
}

/** onDrop moves the dragged item to the place of the item it was dropped on. */
function onDrop(index: number): void {
  const from = draggedIndex.value;
  onDragEnd();
  if (from !== null && from !== index) emit('move', from, index);
}

/** onDragEnd forgets the dragged item. */
function onDragEnd(): void {
  draggedIndex.value = null;
  overIndex.value = null;
}
</script>

<template>
  <ul
    class="border-default divide-default flex flex-col divide-y overflow-hidden rounded-md border"
  >
    <li
      v-for="(item, index) in items"
      :key="item.key"
      class="flex items-center gap-1 py-0.5 pr-1 pl-2 text-sm"
      :class="{
        'bg-primary/10': props.selected.has(item.key),
        'ring-primary ring-2 ring-inset': overIndex === index && draggedIndex !== index,
        'text-muted': item.hidden,
      }"
      :draggable="sortable"
      @dragstart="onDragStart($event, index)"
      @dragover="onDragOver($event, index)"
      @drop="onDrop(index)"
      @dragend="onDragEnd"
    >
      <UIcon
        v-if="sortable"
        name="i-lucide-grip-vertical"
        class="text-dimmed size-4 shrink-0 cursor-grab"
      />
      <button
        type="button"
        class="min-w-0 flex-1 truncate py-1 text-left"
        @click="emit('select', item.key, isAdditiveSelect($event))"
      >
        {{ item.label }}
      </button>
      <UTooltip :text="item.hidden ? t('editor.items.show') : t('editor.items.hide')">
        <UButton
          :icon="item.hidden ? 'i-lucide-eye-off' : 'i-lucide-eye'"
          size="xs"
          color="neutral"
          variant="ghost"
          :aria-label="item.hidden ? t('editor.items.show') : t('editor.items.hide')"
          :aria-pressed="item.hidden"
          @click="emit('toggleHidden', item.key)"
        />
      </UTooltip>
      <UTooltip :text="item.locked ? t('editor.items.unlock') : t('editor.items.lock')">
        <UButton
          :icon="item.locked ? 'i-lucide-lock' : 'i-lucide-lock-open'"
          size="xs"
          :color="item.locked ? 'primary' : 'neutral'"
          variant="ghost"
          :aria-label="item.locked ? t('editor.items.unlock') : t('editor.items.lock')"
          :aria-pressed="item.locked"
          @click="emit('toggleLock', item.key)"
        />
      </UTooltip>
    </li>
  </ul>
</template>
