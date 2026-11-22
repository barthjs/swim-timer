<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { ConfigSettings } from '@/api/types/config';
import HelpHint from '@/components/form/HelpHint.vue';
import { useConfigForm } from '@/composables/config/useConfigForm';
import type { Paths } from '@/lib/form/path';

// The hint slot of a configuration field: an optional character count, a warning while a flag
// or an environment variable overrides the config file, an optional explanation, and a button that resets the field
// to the default its tooltip names. The button always shows, so it keeps its place, and is disabled at the default.
const props = defineProps<{
  path: Paths<ConfigSettings>;
  help?: string;
  count?: string;
  // The choices of a select, so the default shows with the label the select shows for it
  options?: { label: string; value: unknown }[];
}>();

const { t } = useI18n();
const { sourceOf, isOverridden, defaultOf, isSecret, isDefault, resetField } = useConfigForm();

const source = computed(() => sourceOf(props.path));
const overridden = computed(() => isOverridden(props.path));

const sourceLabel = computed(() => {
  switch (source.value?.kind) {
    case 'env':
      return t('pages.config.source.env', {
        name: source.value.name,
      });
    case 'flag':
      return t('pages.config.source.flag', {
        name: source.value.name,
      });
    default:
      return '';
  }
});

const defaultLabel = computed(() => {
  const value = defaultOf(props.path);
  const option = props.options?.find((o) => o.value === value);
  if (option) {
    return option.label;
  }
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(', ') : t('pages.config.source.empty');
  }
  if (typeof value === 'boolean') {
    return value ? t('pages.config.source.on') : t('pages.config.source.off');
  }
  if (value === undefined || value === null || value === '') {
    return t('pages.config.source.empty');
  }
  return typeof value === 'string' || typeof value === 'number'
    ? String(value)
    : JSON.stringify(value);
});

// A secret offers its own button to remove the saved value, which is what a reset does
const resettable = computed(() => !isSecret(props.path));
</script>

<template>
  <span class="flex items-center gap-1.5">
    <span v-if="count">{{ count }}</span>

    <UPopover
      v-if="overridden && source"
      mode="hover"
      enable-touch
      :open-delay="100"
      :close-delay="150"
      :content="{ side: 'right', align: 'start' }"
    >
      <UButton
        icon="i-lucide-triangle-alert"
        :label="t(`pages.config.source.badge.${source.kind}`)"
        color="warning"
        variant="link"
        size="xs"
        class="p-0"
        :aria-label="sourceLabel"
      />

      <template #content>
        <div class="text-default flex max-w-72 flex-col gap-2 p-3 text-sm">
          <p>{{ sourceLabel }}</p>
          <p class="text-warning">{{ t('pages.config.source.overridden') }}</p>
        </div>
      </template>
    </UPopover>

    <HelpHint v-if="help" :text="help" />

    <UTooltip
      v-if="resettable"
      :text="
        t('pages.config.source.resetTo', {
          value: defaultLabel,
        })
      "
    >
      <UButton
        icon="i-lucide-rotate-ccw"
        color="neutral"
        variant="link"
        size="xs"
        class="p-0 disabled:opacity-30"
        :disabled="isDefault(path)"
        :aria-label="
          t('pages.config.source.resetTo', {
            value: defaultLabel,
          })
        "
        @click="resetField(path)"
      />
    </UTooltip>
  </span>
</template>
