<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { fullName } from '@/lib/live/names';
import type { RecordBanner } from '@/lib/record/recordBanner';
import { formatSwimtime } from '@/lib/live/time';

// A record a lane broke or equalled, as the info overlay announces it: the kind and the previous time of the record
// above the swimmer and the time, both rows as wide as the longer one.
const props = defineProps<{
  banner: RecordBanner;
}>();

const { t } = useI18n();

const label = computed(() =>
  props.banner.record.equalled ? t('overlays.records.equalled') : t('overlays.records.new'),
);
</script>

<template>
  <div class="banner">
    <div class="segment head">
      <span class="truncate">{{ label }}</span>
      <span class="previous">
        {{ t('overlays.records.previous', { time: formatSwimtime(banner.record.previousTime) }) }}
      </span>
    </div>

    <div class="segment body">
      <div class="name">
        <span class="truncate">{{ fullName(banner.lane.firstName, banner.lane.lastName) }}</span>
        <span v-if="banner.lane.birthYear" class="year">({{ banner.lane.birthYear }})</span>
      </div>
      <div class="time">{{ formatSwimtime(banner.lane.time) }}</div>
    </div>
  </div>
</template>

<style scoped>
/* The rows take the height and the corners of the bars of the info overlay, and every text one size and weight */
.banner {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-width: 50vw;
  font-size: 1.5rem;
  font-weight: 700;
  text-transform: uppercase;
}

.segment {
  display: flex;
  align-items: center;
  height: 3.5rem;
  overflow: hidden;
  border-radius: 0.375rem;
}

.head {
  justify-content: space-between;
  gap: 2rem;
  padding: 0 1.5rem;
  background: var(--overlay-accent);
  color: var(--overlay-ink);
}

.previous {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.body {
  background: var(--overlay-bar);
}

.name {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  padding: 0 1.5rem;
}

.year {
  flex-shrink: 0;
}

.time {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  align-self: stretch;
  padding: 0 1.5rem;
  background: var(--overlay-primary);
  font-variant-numeric: tabular-nums;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
