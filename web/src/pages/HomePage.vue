<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';
import { ArrowRight } from 'lucide-vue-next';

import { presetsApi } from '@/api/presetsApi.ts';

import { useRaceStore } from '@/stores/raceStore.ts';
import { useToastStore } from '@/stores/toastStore.ts';

import { getErrorMessageKey, isApiError } from '@/types/api.ts';

const presetNames = ref<string[]>([]);

const { t } = useI18n();

const store = useRaceStore();
const toastStore = useToastStore();

const loadPresetNames = async (): Promise<void> => {
  try {
    presetNames.value = await presetsApi.getPresetNames();
  } catch (err) {
    const message = isApiError(err) ? t(getErrorMessageKey(err)) : t('home.loadPresetsError');
    toastStore.error(message, 15000);
    console.error(err);
  }
};

onMounted(() => {
  store.connect();
  loadPresetNames();
});
</script>

<template>
  <div class="m-auto flex w-full max-w-2xl flex-col space-y-6">
    <section class="bg-surface flex flex-col space-y-4 rounded-xl p-5">
      <h2 class="border-subtle border-b pb-1 font-bold tracking-wider uppercase">
        {{ $t('home.configuration') }}
      </h2>

      <div class="grid grid-cols-2 gap-2">
        <RouterLink
          :to="{ name: 'config' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('config.title') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'editor' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('editor.title') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'race-data' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('raceData.title') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'race-overview' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('raceOverview.title') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>
      </div>
    </section>

    <section class="bg-surface flex flex-col space-y-4 rounded-xl p-5">
      <h2 class="border-subtle border-b pb-1 font-bold tracking-wider uppercase">
        {{ $t('home.overlays') }}
      </h2>

      <div class="grid grid-cols-2 gap-2">
        <RouterLink
          :to="{ name: 'overlay-info' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('overlays.info') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'overlay-splits' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('overlays.splits') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'overlay-startlist' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('overlays.startlist') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>

        <RouterLink
          :to="{ name: 'overlay-timer' }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('overlays.timer') }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>
      </div>
    </section>

    <section
      v-if="presetNames.length > 0"
      class="bg-surface flex flex-col space-y-4 rounded-xl p-5"
    >
      <h2 class="border-subtle border-b pb-1 font-bold tracking-wider uppercase">
        {{ $t('home.arOverlays') }}
      </h2>

      <div class="grid grid-cols-2 gap-2">
        <RouterLink
          v-for="preset in presetNames"
          :key="preset"
          :to="{ name: 'overlay-ar', params: { preset: preset } }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ preset }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>
      </div>
    </section>

    <section
      v-if="store.state && Object.keys(store.state.lanes).length > 0"
      class="bg-surface flex flex-col space-y-4 rounded-xl p-5"
    >
      <h2 class="border-subtle border-b pb-1 font-bold tracking-wider uppercase">
        {{ $t('home.lanes') }}
      </h2>

      <div class="grid grid-cols-2 gap-2">
        <RouterLink
          v-for="lane in store.state.lanes"
          :key="lane.number"
          :to="{ name: 'overlay-lane', params: { number: lane.number } }"
          class="border-subtle hover:bg-base flex items-center justify-between rounded-md border px-3 py-2 transition-colors"
        >
          <span>{{ $t('overlays.lane', { number: lane.number }) }}</span>
          <ArrowRight :size="18" class="text-muted" />
        </RouterLink>
      </div>
    </section>
  </div>
</template>
