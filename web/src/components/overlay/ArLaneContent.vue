<script setup lang="ts">
import { computed } from 'vue';

import { TextAnchor, type Lane } from '@/api/types';
import { isResultRanked } from '@/lib/heat';
import { LANE_CONTENT_HEIGHT } from '@/lib/lanes';
import { formatName } from '@/lib/names';
import { formatSwimtime } from '@/lib/time';

// The content of one lane in the AR overlay. It fills the content box of the lane, which the lane transform
// projects into the camera image, see LANE_CONTENT_WIDTH. All sizes follow the height of the box,
// so the text size of the preset is the height of the text in percent of the lane height.
const props = defineProps<{
  lane: Pick<Lane, 'number' | 'rank' | 'firstName' | 'lastName' | 'time' | 'finished'>;
  textAnchor: TextAnchor;
  // The height of the text in percent of the height of the lane.
  textSize: number;
}>();

const sizes = computed(() => ({
  '--lane-height': `${LANE_CONTENT_HEIGHT}px`,
  '--lane-text-size': `${(LANE_CONTENT_HEIGHT * props.textSize) / 100}px`,
}));
</script>

<template>
  <div
    class="content"
    :class="textAnchor === TextAnchor.Right ? 'anchor-right' : 'anchor-left'"
    :style="sizes"
  >
    <div v-if="isResultRanked(lane)" class="badge rank">
      {{ lane.rank }}
    </div>

    <div v-else class="badge">
      {{ lane.number }}
    </div>

    <div class="name">
      {{ formatName(lane.firstName, lane.lastName) }}
    </div>

    <div v-if="lane.finished" class="time">
      {{ formatSwimtime(lane.time) }}
    </div>
  </div>
</template>

<style scoped>
.content {
  display: flex;
  align-items: center;
  gap: calc(var(--lane-height) * 0.2);
  width: 100%;
  height: 100%;
  background: var(--overlay-row);
  color: var(--overlay-secondary);
  font-family: var(--overlay-font-sans);
  font-weight: 700;
  font-size: var(--lane-text-size);
  line-height: 1;
}

.content.anchor-left {
  padding-right: calc(var(--lane-height) * 0.2);
  transform-origin: left;
}

.content.anchor-right {
  flex-direction: row-reverse;
  padding-left: calc(var(--lane-height) * 0.2);
  transform-origin: right;
}

.badge {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: calc(var(--lane-height) * 0.8);
  height: 100%;
}

.badge.rank {
  background: var(--overlay-accent);
  color: var(--overlay-ink);
}

.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-transform: uppercase;
}

.time {
  flex-shrink: 0;
  letter-spacing: 0.025em;
  font-variant-numeric: tabular-nums;
}

.anchor-left .time {
  margin-left: auto;
  padding-right: calc(var(--lane-height) * 0.2);
}

.anchor-right .time {
  margin-right: auto;
  padding-left: calc(var(--lane-height) * 0.2);
}
</style>
