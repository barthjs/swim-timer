<script setup lang="ts">
import { useI18n } from 'vue-i18n';

withDefaults(
  defineProps<{
    color?: 'warning';
    highlight?: boolean;
    placeholder?: string;
  }>(),
  {
    color: undefined,
    highlight: false,
    placeholder: '#0ea5e9',
  },
);

const model = defineModel<string>({
  required: true,
});

// A swatch that opens the color picker in a popover, next to a hex text input bound to the same model,
// which fills the width of the field.
// The saved value is always the visible one, and a color can be typed or pasted as well.
const { t } = useI18n();

/** Takes the color of the picker, which reports undefined when it has none. */
function updateColor(value: string | undefined): void {
  if (value !== undefined) model.value = value;
}
</script>

<template>
  <div class="flex items-center gap-2">
    <UPopover>
      <button
        type="button"
        :style="model ? { backgroundColor: model } : undefined"
        class="border-default flex size-8 shrink-0 items-center justify-center rounded-md border"
        :class="highlight ? 'ring-warning ring-2' : ''"
        :aria-label="t('common.pickColor')"
      >
        <UIcon v-if="!model" name="i-lucide-pipette" class="text-muted size-4" />
      </button>

      <template #content>
        <UColorPicker :model-value="model" class="p-3" @update:model-value="updateColor" />
      </template>
    </UPopover>

    <UInput
      v-model="model"
      :color="color"
      :highlight="highlight"
      :placeholder="placeholder"
      class="min-w-0 flex-1 font-mono"
    />
  </div>
</template>
