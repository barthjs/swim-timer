<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ButtonProps } from '@nuxt/ui';

// The empty state of a table. A list without entries shows its own icon and text,
// a list whose search or filters match nothing says so and offers to reset them.
const props = defineProps<{
  icon: string;
  // Shown while no search or filter is set
  text: string;
  filtered?: boolean;
}>();

const emit = defineEmits<{ reset: [] }>();

const { t } = useI18n();

const actions = computed<ButtonProps[]>(() =>
  props.filtered
    ? [
        {
          label: t('table.resetAll'),
          icon: 'i-lucide-funnel-x',
          color: 'neutral',
          variant: 'outline',
          onClick: () => emit('reset'),
        },
      ]
    : [],
);
</script>

<template>
  <UEmpty
    :icon="filtered ? 'i-lucide-search-x' : icon"
    :title="filtered ? t('table.noResults') : text"
    :actions="actions"
    variant="naked"
    size="sm"
  />
</template>
