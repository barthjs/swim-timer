<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import '@/assets/css/overlay-tokens.css';
import ArLaneContent from '@/components/overlay/ArLaneContent.vue';
import { heatLaneNumber, laneStyle, presetLaneCount, type EditorPreset } from '@/lib/lanes';

// Shows every lane as the AR overlay draws it, with sample values, so the result can be judged while editing.
// Each lane shows the lane of the heat the AR overlay maps to it.
const props = defineProps<{
  preset: EditorPreset;
  firstLane: number;
}>();

const { t } = useI18n();

// Sample finish times, which differ slightly from lane to lane.
const SAMPLE_TIME_MS = 61_230;
const SAMPLE_TIME_STEP_MS = 870;

const lanes = computed(() => {
  const laneCount = presetLaneCount(props.preset.lanes);

  return Object.entries(props.preset.lanes).map(([key, presetLane]) => {
    const heatLane = heatLaneNumber(
      Number(key),
      laneCount,
      props.firstLane,
      props.preset.reverseLanes,
    );
    // The sample ranks count from 1, whatever the pool numbering
    const rank = heatLane - props.firstLane + 1;

    return {
      key,
      style: laneStyle(presetLane),
      lane: {
        number: heatLane,
        rank,
        firstName: t('pages.editor.previewFirstName'),
        lastName: t('pages.editor.previewLastName', {
          lane: heatLane,
        }),
        time: SAMPLE_TIME_MS + rank * SAMPLE_TIME_STEP_MS,
        finished: true,
      },
    };
  });
});
</script>

<template>
  <div class="pointer-events-none absolute inset-0 overflow-hidden">
    <div v-for="item in lanes" :key="item.key" :style="item.style">
      <ArLaneContent
        :lane="item.lane"
        :text-anchor="preset.textAnchor"
        :text-size="preset.textSize"
      />
    </div>
  </div>
</template>
