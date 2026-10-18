<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import HelpHint from '@/components/form/HelpHint.vue';
import { useTimingStore } from '@/stores/timing';

// The connection to the timing device as a short label, with the details on demand.
const { t } = useI18n();
const timingStore = useTimingStore();

const label = computed(() => {
  const status = timingStore.status;
  if (!status) return t('pages.dashboard.timing.connected');

  if (status.connected && status.replayFile && status.simulateEvent > 0) {
    return t('pages.dashboard.timing.replayedEvent', {
      file: status.replayFile,
      event: status.simulateEvent,
    });
  }
  if (status.connected && status.replayFile) {
    return t('pages.dashboard.timing.replayed', {
      file: status.replayFile,
    });
  }
  if (status.connected && status.simulateEvent > 0) {
    return t('pages.dashboard.timing.simulated', {
      event: status.simulateEvent,
    });
  }
  if (status.connected) {
    return t('pages.dashboard.timing.connectedPort', {
      port: status.port,
    });
  }
  if (status.configuredPort) {
    return t('pages.dashboard.timing.disconnectedPort', {
      port: status.configuredPort,
    });
  }
  return t('pages.dashboard.timing.disconnected');
});

const description = computed(() => {
  const status = timingStore.status;
  if (!status) return '';

  if (status.connected && status.replayFile && status.simulateEvent > 0) {
    return t('pages.dashboard.timing.replayedEventHint', {
      file: status.replayFile,
      event: status.simulateEvent,
      port: status.port,
    });
  }
  if (status.connected && status.replayFile) {
    return t('pages.dashboard.timing.replayedHint', {
      file: status.replayFile,
      port: status.port,
    });
  }
  if (status.connected && status.simulateEvent > 0) {
    return t('pages.dashboard.timing.simulatedHint', {
      event: status.simulateEvent,
      port: status.port,
    });
  }
  if (status.connected) {
    return t('pages.dashboard.timing.connectedHint', {
      port: status.port,
    });
  }
  if (status.configuredPort) {
    return t('pages.dashboard.timing.disconnectedPortHint', {
      port: status.configuredPort,
    });
  }
  return t('pages.dashboard.timing.disconnectedHint');
});
</script>

<template>
  <div class="flex max-w-full min-w-0 items-center gap-1.5">
    <UBadge
      :label="label"
      :icon="timingStore.connected ? 'i-lucide-plug' : 'i-lucide-unplug'"
      :color="timingStore.connected ? 'success' : 'error'"
      variant="solid"
      size="lg"
      class="max-w-full min-w-0 truncate"
    />
    <HelpHint v-if="description" :text="description" />
  </div>
</template>
