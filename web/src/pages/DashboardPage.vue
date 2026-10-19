<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { presetRepository } from '@/api/repositories/preset';
import { useApiError } from '@/composables/useApiError';
import { useHeatLabel } from '@/composables/useHeatLabel';
import OverlayLinks, { type OverlayLink } from '@/components/overlay/OverlayLinks.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import TimingStatusBadge from '@/components/timing/TimingStatusBadge.vue';
import { useLiveStore } from '@/stores/live';
import { useTimingStore } from '@/stores/timing';

const { t } = useI18n();
const router = useRouter();
const { messageFor } = useApiError();
const { heatLabel: formatHeat } = useHeatLabel();

const liveStore = useLiveStore();
const timingStore = useTimingStore();

const presetNames = ref<string[]>([]);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

onMounted(async () => {
  liveStore.connect();
  timingStore.start();

  try {
    presetNames.value = await presetRepository.list();
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  }
});

onUnmounted(() => {
  timingStore.stop();
});

// The event, name and heat of the live heat, or null before the first frame.
// An event missing from the meet data has no name, which is left out.
const heatLabel = computed(() => {
  const state = liveStore.state;
  if (!state || !state.status) return null;

  return [
    t('overlays.event', {
      number: state.event,
    }),
    state.name.trim(),
    formatHeat(state.heat, state.heatCount),
  ]
    .filter(Boolean)
    .join(' · ');
});

const overlayLinks = computed<OverlayLink[]>(() => [
  {
    label: t('overlays.info'),
    path: '/overlays/info',
  },
  {
    label: t('overlays.splits'),
    path: '/overlays/splits',
  },
  {
    label: t('overlays.startList'),
    path: '/overlays/startlist',
  },
  {
    label: t('overlays.timer'),
    path: '/overlays/timer',
  },
  {
    label: t('pages.scoreboard.title'),
    path: router.resolve({
      name: 'scoreboard',
    }).href,
    wide: true,
  },
]);

const arLinks = computed<OverlayLink[]>(() =>
  presetNames.value.map((preset) => ({
    label: t('overlays.ar', {
      preset,
    }),
    path: `/overlays/ar/${encodeURIComponent(preset)}`,
  })),
);

// The lanes of the live heat, which change with every loaded heat.
const laneLinks = computed<OverlayLink[]>(() => {
  if (!liveStore.state) return [];
  return Object.values(liveStore.state.lanes)
    .sort((a, b) => a.number - b.number)
    .map((lane) => ({
      label: t('overlays.lane', {
        number: lane.number,
      }),
      path: `/overlays/lane/${lane.number}`,
    }));
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-1">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-1">
          <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.dashboard.title') }}</h1>
          <DocsLink page="dashboard" />
        </div>
        <TimingStatusBadge />
      </div>

      <p class="text-muted flex items-center gap-2 text-sm">
        <UIcon name="i-lucide-waves" class="size-4 shrink-0" />
        <span>{{ heatLabel ?? t('pages.dashboard.noHeat') }}</span>
      </p>
    </div>

    <UAlert
      v-if="loadFailed"
      :title="t('pages.dashboard.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <OverlayLinks
      :title="t('pages.dashboard.overlays')"
      :help="t('pages.dashboard.help.overlays')"
      icon="i-lucide-layers"
      :links="overlayLinks"
    />

    <OverlayLinks
      v-if="arLinks.length"
      :title="t('pages.dashboard.arOverlays')"
      :help="t('pages.dashboard.help.arOverlays')"
      icon="i-lucide-scan"
      :links="arLinks"
    />

    <OverlayLinks
      v-if="laneLinks.length"
      :title="t('pages.dashboard.lanes')"
      :help="t('pages.dashboard.help.lanes')"
      icon="i-lucide-rows-3"
      :links="laneLinks"
    />
  </div>
</template>
