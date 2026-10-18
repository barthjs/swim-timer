<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { LiveHeat } from '@/api/types';
import { useHeatLabel } from '@/composables/useHeatLabel';
import { finalPhaseName } from '@/lib/heat';

// Names the live heat in one line, with dividers sized for the surrounding text.
// With showFinal, a final also names its phase.
const props = defineProps<{
  state: LiveHeat;
  size: 'small' | 'large';
  showFinal?: boolean;
}>();

const { t } = useI18n();
const { heatLabel } = useHeatLabel();

// An event missing from the meet data has no name, which is left out together with its divider
const name = computed(() => props.state.name.trim());
const phaseName = computed(() => (props.showFinal ? finalPhaseName(props.state) : ''));
</script>

<template>
  <span class="accent">
    {{ t('overlays.event', { number: state.event }) }}
  </span>
  <template v-if="name">
    <span class="divider" :class="size"></span>
    <span>{{ name }}</span>
  </template>
  <template v-if="phaseName">
    <span class="divider" :class="size"></span>
    <span>{{ phaseName }}</span>
  </template>
  <span class="divider" :class="size"></span>
  <span>
    {{ heatLabel(state.heat, state.heatCount) }}
  </span>
</template>

<style scoped>
.accent {
  color: var(--overlay-accent);
}

.divider {
  background: var(--overlay-divider);
}

.divider.small {
  width: 0.125rem;
  height: 1.25rem;
}

.divider.large {
  width: 0.25rem;
  height: 1.75rem;
}
</style>
