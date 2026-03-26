<script setup lang="ts">
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import { ArrowRight } from 'lucide-vue-next';

import { useRaceStore } from '@/stores/raceStore.ts';

const store = useRaceStore();

onMounted(() => {
  store.connect();
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
