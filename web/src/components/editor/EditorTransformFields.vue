<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { Bounds } from '@/lib/editor/geometry';

// The position and the size of several selected items together, in content units. A changed value moves
// or scales the selection as a whole, so its items keep their places relative to each other.
const props = defineProps<{
  bounds: Bounds;
  // The unit of the content, such as "px" or "mm".
  unit: string;
}>();

const emit = defineEmits<{
  change: [bounds: Bounds];
}>();

const { t } = useI18n();

const x = computed(() => Math.round(props.bounds.minX));
const y = computed(() => Math.round(props.bounds.minY));
const width = computed(() => Math.round(props.bounds.maxX - props.bounds.minX));
const height = computed(() => Math.round(props.bounds.maxY - props.bounds.minY));

/** setX moves the selection to a new left edge. */
function setX(value: number | null | undefined): void {
  if (typeof value !== 'number') return;
  emit('change', shifted(value - props.bounds.minX, 0));
}

/** setY moves the selection to a new top edge. */
function setY(value: number | null | undefined): void {
  if (typeof value !== 'number') return;
  emit('change', shifted(0, value - props.bounds.minY));
}

/** setWidth scales the selection to a new width from its left edge. */
function setWidth(value: number | null | undefined): void {
  if (typeof value !== 'number' || value <= 0) return;
  emit('change', {
    ...props.bounds,
    maxX: props.bounds.minX + value,
  });
}

/** setHeight scales the selection to a new height from its top edge. */
function setHeight(value: number | null | undefined): void {
  if (typeof value !== 'number' || value <= 0) return;
  emit('change', {
    ...props.bounds,
    maxY: props.bounds.minY + value,
  });
}

/** shifted returns the bounds of the selection moved by an offset. */
function shifted(dx: number, dy: number): Bounds {
  return {
    minX: props.bounds.minX + dx,
    maxX: props.bounds.maxX + dx,
    minY: props.bounds.minY + dy,
    maxY: props.bounds.maxY + dy,
  };
}
</script>

<template>
  <div class="grid grid-cols-2 gap-2">
    <UFormField :label="t('editor.transform.x', { unit })" class="min-w-0">
      <UInputNumber
        :model-value="x"
        orientation="vertical"
        class="w-full"
        @update:model-value="setX"
      />
    </UFormField>
    <UFormField :label="t('editor.transform.y', { unit })" class="min-w-0">
      <UInputNumber
        :model-value="y"
        orientation="vertical"
        class="w-full"
        @update:model-value="setY"
      />
    </UFormField>
    <UFormField :label="t('editor.transform.width', { unit })" class="min-w-0">
      <UInputNumber
        :model-value="width"
        orientation="vertical"
        :min="1"
        class="w-full"
        @update:model-value="setWidth"
      />
    </UFormField>
    <UFormField :label="t('editor.transform.height', { unit })" class="min-w-0">
      <UInputNumber
        :model-value="height"
        orientation="vertical"
        :min="1"
        class="w-full"
        @update:model-value="setHeight"
      />
    </UFormField>
  </div>
</template>
