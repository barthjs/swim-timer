<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

// A UInput bound to type="password" with a trailing toggle to reveal the value in plain text.
const model = defineModel<string>({
  required: true,
});

withDefaults(
  defineProps<{
    color?: 'warning';
    highlight?: boolean;
    maxlength?: number;
    placeholder?: string;
  }>(),
  {
    color: undefined,
    highlight: false,
    maxlength: undefined,
    placeholder: undefined,
  },
);

const { t } = useI18n();
const visible = ref(false);
</script>

<template>
  <UInput
    v-model="model"
    :type="visible ? 'text' : 'password'"
    :color="color"
    :highlight="highlight"
    :maxlength="maxlength"
    :placeholder="placeholder"
    class="w-full"
  >
    <template #trailing>
      <UButton
        :icon="visible ? 'i-lucide-eye-off' : 'i-lucide-eye'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="visible ? t('common.hidePassword') : t('common.showPassword')"
        @click="visible = !visible"
      />
    </template>
  </UInput>
</template>
