<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { ThemeSettings } from '@/api/types';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import ColorField from '@/components/form/ColorField.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import { useConfigForm } from '@/composables/useConfigForm';

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
  <UCard variant="subtle" class="mt-4">
    <div class="flex flex-col gap-4">
      <div class="text-muted flex items-center gap-1.5 text-sm">
        <span>{{ t('pages.config.theme.title') }}</span>
        <HelpHint :text="t('pages.config.theme.hint')" />
      </div>
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
          <ColorField v-model="state.theme[key]" v-bind="fieldProps(`theme.${key}`)" />
        </UFormField>
      </div>
    </div>
  </UCard>
</template>
