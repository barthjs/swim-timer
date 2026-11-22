<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useConfigForm } from '@/composables/config/useConfigForm';

// A button that resets the settings of a section, or all settings without one, to their defaults.
// It only fills the form, so the reset takes effect once the configuration is saved.
// Saved secrets are marked to be removed, since a secret has no default.
const props = defineProps<{
  // The top-level keys of the configuration the section edits, such as "app" and "log"
  sections?: string[];
  // The name of the section
  label?: string;
}>();

const { t } = useI18n();
const { isSectionDefault, resetSections } = useConfigForm();

const text = computed(() =>
  props.sections
    ? t('pages.config.source.resetSection', {
        section: props.label ?? '',
      })
    : t('pages.config.source.resetAll'),
);
</script>

<template>
  <UTooltip :text="text">
    <UButton
      :label="sections ? undefined : t('pages.config.source.defaults')"
      :aria-label="text"
      icon="i-lucide-list-restart"
      color="neutral"
      :variant="sections ? 'ghost' : 'outline'"
      :disabled="isSectionDefault(sections)"
      :ui="{ label: 'hidden sm:inline' }"
      @click="resetSections(sections)"
    />
  </UTooltip>
</template>
