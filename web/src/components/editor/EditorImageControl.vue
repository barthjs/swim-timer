<script setup lang="ts">
import { useI18n } from 'vue-i18n';

// The background image of an editor in the toolbar. Without an image, the button opens the file picker.
// With one, it opens a menu with the opacity, another image, and removing it.
const opacity = defineModel<number>('opacity', {
  required: true,
});

defineProps<{
  // What the editor calls its image, such as a background image.
  label: string;
  hasImage: boolean;
  // The key labels of the shortcut that picks an image.
  kbds: string[];
}>();

const emit = defineEmits<{
  pick: [];
  clear: [];
}>();

const { t } = useI18n();
</script>

<template>
  <UPopover v-if="hasImage">
    <UTooltip :text="label" :kbds="kbds">
      <UButton icon="i-lucide-image" size="sm" color="primary" variant="soft" :aria-label="label" />
    </UTooltip>

    <template #content>
      <div class="flex w-64 flex-col gap-3 p-3">
        <UFormField :label="t('editor.image.opacity')">
          <USlider v-model="opacity" :min="0.1" :max="1" :step="0.05" />
        </UFormField>
        <div class="flex gap-2">
          <UButton
            :label="t('editor.image.replace')"
            icon="i-lucide-image-up"
            size="sm"
            color="neutral"
            variant="outline"
            @click="emit('pick')"
          />
          <UButton
            :label="t('editor.image.clear')"
            icon="i-lucide-image-off"
            size="sm"
            color="error"
            variant="outline"
            @click="emit('clear')"
          />
        </div>
      </div>
    </template>
  </UPopover>
  <UTooltip v-else :text="label" :kbds="kbds">
    <UButton
      icon="i-lucide-image"
      size="sm"
      color="neutral"
      variant="ghost"
      :aria-label="label"
      @click="emit('pick')"
    />
  </UTooltip>
</template>
