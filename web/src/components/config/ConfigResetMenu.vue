<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import { useConfigForm } from '@/composables/useConfigForm';

// A menu that resets the settings of the open section, or all settings, to their defaults.
// It only fills the form, so the reset takes effect once the configuration is saved.
// Saved secrets are marked to be removed, since a secret has no default.
const props = defineProps<{
  // The sections of the open tab, such as "app" and "log".
  sections: string[];
  // The name of the open tab.
  label: string;
}>();

const { t } = useI18n();
const { isSectionDefault, resetSections } = useConfigForm();

const items = computed<DropdownMenuItem[]>(() => [
  {
    label: t('pages.config.source.resetSection', {
      section: props.label,
    }),
    icon: 'i-lucide-rotate-ccw',
    disabled: isSectionDefault(props.sections),
    onSelect: () => resetSections(props.sections),
  },
  {
    label: t('pages.config.source.resetAll'),
    icon: 'i-lucide-list-restart',
    disabled: isSectionDefault(),
    onSelect: () => resetSections(),
  },
]);
</script>

<template>
  <UDropdownMenu :items="items" :content="{ align: 'end' }">
    <UButton
      :label="t('pages.config.source.defaults')"
      :aria-label="t('pages.config.source.defaults')"
      icon="i-lucide-list-restart"
      color="neutral"
      variant="outline"
      :ui="{ label: 'hidden sm:inline' }"
    />
  </UDropdownMenu>
</template>
