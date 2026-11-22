<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { ThemeSettings } from '@/api/types/config';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import ColorField from '@/components/form/ColorField.vue';
import { useConfigForm } from '@/composables/config/useConfigForm';

const THEME_COLOR_KEYS: (keyof ThemeSettings)[] = [
  'primary',
  'secondary',
  'success',
  'info',
  'warning',
  'error',
  'neutral',
];

// The colors of the admin pages. An empty color keeps the default.
const { t } = useI18n();
const { state, fieldProps } = useConfigForm();
</script>

<template>
  <UCard variant="subtle">
    <div class="flex flex-col gap-4">
      <h3 class="text-highlighted heading-card">{{ t('pages.config.theme.title') }}</h3>
      <div class="grid grid-cols-1 gap-6 sm:grid-cols-4">
        <UFormField
          v-for="key in THEME_COLOR_KEYS"
          :key="key"
          :label="t(`fields.theme.${key}`)"
          :name="`theme.${key}`"
        >
          <template #hint>
            <ConfigFieldHint :path="`theme.${key}`" />
          </template>
          <ColorField
            v-model="state.theme[key]"
            v-bind="fieldProps(`theme.${key}`)"
            :placeholder="t('pages.config.theme.default')"
          />
        </UFormField>
      </div>
    </div>
  </UCard>
</template>
