<script setup lang="ts" generic="T extends string">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { BadgeProps, TabsItem } from '@nuxt/ui';

/** A section of a page with its icon, an optional count, and its state. */
export type Section<T extends string> = {
  value: T;
  label: string;
  icon: string;
  count?: number;
  // The section holds unsaved changes
  changed?: boolean;
  // The number of errors in the section, which replaces the count
  errors?: number;
};

// The sections of a page as tabs, or as a select on narrow screens where the tabs do not fit.
// The page renders the open section below the tabs, or next to them when they stand vertical.
const props = withDefaults(
  defineProps<{
    sections: Section<T>[];
    // Pills sit next to a heading, a link bar spans the width above the content
    variant?: 'link' | 'pill';
    // Compact pills take the height of a field, so they fit in the row of a heading
    compact?: boolean;
    centered?: boolean;
    // Vertical tabs stand next to the content
    orientation?: 'horizontal' | 'vertical';
  }>(),
  {
    variant: 'link',
    compact: false,
    centered: false,
    orientation: 'horizontal',
  },
);

const active = defineModel<T>({
  required: true,
});

const { t } = useI18n();

const items = computed<TabsItem[]>(() =>
  props.sections.map((section) => ({
    label: section.label,
    icon: section.icon,
    value: section.value,
    badge: sectionBadge(section),
  })),
);

/** Opens the section the tabs or the select chose, both work on plain values. */
const onSelect = (value: string | number | undefined) => {
  const section = props.sections.find((item) => item.value === value);
  if (section) active.value = section.value;
};

/** Returns the badge of a section: its errors, a dot for unsaved changes, or its count. */
function sectionBadge(section: Section<T>): BadgeProps | undefined {
  if ((section.errors ?? 0) > 0) {
    return {
      label: String(section.errors),
      color: 'error',
      variant: 'solid',
    };
  }
  if (section.changed) {
    return {
      color: 'warning',
      variant: 'solid',
      class: 'size-2 rounded-full p-0',
    };
  }
  if ((section.count ?? 0) > 0) {
    return {
      label: String(section.count),
      color: 'neutral',
      variant: 'subtle',
    };
  }
  return undefined;
}
</script>

<template>
  <div class="min-w-0" :class="orientation === 'vertical' ? 'shrink-0 md:w-48' : ''">
    <USelect
      :model-value="active as string"
      :items="items"
      value-key="value"
      :aria-label="t('common.sections')"
      class="w-full md:hidden"
      @update:model-value="onSelect"
    />
    <UTabs
      :model-value="active as string"
      :items="items"
      :content="false"
      :variant="variant"
      :orientation="orientation"
      :aria-label="t('common.sections')"
      :ui="{
        list: [centered ? 'justify-center' : '', variant === 'pill' && compact ? 'p-0.5' : ''],
        trigger: variant === 'pill' && compact ? 'py-1' : '',
      }"
      class="hidden md:flex"
      @update:model-value="onSelect"
    />
  </div>
</template>
