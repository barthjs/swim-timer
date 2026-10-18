<script setup lang="ts">
import { useI18n } from 'vue-i18n';

// A swatch that opens the color picker in a popover, next to a hex text input bound to the same model.
// The saved value is always the visible one, and a color can be typed or pasted as well.
const { t } = useI18n();

const model = defineModel<string>({
  required: true,
});

withDefaults(
  defineProps<{
    color?: 'warning';
    highlight?: boolean;
  }>(),
  {
    color: undefined,
    highlight: false,
  },
);
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
        <UColorPicker v-model="model" class="p-3" />
      </template>
    </UPopover>

    <UInput
      v-model="model"
      :color="color"
      :highlight="highlight"
      placeholder="#0ea5e9"
      class="w-32 font-mono"
    />
  </div>
</template>
