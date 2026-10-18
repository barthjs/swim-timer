<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { ConfigSettings } from '@/api/types';
import HelpHint from '@/components/form/HelpHint.vue';
import { useConfigForm } from '@/composables/useConfigForm';
import type { Paths } from '@/lib/form/path';

// The hint slot of a configuration field: an optional character count, where the value comes from with its default,
// a button that resets the field to its default, and an optional explanation. A flag or an environment variable
// overrides the config file, so such a source shows as a warning.
const props = defineProps<{
  path: Paths<ConfigSettings>;
  help?: string;
  count?: string;
  // The choices of a select, so the default shows with the label the select shows for it.
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
    case 'file':
      return t('pages.config.source.file');
    default:
      return t('pages.config.source.default');
  }
});

const sourceIcon = computed(() => {
  switch (source.value?.kind) {
    case 'env':
    case 'flag':
      return 'i-lucide-triangle-alert';
    case 'file':
      return 'i-lucide-file-text';
    default:
      return 'i-lucide-circle-dashed';
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
  return String(value);
});

// A secret offers its own button to remove the saved value, which is what a reset does
const resettable = computed(() => !isSecret(props.path) && !isDefault(props.path));
</script>

<template>
  <span class="flex items-center gap-1.5">
    <span v-if="count">{{ count }}</span>

    <UPopover
      v-if="source"
      mode="hover"
      enable-touch
      :open-delay="100"
      :close-delay="150"
      :content="{ side: 'right', align: 'start' }"
    >
      <UButton
        :icon="sourceIcon"
        :label="overridden ? t(`pages.config.source.badge.${source.kind}`) : undefined"
        :color="overridden ? 'warning' : 'neutral'"
        variant="link"
        size="xs"
        class="p-0"
        :aria-label="sourceLabel"
      />

      <template #content>
        <div class="text-default flex max-w-72 flex-col gap-2 p-3 text-sm">
          <p>
            <span class="text-muted">{{ t('pages.config.source.title') }}:</span>{{ ' ' }}
            {{ sourceLabel }}
          </p>
          <p>
            <span class="text-muted">{{ t('pages.config.source.defaultValue') }}:</span>{{ ' ' }}
            <span class="font-mono break-all">{{ defaultLabel }}</span>
          </p>
          <p v-if="overridden" class="text-warning">
            {{ t('pages.config.source.overridden') }}
          </p>
        </div>
      </template>
    </UPopover>

    <UTooltip v-if="resettable" :text="t('pages.config.source.reset')">
      <UButton
        icon="i-lucide-rotate-ccw"
        color="neutral"
        variant="link"
        size="xs"
        class="p-0"
        :aria-label="t('pages.config.source.reset')"
        @click="resetField(path)"
      />
    </UTooltip>

    <HelpHint v-if="help" :text="help" />
  </span>
</template>
